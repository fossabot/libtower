# libtower

[![Go Reference](https://pkg.go.dev/badge/github.com/mismatched/libtower.svg)](https://pkg.go.dev/github.com/mismatched/libtower)
[![Go Report Card](https://goreportcard.com/badge/github.com/mismatched/libtower)](https://goreportcard.com/report/github.com/mismatched/libtower)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmismatched%2Flibtower.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fmismatched%2Flibtower?ref=badge_shield)

Network health-check primitives for Go — TCP, TLS, HTTP, HTTPS, DNS, ICMP ping, WebSocket, and certificate expiry. Flat package, no sub-packages.

## Install

```bash
go get github.com/mismatched/libtower
```

## Quick start

```go
package main

import (
    "context"
    "net/url"
    "time"

    "github.com/mismatched/libtower"
)

func main() {
    ctx := context.Background()

    // TCP port check
    u, _ := url.Parse("tcp://example.com:443")
    tcp := &libtower.TCP{URL: u, Timeout: 5 * time.Second}
    ok, _ := tcp.TCPPortCheck(ctx)

    // DNS lookup
    ip, dur, _ := libtower.DNSLookup("example.com")

    // TLS cert check with expiry warning
    hs := &libtower.HTTPS{
        Host:                  "example.com",
        Timeout:               5 * time.Second,
        WarnIfExpiringWithin:  30 * 24 * time.Hour,
    }
    r := hs.Check(ctx)
    if r.Warning != nil {
        println("cert expiring soon:", r.Warning.Error())
    }

    _ = ok
    _ = ip
    _ = dur
}
```

## Checks

| Check | Type/Function | Returns |
|-------|--------------|---------|
| TCP port | `TCP.TCPPortCheck(ctx)` | `(bool, error)` |
| TLS port (client cert) | `TCP.TLSPortCheck(ctx)` | `(bool, error)` |
| HTTP status | `HTTP.HTTPStatus()` | `error` (receiver populated) |
| HTTP trace (per-phase timing) | `HTTPTrace.Trace()` | `error` (receiver populated) |
| TLS cert validity | `HTTPS.HTTPSCheck(ctx)` | `(bool, time.Time, error)` |
| DNS lookup | `DNSLookup(addr)` | `(*net.IPAddr, time.Duration, error)` |
| Direct DNS query | `DNSLookupFrom(addr, server)` | `(*net.IPAddr, time.Duration, error)` |
| ICMP ping ⚠️ | `Ping(addr, seq)` | `(*net.IPAddr, time.Duration, error)` |
| WebSocket | `WebSocket.WSCheck(ctx)` | `error` |

⚠️ **Ping requires root** (or `CAP_NET_RAW` on Linux).

## Composing checks

All types implement `Checker`, so you can run them uniformly:

```go
checks := []libtower.Checker{
    &libtower.TCP{URL: u, Timeout: time.Second},
    &libtower.HTTPS{Host: "example.com", WarnIfExpiringWithin: 30 * 24 * time.Hour},
    &libtower.DNS{ADDR: "example.com"},
}
for _, c := range checks {
    r := c.Check(ctx)
    if !r.OK { panic(r.Error) }
}
```

## Context variants

Short names (`Ping`, `DNSLookup`) use `context.Background()`. Use the `...Context` variants for cancellation:

```go
PingContext(ctx, addr, seq)
DNSLookupContext(ctx, addr)
DNSLookupFromContext(ctx, addr, server)
HTTPStatusContext(ctx)
TraceContext(ctx)
```

## License

MIT


[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmismatched%2Flibtower.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fmismatched%2Flibtower?ref=badge_large)