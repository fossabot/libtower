package libtower

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// ProtocolICMP DSCP
	IPv4ProtocolICMP = 1
	IPv6ProtocolICMP = 58
)

// icmpPacketConn abstracts the ICMP socket operations so tests can inject a mock.
type icmpPacketConn interface {
	WriteTo(b []byte, dst net.Addr) (int, error)
	ReadFrom(b []byte) (int, net.Addr, error)
	SetReadDeadline(t time.Time) error
	Close() error
}

// icmpListenFn creates an ICMP listening socket. Tests swap this for offline execution.
var icmpListenFn = func(network, address string) (icmpPacketConn, error) {
	return icmp.ListenPacket(network, address)
}

// icmpPingFn performs an ICMP echo round-trip. Tests swap this for offline/no-root execution.
var icmpPingFn = realICMPPing

// dnsLookupFn resolves hostnames to IP addresses. Tests swap this for offline execution.
var dnsLookupFn = DNSLookup

// Ping sends an ICMP echo request to addr and returns the resolved address,
// round-trip time, and any error.
//
// Requires root privileges (or CAP_NET_RAW on Linux) for raw ICMP sockets.
// Without root, this returns "socket: operation not permitted".
// Use -tags offline for rootless testing.
func Ping(addr string, seq int) (*net.IPAddr, time.Duration, error) {
	dst, _, err := dnsLookupFn(addr)
	if err != nil {
		return dst, 0, err
	}
	dur, err := icmpPingFn(dst, seq)
	return dst, dur, err
}

// PingContext sends an ICMP echo request to addr, respecting ctx cancellation.
//
// Requires root privileges (or CAP_NET_RAW on Linux) for raw ICMP sockets.
// Without root, this returns "socket: operation not permitted".
// Use -tags offline for rootless testing.
func PingContext(ctx context.Context, addr string, seq int) (*net.IPAddr, time.Duration, error) {
	dst, _, err := DNSLookupContext(ctx, addr)
	if err != nil {
		return dst, 0, err
	}
	dur, err := icmpPingFn(dst, seq)
	return dst, dur, err
}

// realICMPPing performs the actual ICMP echo round-trip to an already-resolved address.
func realICMPPing(dst *net.IPAddr, seq int) (time.Duration, error) {
	var c icmpPacketConn
	var err error

	if isIPv4(dst.IP) {
		c, err = icmpListenFn("ip4:icmp", "0.0.0.0")
	} else if isIPv6(dst.IP) {
		c, err = icmpListenFn("ip6:ipv6-icmp", "::")
	} else {
		return 0, fmt.Errorf("cannot determine IP version for %s", dst.IP.String())
	}
	if err != nil {
		return 0, err
	}
	defer c.Close()

	var msg icmp.Message
	if isIPv4(dst.IP) {
		msg = icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  seq,
				Data: []byte(""),
			},
		}
	} else { // IPv6
		msg = icmp.Message{
			Type: ipv6.ICMPTypeEchoRequest,
			Code: 0,
			Body: &icmp.Echo{
				ID:   os.Getpid() & 0xffff,
				Seq:  seq,
				Data: []byte(""),
			},
		}
	}
	bmsg, err := msg.Marshal(nil)
	if err != nil {
		return 0, err
	}

	// Send ICMP message
	start := time.Now()
	n, err := c.WriteTo(bmsg, dst)
	if err != nil {
		return 0, err
	} else if n != len(bmsg) {
		return 0, fmt.Errorf("got %v; want %v", n, len(bmsg))
	}

	// Wait for an ICMP reply
	reply := make([]byte, 1500)
	err = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return 0, err
	}
	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		return 0, err
	}
	duration := time.Since(start)

	if isIPv4(dst.IP) {
		rm, err := icmp.ParseMessage(IPv4ProtocolICMP, reply[:n])
		if err != nil {
			return 0, err
		}
		if rm.Type == ipv4.ICMPTypeEchoReply {
			return duration, nil
		}
		return 0, fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
	}
	// IPv6
	rm, err := icmp.ParseMessage(IPv6ProtocolICMP, reply[:n])
	if err != nil {
		return 0, err
	}
	if rm.Type == ipv6.ICMPTypeEchoReply {
		return duration, nil
	}
	return 0, fmt.Errorf("got %+v from %v; want echo reply v6", rm, peer)
}
