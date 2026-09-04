// Command thugsflooder is a scoped network/log load-generation tool for
// authorized red-team/blue-team exercises. See internal/about for the
// full notice — it is the only output produced without
// --i-understand-the-risk.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thugsred/thugsflooder/internal/about"
	"github.com/thugsred/thugsflooder/internal/audit"
	"github.com/thugsred/thugsflooder/internal/config"
	"github.com/thugsred/thugsflooder/internal/generator"
	"github.com/thugsred/thugsflooder/internal/logbuf"
	"github.com/thugsred/thugsflooder/internal/recording"
	"github.com/thugsred/thugsflooder/internal/stats"
	"github.com/thugsred/thugsflooder/internal/tui"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		about.PrintGate()
		os.Exit(0)
	}
	if args[0] == "--version" {
		about.PrintVersion()
		os.Exit(0)
	}

	switch args[0] {
	case "gen-config":
		runGenConfig(args[1:])
	case "run":
		runFlood(args[1:])
	default:
		about.PrintGate()
		os.Exit(1)
	}
}

func runGenConfig(args []string) {
	path := "targets.yaml"
	if len(args) > 0 {
		path = args[0]
	}
	if err := config.WriteExample(path); err != nil {
		fmt.Fprintln(os.Stderr, "thugsflooder: gen-config:", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote example allowlist config to %s\n", path)
}

func runFlood(args []string) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // help/usage is always about.PrintGate, not flag's default

	configPath := fs.String("config", "", "allowlist config file (required)")
	riskFlag := fs.Bool("i-understand-the-risk", false, "required to run any traffic generation")
	headless := fs.Bool("headless", false, "run without the TUI, printing periodic stats instead")
	fs.Bool("tui", false, "run with the live TUI dashboard (default)")
	maxDuration := fs.Duration("max-duration", 0, "stop automatically after this duration (0 = unbounded)")
	replayPath := fs.String("replay", "", "recording file (JSON lines) to replay alongside the configured generators")
	speed := fs.Float64("speed", 1.0, "replay speed multiplier")

	if err := fs.Parse(args); err != nil {
		about.PrintGate()
		os.Exit(1)
	}

	if !*riskFlag {
		about.PrintGate()
		os.Exit(1)
	}
	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "thugsflooder: run: --config is required")
		os.Exit(1)
	}

	cfg, allow, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "thugsflooder: config error:", err)
		os.Exit(1)
	}

	auditLogger, err := audit.Open(cfg.AuditLog)
	if err != nil {
		fmt.Fprintln(os.Stderr, "thugsflooder: audit log error:", err)
		os.Exit(1)
	}
	defer auditLogger.Close()

	logs := logbuf.New(1000)
	st := stats.New()
	sinks := &generator.Sinks{Stats: st, Audit: auditLogger, Log: logs.Logf}

	mgr := generator.NewManager(cfg, allow, sinks)

	if *replayPath != "" {
		entries, err := recording.Load(*replayPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "thugsflooder: replay error:", err)
			os.Exit(1)
		}
		mgr = mgr.WithReplay(entries, *speed)
		logs.Logf("loaded %d replay entries from %s", len(entries), *replayPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *maxDuration > 0 {
		go func() {
			select {
			case <-time.After(*maxDuration):
				logs.Logf("max-duration elapsed, stopping")
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// Roll the per-second rate window into history for the bar graph,
	// independent of TUI/headless mode.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				st.Tick()
			case <-ctx.Done():
				return
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		mgr.Run(ctx)
		close(done)
	}()

	logs.Logf("thugsflooder started: session=%s targets=%v audit=%s", mgr.Session(), allow.Hosts(), cfg.AuditLog)

	if *headless {
		runHeadless(ctx, st)
	} else {
		runTUI(ctx, st, logs, cfg.AuditLog, cancel)
	}

	<-done
	printSummary(st.Snapshot(), cfg.AuditLog)
}

func runHeadless(ctx context.Context, st *stats.Stats) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s := st.Snapshot()
			fmt.Printf("[%s] sent=%d blocked=%d dropped=%d hosts=%d pps=%.0f\n",
				s.Elapsed.Round(time.Second), s.Sent, s.Blocked, s.Dropped, s.ActiveHosts, s.CurrentPPS)
		case <-ctx.Done():
			return
		}
	}
}

func runTUI(ctx context.Context, st *stats.Stats, logs *logbuf.Buffer, auditPath string, cancel context.CancelFunc) {
	m := tui.New(st, logs, auditPath, cancel)
	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		<-ctx.Done()
		p.Quit()
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "thugsflooder: tui error:", err)
	}
}

func printSummary(s stats.Snapshot, auditPath string) {
	fmt.Println()
	fmt.Println("thugsflooder stopped.")
	fmt.Printf("  elapsed:      %s\n", s.Elapsed.Round(time.Second))
	fmt.Printf("  sent:         %d\n", s.Sent)
	fmt.Printf("  blocked:      %d\n", s.Blocked)
	fmt.Printf("  dropped:      %d\n", s.Dropped)
	fmt.Printf("  bytes:        %d\n", s.Bytes)
	fmt.Printf("  active hosts: %d\n", s.ActiveHosts)
	fmt.Printf("  audit log:    %s\n", auditPath)
}
