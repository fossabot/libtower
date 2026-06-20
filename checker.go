package libtower

import (
	"context"
	"net"
	"time"
)

// CheckData is implemented by types carrying check-specific result data.
type CheckData interface {
	// Kind returns a short identifier for the check type.
	Kind() string
}

// Result is the unified return type for all health checks.
type Result struct {
	OK       bool
	Duration time.Duration
	Data     CheckData // check-specific data, nil if none
	Warning  error     // non-nil when the check passed but with a warning (e.g., cert expiring soon)
	Error    error
}

// Checker is implemented by types that can perform a health check.
type Checker interface {
	Check(ctx context.Context) Result
}

// --- Concrete CheckData types ---

// PingData holds the resolved IP address from a Ping check.
type PingData struct{ IP *net.IPAddr }

func (PingData) Kind() string { return "ping" }

// DNSData holds the resolved IP address from a DNS check.
type DNSData struct{ IP *net.IPAddr }

func (DNSData) Kind() string { return "dns" }

// CertData holds the earliest certificate NotAfter time from a TLS check.
type CertData struct{ NotAfter time.Time }

func (CertData) Kind() string { return "tls_cert" }

// PingCheck is a struct wrapper that makes the Ping function implement Checker.
type PingCheck struct {
	Addr string
	Seq  int
}

// Check sends an ICMP echo request to Addr.
func (pc *PingCheck) Check(ctx context.Context) Result {
	ip, dur, err := PingContext(ctx, pc.Addr, pc.Seq)
	return Result{
		OK:       err == nil,
		Duration: dur,
		Data:     PingData{IP: ip},
		Error:    err,
	}
}
