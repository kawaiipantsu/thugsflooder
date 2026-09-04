// Package about holds the tool's identity, legal notice, and help text.
// This text is the ONLY output thugsflooder ever produces without
// --i-understand-the-risk, regardless of what other flags are passed.
package about

import "fmt"

// Version is set at build time via -ldflags "-X ...about.Version=...".
var Version = "dev"

const aboutText = `thugsflooder — scoped network/log load-generation tool
by THUGS(red) (https://thugs.red)

PURPOSE
  thugsflooder generates sustained, high-volume synthetic network traffic
  (UDP/TCP/HTTP junk and replayed traffic recordings) against an explicit,
  user-supplied allowlist of hosts/ports/protocols. It exists so a blue
  team / SOC can train against, and tune detection and logging pipelines
  for, what a real large-scale flood looks like in their own environment.

  thugsflooder will NEVER send traffic to a target that is not explicitly
  listed in the --config allowlist file, and it ALWAYS writes a permanent,
  append-only audit log of every attempt it makes. It is intentionally
  self-documenting: it is a training/testing tool for an authorized
  environment, not a tool for disguising or concealing other activity.
  It does not accept, inject, or mix in any externally supplied exploit
  or C2 payloads — the traffic it sends is its own synthetic content only,
  and that content is tagged with an identifiable marker.

AUTHORIZATION
  Only run this tool against networks and hosts you own or are explicitly
  authorized to test (e.g. a written red-team/blue-team exercise scope).
  Running it against systems you do not have authorization for is illegal
  in most jurisdictions. You are solely responsible for how you use it.

  This tool will not start any traffic generation in any capacity unless
  you pass --i-understand-the-risk. Without that flag, it only ever prints
  this notice and usage help.
`

const usageText = `USAGE
  thugsflooder [--help]
  thugsflooder gen-config [path]
      Write an example allowlist config to 'path' (default ./targets.yaml).
      Does not require --i-understand-the-risk (sends no traffic).

  thugsflooder run --i-understand-the-risk --config <file> [options]
      Start traffic generation against the allowlisted targets in <file>.

  RUN OPTIONS
      --config <file>            Allowlist config (YAML). Required.
      --i-understand-the-risk    Required to do anything but print this help.
      --tui                      Launch the live dashboard (default).
      --headless                 Run without the TUI; print periodic stats.
      --max-duration <dur>       Stop automatically after e.g. "5m" (default: unbounded, runs until Ctrl+C/SIGINT).
      --replay <file>            Also replay a recording file (JSON lines).
      --speed <mult>             Replay speed multiplier (default 1.0).

  OTHER
      --version                  Print version and exit.
      --help, -h                  Print this notice and exit.

  Press 'q' or Ctrl+C in the TUI, or send SIGINT/SIGTERM, to stop.
  On stop, a final summary is printed and the audit log is flushed.
`

// PrintGate prints the About + usage text. Called whenever the risk flag
// is missing, --help/-h is passed, or no args are given.
func PrintGate() {
	fmt.Print(aboutText)
	fmt.Println()
	fmt.Print(usageText)
}

// PrintVersion prints the tool's version string.
func PrintVersion() {
	fmt.Printf("thugsflooder %s\n", Version)
}
