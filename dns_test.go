package libtower

import (
	"context"
	"net"
	"testing"

	"github.com/miekg/dns"
)

// startDNSServer starts a local DNS server that responds to A queries for
// "example.com." with 1.2.3.4 and returns NXDOMAIN for everything else.
// Returns the server and its address (e.g. "127.0.0.1:PORT").
func startDNSServer(t *testing.T) (*dns.Server, string) {
	t.Helper()

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		if len(r.Question) == 0 {
			w.WriteMsg(m)
			return
		}
		q := r.Question[0]
		if q.Name == "example.com." && q.Qtype == dns.TypeA {
			m.Answer = []dns.RR{&dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
				A:   net.ParseIP("1.2.3.4"),
			}}
		} else {
			m.SetRcode(r, dns.RcodeNameError)
		}
		w.WriteMsg(m)
	})

	// Use PacketConn (UDP) for miekg/dns
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket: %v", err)
	}
	server := &dns.Server{PacketConn: pc, Handler: handler}
	go server.ActivateAndServe()
	return server, pc.LocalAddr().String()
}

func TestDNSLookupSuccess(t *testing.T) {
	ip, dur, err := DNSLookup("example.com")
	if err != nil {
		t.Fatalf("DNSLookup() error: %v", err)
	}
	if ip == nil {
		t.Error("got nil IP, want non-nil")
	}
	if dur < 0 {
		t.Errorf("Duration = %v, want >= 0", dur)
	}
}

func TestDNSLookupNXDOMAIN(t *testing.T) {
	orig := resolverFn
	defer func() { resolverFn = orig }()

	resolverFn = func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return nil, &net.DNSError{Err: "no such host", Name: host}
	}

	_, _, err := DNSLookup("nonexistent.example")
	if err == nil {
		t.Fatal("expected error for NXDOMAIN, got nil")
	}
}

func TestDNSLookupFromSuccess(t *testing.T) {
	if !isOffline {
		t.Skip("skipping in online mode; run with -tags offline")
	}
	srv, addr := startDNSServer(t)
	defer srv.Shutdown()

	ip, dur, err := DNSLookupFrom("example.com", addr)
	if err != nil {
		t.Fatalf("DNSLookupFrom() error: %v", err)
	}
	if ip.String() != "1.2.3.4" {
		t.Errorf("got %s, want 1.2.3.4", ip.String())
	}
	if dur < 0 {
		t.Errorf("Duration = %v, want >= 0", dur)
	}
}

func TestDNSLookupFromNXDOMAIN(t *testing.T) {
	srv, addr := startDNSServer(t)
	defer srv.Shutdown()

	_, _, err := DNSLookupFrom("unknown.example", addr)
	if err == nil {
		t.Fatal("expected error for NXDOMAIN, got nil")
	}
}

func TestDNSLookupFromBadServerIP(t *testing.T) {
	_, _, err := DNSLookupFrom("example.com", "not-an-ip")
	if err == nil {
		t.Fatal("expected error for bad server IP, got nil")
	}
}

func TestDNSLookupFromDefaultPort(t *testing.T) {
	// Passing a bare IP (no port) defaults to :53
	// No server is listening on 127.0.0.2:53, so we expect a network error
	_, _, err := DNSLookupFrom("example.com", "127.0.0.2")
	if err == nil {
		t.Fatal("expected network error for unreachable server, got nil")
	}
}

func TestDNSLookupUnknownHost(t *testing.T) {
	// Host not in the offline resolver map hits the default error case.
	_, _, err := DNSLookup("totally.unknown.host")
	if err == nil {
		t.Fatal("expected error for unknown host, got nil")
	}
}

func TestDNSLookupEmptyResponse(t *testing.T) {
	orig := resolverFn
	defer func() { resolverFn = orig }()

	resolverFn = func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{}, nil
	}

	_, _, err := DNSLookup("example.com")
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
}

func TestDNSLookupFromNoARecord(t *testing.T) {
	// Start a server that returns success but no A records
	handler := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		// Include an AAAA record but no A record
		if len(r.Question) > 0 && r.Question[0].Name == "example.com." {
			m.Answer = []dns.RR{&dns.AAAA{
				Hdr:  dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300},
				AAAA: net.ParseIP("2001:db8::1"),
			}}
		}
		w.WriteMsg(m)
	})
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenPacket: %v", err)
	}
	srv := &dns.Server{PacketConn: pc, Handler: handler}
	go srv.ActivateAndServe()
	defer srv.Shutdown()

	_, _, err = DNSLookupFrom("example.com", pc.LocalAddr().String())
	if err == nil {
		t.Fatal("expected error for no A record, got nil")
	}
}
