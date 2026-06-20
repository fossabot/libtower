# AGENTS.md

Guidance for Claude Code when working in this repository.

## Build & Test

```bash
go build ./...                                              # Build
go test -v -race -tags offline .                            # Offline (no network, no root — default)
sudo -E env "PATH=$PATH" go test -v -race .                 # Online (real network, root for ping)
go test -v -race -tags offline -run TestDNSLookup .         # Single test
go test -cover -coverprofile=coverage.out -tags offline .   # Coverage
go tool cover -func=coverage.out
```

### Build tags

| Command | Mode | Vars | Network |
|---------|------|------|---------|
| `go test -tags offline .` | Offline | `vars_offline.go` sets mocks | None |
| `go test .` | Online | Real implementations | Required (root for ping) |

### Injection points

Package-level vars with real defaults; `vars_offline.go` (`//go:build offline`) sets mocks via `init()`.

| Var | Used by |
|-----|---------|
| `icmpListenFn` | `realICMPPing` |
| `icmpPingFn` | `Ping`, `PingContext` |
| `dnsLookupFn` | `Ping` |
| `resolverFn` | `DNSLookup`, `DNSLookupContext` |
| `wsHandshakeFn` | `WebSocket.WSCheck` |

Tests follow: save `orig`, `defer` restore, set mock.

## Architecture

Flat Go package (`github.com/mismatched/libtower`), no sub-packages.

### Types & files

| File | Type / Function | Purpose |
|------|----------------|---------|
| `libtower.go` | `Time`, `Timeout`, `isOffline` | Shared timing types |
| `checker.go` | `Result`, `Checker`, `CheckData`, `PingCheck` | Unified result interface |
| `tcp.go` | `TCP` | `TCPPortCheck`, `TLSPortCheck` |
| `https.go` | `HTTPS`, `CertExpiringWarning` | `HTTPSCheck` with cert expiry warning |
| `dns.go` | `DNS`, `DNSLookup*`, `DNSLookupFrom*` | DNS resolution |
| `http.go` | `HTTP`, `HTTPTrace` | Type definitions |
| `httpstatus.go` | `HTTP.HTTPStatus*` | HTTP status checks |
| `httptrace.go` | `HTTPTrace.Trace*` | Per-phase HTTP timing |
| `ping.go` | `Ping`, `PingContext`, `realICMPPing` | ICMP echo (needs root) |
| `websocket.go` | `WebSocket` | WebSocket health check |
| `ip.go` | `isIPv4`, `isIPv6` | IP helpers |
| `vars_offline.go` | `offlineICMPConn`, `init()` | Offline test mocks |

### Conventions

- **Pointer receivers** — mutate receiver to set `Start`, `End`, `Duration`, results
- **Context** — all I/O methods accept `context.Context`; short names wrap `context.Background()`
- **Timing** — every struct records `Start`, `End`, `Duration`
- **Go 1.25** — module directive; generics not used (package stays simple)
- **Backward compat** — existing signatures preserved; new `Context` variants added alongside
