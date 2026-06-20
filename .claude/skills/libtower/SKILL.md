---
name: libtower
description: |-
  Use libtower for network health checks in Go covering TCP, TLS, HTTP, HTTPS, DNS, ICMP ping, WebSocket, and certificate expiry.
  Trigger whenever the user needs to: monitor a service, probe an endpoint, check if a port is open, resolve DNS, measure latency, verify SSL certificates, write a readiness/liveness probe, do synthetic monitoring, or build a health-check function in Go.
  Reach for this skill even if libtower isn't named — "is X reachable?", "is port Y open?", "health check service Z", "uptime monitor", "readiness probe", "ping from Go", "check SSL expiry", "DNS health check", "websocket health check" all signal this skill.
  Also use when composing multiple checks or needing the Checker interface pattern.
---

# libtower — Go Network Health Checks

Package `github.com/mismatched/libtower` — flat package, no sub-packages. Every check follows the same pattern: create a struct, call its method, read `Duration` + result.

## Quick reference

| Goal | Code |
|------|------|
| TCP port open? | `tcp := &libtower.TCP{URL: u, Timeout: 5*time.Second}` → `tcp.TCPPortCheck(ctx)` |
| TLS port + client cert? | Same `TCP`, set `CertFile`/`PrivateKeyFile` → `tcp.TLSPortCheck(ctx)` |
| HTTP status? | `http := &libtower.HTTP{URL: url, Method: "GET"}` → `http.HTTPStatus()` |
| HTTP per-phase timing? | `trace := &libtower.HTTPTrace{URL: url, Method: "GET"}` → `trace.Trace()` |
| TLS cert valid + expiry? | `hs := &libtower.HTTPS{Host: host, Timeout: 5*time.Second}` → `hs.HTTPSCheck(ctx)` |
| Cert expiring within N days? | Set `hs.WarnIfExpiringWithin = N * 24 * time.Hour` → check `r.Warning` after `hs.Check(ctx)` |
| DNS resolve? | `ip, dur, err := libtower.DNSLookup("example.com")` |
| Direct DNS query? | `ip, dur, err := libtower.DNSLookupFrom("example.com", "8.8.8.8")` |
| ICMP ping? ⚠️ **needs root** | `ip, dur, err := libtower.Ping("example.com", 1)` |
| WebSocket? | `ws := &libtower.WebSocket{URL: "ws://...", Timeout: 5*time.Second}` → `ws.WSCheck(ctx)` |
| Compose N checks generically? | Use `Checker` interface — all types implement `Check(ctx) Result` |

⚠️ **Ping requires root** (or `CAP_NET_RAW` on Linux). Without root: `"socket: operation not permitted"`.

## Key conventions

- **Pointer receivers** — methods mutate the receiver, populating `Start`, `End`, `Duration`
- **Context** — all I/O methods accept `context.Context`. Short names (`Ping`, `DNSLookup`) wrap `context.Background()`. Use `PingContext`, `DNSLookupContext`, etc. for cancellation
- **URLs** — TCP type uses `*url.URL` with `tcp://` scheme: `url.Parse("tcp://host:port")`
- **Timing** — every struct has `Start`, `End`, `Duration`; set automatically

## Core types

```go
type Result struct {
    OK       bool
    Duration time.Duration
    Data     CheckData   // PingData{IP}, DNSData{IP}, CertData{NotAfter}, or nil
    Warning  error       // non-nil on soft warnings (e.g., cert expiring soon)
    Error    error
}
type Checker interface {
    Check(ctx context.Context) Result
}
```

### TCP
```go
u, _ := url.Parse("tcp://example.com:443")
tcp := &libtower.TCP{URL: u, Timeout: 5 * time.Second}
ok, err := tcp.TCPPortCheck(ctx)   // plain TCP dial
ok, err := tcp.TLSPortCheck(ctx)   // TLS handshake with client cert
```

### HTTPS
```go
hs := &libtower.HTTPS{
    Host:                  "example.com",
    Timeout:               5 * time.Second,
    WarnIfExpiringWithin:  30 * 24 * time.Hour, // optional; zero = no warning
}
ok, notAfter, err := hs.HTTPSCheck(ctx)  // raw: (bool, time.Time, error)
r := hs.Check(ctx)                       // Result with .Warning populated
```

### DNS
```go
ip, dur, err := libtower.DNSLookup("example.com")
ip, dur, err := libtower.DNSLookupContext(ctx, "example.com")       // with cancellation
ip, dur, err := libtower.DNSLookupFrom("example.com", "8.8.8.8")   // query specific server
ip, dur, err := libtower.DNSLookupFromContext(ctx, "example.com", "8.8.8.8")
```

### HTTP
```go
h := &libtower.HTTP{URL: "https://example.com", Method: "GET"}
h.HTTPStatus()              // populates StatusCode, Header, Body, Duration
h.HTTPStatusContext(ctx)    // context-aware

tr := &libtower.HTTPTrace{URL: "https://example.com", Method: "GET"}
tr.Trace()                  // populates DNS, TLSHandshake, Connect, Total
tr.TraceContext(ctx)
```

### Ping — ⚠️ needs root
```go
ip, dur, err := libtower.Ping("example.com", 1)
ip, dur, err := libtower.PingContext(ctx, "example.com", 1)

// Struct wrapper for the Checker interface:
r := libtower.PingCheck{Addr: "example.com", Seq: 1}.Check(ctx)
```

### WebSocket
```go
ws := &libtower.WebSocket{URL: "ws://echo.example.com", Timeout: 5 * time.Second}
err := ws.WSCheck(ctx)
```

## Composing checks with the Checker interface

Every type implements `Checker`, so you can run them uniformly:

```go
u, _ := url.Parse("tcp://example.com:443")
checks := []libtower.Checker{
    &libtower.TCP{URL: u, Timeout: time.Second},
    &libtower.HTTPS{Host: "example.com", WarnIfExpiringWithin: 30 * 24 * time.Hour},
    &libtower.DNS{ADDR: "example.com"},
    libtower.PingCheck{Addr: "example.com", Seq: 1},
}
for _, c := range checks {
    r := c.Check(ctx)
    switch {
    case !r.OK:
        log.Printf("%T FAILED: %v", c, r.Error)
    case r.Warning != nil:
        log.Printf("%T WARN: %v", c, r.Warning)
    default:
        log.Printf("%T OK (%v)", c, r.Duration)
    }
}
```

## Anti-patterns (don't do these)

- ❌ `net.DialTimeout` / `crypto/tls.Dial` — use libtower types instead
- ❌ Hand-rolling ICMP with `golang.org/x/net/icmp` — use `libtower.Ping()`
- ❌ Using `Ping()` without root — it will fail with `"socket: operation not permitted"`
- ❌ Not checking `r.Warning` after `HTTPS.Check()` when `WarnIfExpiringWithin` is set
