package libtower

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// mockICMPConn implements icmpPacketConn for error-injection tests.
type mockICMPConn struct {
	readReply   []byte
	readPeer    net.Addr
	readErr     error
	writeErr    error
	writeN      int // if 0, defaults to len(b)
	deadlineErr error
	closed      bool
}

func (m *mockICMPConn) WriteTo(b []byte, dst net.Addr) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	if m.writeN > 0 {
		return m.writeN, nil
	}
	return len(b), nil
}

func (m *mockICMPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	if m.readErr != nil {
		return 0, nil, m.readErr
	}
	n := copy(b, m.readReply)
	return n, m.readPeer, nil
}

func (m *mockICMPConn) SetReadDeadline(t time.Time) error { return m.deadlineErr }
func (m *mockICMPConn) Close() error                      { m.closed = true; return nil }

func makeIPv4EchoReply(seq int) []byte {
	reply := icmp.Message{
		Type: ipv4.ICMPTypeEchoReply, Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: seq},
	}
	b, _ := reply.Marshal(nil)
	return b
}

// --- realICMPPing happy-path tests use the offlineICMPConn from vars_offline.go ---

func TestRealICMPPingIPv4Success(t *testing.T) {
	dst := &net.IPAddr{IP: net.ParseIP("1.2.3.4")}
	dur, err := realICMPPing(dst, 42)
	if err != nil {
		t.Fatalf("realICMPPing() error: %v", err)
	}
	if dur <= 0 {
		t.Errorf("duration = %v, want > 0", dur)
	}
}

func TestRealICMPPingIPv6Success(t *testing.T) {
	dst := &net.IPAddr{IP: net.ParseIP("2001:db8::1")}
	dur, err := realICMPPing(dst, 7)
	if err != nil {
		t.Fatalf("realICMPPing() error: %v", err)
	}
	if dur <= 0 {
		t.Errorf("duration = %v, want > 0", dur)
	}
}

// --- Error-path tests override icmpListenFn for specific failures ---

func TestRealICMPPingListenError(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	listenErr := errors.New("listen failed")
	icmpListenFn = func(network, address string) (icmpPacketConn, error) { return nil, listenErr }

	_, err := realICMPPing(&net.IPAddr{IP: net.ParseIP("1.2.3.4")}, 1)
	if err == nil {
		t.Fatal("expected listen error, got nil")
	}
}

func TestRealICMPPingWriteError(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	writeErr := errors.New("write failed")
	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &mockICMPConn{writeErr: writeErr}, nil
	}

	_, err := realICMPPing(&net.IPAddr{IP: net.ParseIP("1.2.3.4")}, 1)
	if err == nil {
		t.Fatal("expected write error, got nil")
	}
}

func TestRealICMPPingReadError(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	readErr := errors.New("read timeout")
	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &mockICMPConn{readErr: readErr}, nil
	}

	_, err := realICMPPing(&net.IPAddr{IP: net.ParseIP("1.2.3.4")}, 1)
	if err == nil {
		t.Fatal("expected read error, got nil")
	}
}

func TestRealICMPPingUnknownIPVersion(t *testing.T) {
	// empty IP fails both isIPv4 and isIPv6
	_, err := realICMPPing(&net.IPAddr{IP: net.IP{}}, 1)
	if err == nil {
		t.Fatal("expected unknown IP version error, got nil")
	}
}

func TestRealICMPPingWrongReplyType(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	dst := &net.IPAddr{IP: net.ParseIP("1.2.3.4")}
	wrongReply := icmp.Message{
		Type: ipv4.ICMPTypeDestinationUnreachable, Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1},
	}
	replyBytes, _ := wrongReply.Marshal(nil)

	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &mockICMPConn{readReply: replyBytes, readPeer: dst}, nil
	}

	_, err := realICMPPing(dst, 1)
	if err == nil {
		t.Fatal("expected wrong reply type error, got nil")
	}
}

func TestRealICMPPingSetReadDeadlineError(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	deadlineErr := errors.New("set deadline failed")
	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &mockICMPConn{readReply: makeIPv4EchoReply(1), readPeer: &net.IPAddr{IP: net.ParseIP("1.2.3.4")}, deadlineErr: deadlineErr}, nil
	}

	_, err := realICMPPing(&net.IPAddr{IP: net.ParseIP("1.2.3.4")}, 1)
	if err == nil {
		t.Fatal("expected deadline error, got nil")
	}
}

func TestRealICMPPingMalformedIPv4Reply(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	dst := &net.IPAddr{IP: net.ParseIP("1.2.3.4")}
	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &mockICMPConn{readReply: []byte{0xff}, readPeer: dst}, nil
	}

	_, err := realICMPPing(dst, 1)
	if err == nil {
		t.Fatal("expected parse error for malformed IPv4 reply, got nil")
	}
}

func TestRealICMPPingMalformedIPv6Reply(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	dst := &net.IPAddr{IP: net.ParseIP("2001:db8::1")}
	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &mockICMPConn{readReply: []byte{0xff, 0xff}, readPeer: dst}, nil
	}

	_, err := realICMPPing(dst, 1)
	if err == nil {
		t.Fatal("expected parse error for malformed reply, got nil")
	}
}

func TestRealICMPPingWrongReplyTypeIPv6(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	dst := &net.IPAddr{IP: net.ParseIP("2001:db8::1")}
	wrongReply := icmp.Message{
		Type: ipv6.ICMPTypeDestinationUnreachable, Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1},
	}
	replyBytes, _ := wrongReply.Marshal(nil)

	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &mockICMPConn{readReply: replyBytes, readPeer: dst}, nil
	}

	_, err := realICMPPing(dst, 1)
	if err == nil {
		t.Fatal("expected wrong reply type error (IPv6), got nil")
	}
}

// --- Ping tests using default mocks from vars_offline.go ---

func TestPingSuccess(t *testing.T) {
	dst, dur, err := Ping("example.com", 1)
	if err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
	if isOffline && dst.IP.String() != "1.2.3.4" {
		t.Errorf("Ping() dst = %v, want 1.2.3.4", dst)
	}
	if dur <= 0 {
		t.Errorf("Ping() dur = %v, want > 0", dur)
	}
}

func TestPingDNSError(t *testing.T) {
	orig := dnsLookupFn
	defer func() { dnsLookupFn = orig }()

	dnsErr := errors.New("dns resolution failed")
	dnsLookupFn = func(addr string) (*net.IPAddr, time.Duration, error) { return nil, 0, dnsErr }

	_, _, err := Ping("bogus.test", 1)
	if err == nil {
		t.Fatal("expected DNS error, got nil")
	}
}

func TestPingICMPError(t *testing.T) {
	origICMP := icmpPingFn
	origDNS := dnsLookupFn
	defer func() { icmpPingFn = origICMP; dnsLookupFn = origDNS }()

	wantIP := &net.IPAddr{IP: net.ParseIP("5.6.7.8")}
	dnsLookupFn = func(addr string) (*net.IPAddr, time.Duration, error) { return wantIP, 1 * time.Millisecond, nil }
	pingErr := errors.New("icmp send failed")
	icmpPingFn = func(dst *net.IPAddr, seq int) (time.Duration, error) { return 0, pingErr }

	dst, _, err := Ping("example.com", 1)
	if err == nil {
		t.Fatal("expected ICMP error, got nil")
	}
	if dst.IP.String() != wantIP.IP.String() {
		t.Errorf("dst = %v, want %v", dst, wantIP)
	}
}

func TestRealICMPPingShortWrite(t *testing.T) {
	orig := icmpListenFn
	defer func() { icmpListenFn = orig }()

	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &mockICMPConn{
			writeN:    5, // return fewer bytes than written
			readReply: makeIPv4EchoReply(1),
			readPeer:  &net.IPAddr{IP: net.ParseIP("1.2.3.4")},
		}, nil
	}

	_, err := realICMPPing(&net.IPAddr{IP: net.ParseIP("1.2.3.4")}, 1)
	if err == nil {
		t.Fatal("expected short write error, got nil")
	}
}

// --- Ping tests exercising the offline mock chain (dnsLookupFn → resolverFn) ---

func TestPingDNSNonexistent(t *testing.T) {
	// nonexistent.example hits the resolverFn "nonexistent.example" case → error,
	// which exercises dnsLookupFn's error-return path.
	_, _, err := Ping("nonexistent.example", 1)
	if err == nil {
		t.Fatal("expected DNS error for nonexistent.example, got nil")
	}
}

func TestPingDNSUnknown(t *testing.T) {
	// Any host not in the offline resolver map hits the default case → error.
	_, _, err := Ping("totally.unknown.host", 1)
	if err == nil {
		t.Fatal("expected DNS error for unknown host, got nil")
	}
}

func TestPingDNSEmptyIps(t *testing.T) {
	orig := resolverFn
	defer func() { resolverFn = orig }()

	resolverFn = func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{}, nil
	}

	_, _, err := Ping("example.com", 1)
	if err == nil {
		t.Fatal("expected error for empty DNS response, got nil")
	}
}
