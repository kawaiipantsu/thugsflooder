# Payload library

A ready-to-use library of 187 `--replay` entries (see [`internal/recording`](../internal/recording))
spanning several traffic flavors, so you don't have to hand-craft a recording just to see varied
noise on the wire:

| File | Entries | Flavor |
|---|---|---|
| `http-requests.jsonl` | 30 | Generic HTTP GET/POST/PUT/HEAD/OPTIONS/DELETE requests, varied paths and user-agents |
| `web-traffic-mixed.jsonl` | 20 | Ordinary browsing patterns: cookies, referrers, query strings, mixed content types |
| `app-json.jsonl` | 15 | JSON API event bodies (`order.created`, `user.login`, etc.) over HTTP POST |
| `dns-queries.jsonl` | 22 | DNS-lookup-flavored UDP blobs (marked text, not real DNS wire format) |
| `ntp-like-udp-noise.jsonl` | 16 | Generic marked UDP filler shaped like NTP-flavored noise — **not** real NTP wire-format bytes, so this is not a functional amplification payload |
| `tcp-banners.jsonl` | 24 | SSH/SMTP/FTP/POP3/IMAP-style banner text sent over TCP |
| `irc-traffic.jsonl` | 22 | IRC-flavored lines (`NICK`/`USER`/`JOIN`/`PRIVMSG`/`PING`/`PONG`, ...) |
| `smtp-ftp-traffic.jsonl` | 14 | SMTP/FTP command-flavored TCP lines (`HELO`/`MAIL FROM`/`USER`/`RETR`, ...) |
| `generic-junk.jsonl` | 24 | Unstructured marked filler spread across every protocol |

Every entry targets one of the two hosts in [`../examples/targets.example.yaml`](../examples/targets.example.yaml)
(`10.0.0.5` on ports 53/123/8080, `10.0.0.10` on port 443), so the whole library runs immediately
against that example allowlist — no editing required, just point `--config` at your own copy with
those hosts changed to ones you actually own or are authorized to test:

```bash
thugsflooder gen-config targets.yaml     # writes the same 10.0.0.5 / 10.0.0.10 example
$EDITOR targets.yaml                     # point host: at your own lab hosts

thugsflooder run --i-understand-the-risk --config targets.yaml \
  --replay payloads/http-requests.jsonl

# or combine the whole library into one recording:
cat payloads/*.jsonl > /tmp/all-payloads.jsonl
thugsflooder run --i-understand-the-risk --config targets.yaml --replay /tmp/all-payloads.jsonl
```

Like every other payload thugsflooder sends, every entry's content carries the
`THUGSFLOODER-TEST-TRAFFIC` marker plus a category/sequence tag baked directly into the
payload — since replayed content is sent byte-for-byte as recorded (so it can faithfully mimic a
real protocol's framing), the marker lives in the payload text itself here rather than being
prepended by the code path the way the built-in junk generators do it. It's still synthetic,
still identifiable, still not a channel for real exploit/C2 content — just a wider variety of
shapes than "random bytes" for making SOC/log training more realistic.
