// Package libtower provides network health-check primitives for Go.
//
// It covers TCP dials, TLS handshakes (with client certificates), HTTP status
// checks, per-phase HTTP timing traces, TLS certificate validation with expiry
// warnings, DNS resolution (both system resolver and direct queries), ICMP
// ping, and WebSocket connectivity.
//
// # Common pattern
//
// Every check follows the same pattern: create a struct, set its input fields,
// call its method (which records Start, End, and Duration), and read the
// result. All I/O methods accept a context.Context.
//
// # Checker interface
//
// All check types implement the [Checker] interface, returning a [Result]:
//
//	checks := []Checker{
//	    &TCP{URL: u, Timeout: time.Second},
//	    &HTTPS{Host: "example.com", WarnIfExpiringWithin: 30 * 24 * time.Hour},
//	    &DNS{ADDR: "example.com"},
//	}
//	for _, c := range checks {
//	    r := c.Check(ctx)
//	    if !r.OK { log.Fatal(r.Error) }
//	}
//
// # Context variants
//
// Short-form functions ([Ping], [DNSLookup]) use [context.Background] internally.
// Use the ...Context variants ([PingContext], [DNSLookupContext]) for
// cancellation and timeouts:
//
//	ip, dur, err := libtower.PingContext(ctx, "example.com", 1)
//
// # Root requirement
//
// [Ping] and [PingContext] require raw ICMP sockets, which need root privileges
// (or CAP_NET_RAW on Linux). All other checks work without special permissions.
package libtower
