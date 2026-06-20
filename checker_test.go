package libtower

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"
)

func TestResult(t *testing.T) {
	r := Result{OK: true, Duration: 5 * time.Millisecond}
	if !r.OK {
		t.Error("OK = false, want true")
	}
	if r.Duration != 5*time.Millisecond {
		t.Errorf("Duration = %v, want 5ms", r.Duration)
	}
	if r.Error != nil {
		t.Errorf("Error = %v, want nil", r.Error)
	}
	if r.Data != nil {
		t.Errorf("Data = %v, want nil", r.Data)
	}
}

func TestCheckDataKinds(t *testing.T) {
	tests := []struct {
		data CheckData
		kind string
	}{
		{PingData{IP: &net.IPAddr{}}, "ping"},
		{DNSData{IP: &net.IPAddr{}}, "dns"},
		{CertData{NotAfter: time.Now()}, "tls_cert"},
	}
	for _, tt := range tests {
		if tt.data.Kind() != tt.kind {
			t.Errorf("%T.Kind() = %q, want %q", tt.data, tt.data.Kind(), tt.kind)
		}
	}
}

func TestTCPCheck(t *testing.T) {
	ctx := context.TODO()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	u, _ := url.Parse("tcp://" + ln.Addr().String())
	tr := &TCP{URL: u, Timeout: time.Second}

	r := tr.Check(ctx)
	if !r.OK {
		t.Fatalf("Check() OK = false, want true; err=%v", r.Error)
	}
	if r.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", r.Duration)
	}
	if r.Data != nil {
		t.Errorf("Data = %v, want nil (TCP has no data)", r.Data)
	}
}

func TestTCPCheckRefused(t *testing.T) {
	ctx := context.TODO()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	u, _ := url.Parse("tcp://" + addr)
	tr := &TCP{URL: u, Timeout: time.Second}

	r := tr.Check(ctx)
	if r.OK {
		t.Fatal("Check() OK = true, want false (connection refused)")
	}
	if r.Error == nil {
		t.Fatal("Error = nil, want non-nil")
	}
}

func TestHTTPCheck(t *testing.T) {
	hsr := &HTTP{URL: "://bogus", Method: "GET"}
	r := hsr.Check(context.TODO())
	if r.OK {
		t.Fatal("Check() OK = true, want false (invalid URL)")
	}
	if r.Error == nil {
		t.Fatal("Error = nil, want non-nil")
	}
}

func TestHTTPTraceCheck(t *testing.T) {
	ht := &HTTPTrace{URL: "://bogus", Method: "GET"}
	r := ht.Check(context.TODO())
	if r.OK {
		t.Fatal("Check() OK = true, want false (invalid URL)")
	}
}

func TestHTTPSCheckResult(t *testing.T) {
	ctx := context.TODO()
	// Port empty, nothing listening — expects error
	hs := &HTTPS{Host: "127.0.0.1", Timeout: 100 * time.Millisecond}
	r := hs.Check(ctx)
	if r.OK {
		t.Skip("something running on 127.0.0.1:443 — skipping")
	}
	if r.Error == nil {
		t.Fatal("Error = nil, want non-nil")
	}
}

func TestDNSCheck(t *testing.T) {
	d := &DNS{ADDR: "totally.unknown.host"}
	r := d.Check(context.TODO())
	if r.OK {
		t.Fatal("Check() OK = true, want false (unknown host)")
	}
	if r.Error == nil {
		t.Fatal("Error = nil, want non-nil")
	}
}

func TestPingCheck(t *testing.T) {
	pc := PingCheck{Addr: "nonexistent.example", Seq: 1}
	r := pc.Check(context.TODO())
	if r.OK {
		t.Fatal("Check() OK = true, want false (nonexistent host)")
	}
	if r.Error == nil {
		t.Fatal("Error = nil, want non-nil")
	}
}

func TestCheckerInterface(t *testing.T) {
	// Verify all types satisfy the Checker interface
	var _ Checker = (*TCP)(nil)
	var _ Checker = (*HTTPS)(nil)
	var _ Checker = (*HTTP)(nil)
	var _ Checker = (*HTTPTrace)(nil)
	var _ Checker = (*DNS)(nil)
	var _ Checker = PingCheck{}
}