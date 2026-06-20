package libtower

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"time"
)

// Check performs an HTTP trace check.
func (ht *HTTPTrace) Check(ctx context.Context) Result {
	err := ht.TraceContext(ctx)
	return Result{OK: err == nil, Duration: ht.Total, Error: err}
}

// Trace http
func (ht *HTTPTrace) Trace() error {
	return ht.TraceContext(context.Background())
}

// TraceContext round-trips an HTTP request capturing per-phase latency, respecting ctx cancellation.
func (ht *HTTPTrace) TraceContext(ctx context.Context) error {
	// res := HTTPTrace{URL: url, Method: method}

	req, err := http.NewRequestWithContext(ctx, ht.Method, ht.URL, nil)
	if err != nil {
		return err
	}

	var start, connect, dns, tlsHandshake time.Time

	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			ht.DNS = time.Since(dns)
		},

		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			ht.TLSHandshake = time.Since(tlsHandshake)
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			ht.Connect = time.Since(connect)
		},

		GotFirstResponseByte: func() {
			ht.Connect = time.Since(start)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
		return err
	}
	ht.Total = time.Since(start)

	return nil
}
