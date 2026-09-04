# thugsflooder

Scoped network/log load-generation tool for authorized red-team/blue-team
exercises, by [THUGS(red)](https://thugs.red).

thugsflooder generates sustained, high-volume synthetic network traffic
(UDP/TCP/HTTP junk and replayed traffic recordings) against an explicit,
user-supplied allowlist of hosts/ports/protocols, with a live terminal
dashboard: a bandwidth/rate bar graph, a scrolling log/status pane, and a
summary stats box (sent/blocked/dropped/bytes/hosts/per-protocol counters).

It exists so a blue team / SOC can train against, and tune detection and
logging pipelines for, what a real large-scale flood looks like in an
authorized environment — run alongside actual red-team payload testing,
not as a way to disguise it.

## Safety model

- **Allowlist-only.** thugsflooder will never send traffic to a
  host:port:protocol that isn't explicitly listed in the `--config` file.
  There is no "discover and flood everything" mode. This is enforced
  twice: once when the config is loaded, and again immediately before
  every single send.
- **Mandatory audit log.** Every send attempt (sent/blocked/dropped) is
  permanently recorded to a local, append-only JSON-lines audit log. It
  cannot be disabled or pointed at `/dev/null`.
- **Marked synthetic traffic.** Every payload thugsflooder sends carries
  an identifiable marker (`THUGSFLOODER-TEST-TRAFFIC` + a per-run session
  id), so anyone inspecting captured traffic can tell it's this tool's
  own synthetic content. thugsflooder does not accept, inject, or mix in
  externally supplied exploit/C2 payloads.
- **Gated by design.** thugsflooder will not generate any traffic
  without `--i-understand-the-risk`. Without it, running the binary in
  any way only ever prints the about/legal notice and usage help.

Only run this against networks and hosts you own or are explicitly
authorized to test.

## Usage

```
thugsflooder gen-config [path]                # write an example allowlist config
thugsflooder run --i-understand-the-risk \
  --config targets.yaml [--headless] \
  [--max-duration 10m] [--replay recording.jsonl] [--speed 2.0]
```

See `thugsflooder --help` for full details, and `examples/targets.example.yaml`
/ `examples/recording.example.jsonl` for config and replay-recording formats.

## Building

Requires Go and `dpkg-deb` (any Debian/Ubuntu host — packaging doesn't
require matching the host architecture).

```
make build   # cross-compiled binaries -> dist/<arch>/thugsflooder
make deb     # .deb packages -> dist/thugsflooder_<version>_<arch>.deb
make all     # both
```

Targets: `i386` (32-bit Intel), `amd64` (64-bit Intel), `armhf` (32-bit
ARM), `arm64` (64-bit ARM).

## License

MIT — see [LICENSE](LICENSE).
