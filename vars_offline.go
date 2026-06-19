//go:build offline
// +build offline

package libtower

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// offlineICMPConn is a mock ICMP connection that echoes back a valid reply
// matching the request's protocol version, ID, and sequence number.
type offlineICMPConn struct {
	writeBuf []byte
}

func (c *offlineICMPConn) WriteTo(b []byte, dst net.Addr) (int, error) {
	c.writeBuf = make([]byte, len(b))
	copy(c.writeBuf, b)
	return len(b), nil
}

func (c *offlineICMPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	// Parse the stored echo request to get protocol version.
	// If it starts with ICMPv4 type 8, build an IPv4 echo reply.
	// If it starts with ICMPv6 type 128, build an IPv6 echo reply.
	var reply []byte
	if len(c.writeBuf) > 0 {
		switch c.writeBuf[0] {
		case byte(ipv4.ICMPTypeEcho):
			rm, err := icmp.ParseMessage(IPv4ProtocolICMP, c.writeBuf)
			if err == nil {
				echo, ok := rm.Body.(*icmp.Echo)
				if ok {
					replyMsg := icmp.Message{
						Type: ipv4.ICMPTypeEchoReply,
						Code: 0,
						Body: &icmp.Echo{ID: echo.ID, Seq: echo.Seq},
					}
					reply, _ = replyMsg.Marshal(nil)
				}
			}
		case byte(ipv6.ICMPTypeEchoRequest):
			rm, err := icmp.ParseMessage(IPv6ProtocolICMP, c.writeBuf)
			if err == nil {
				echo, ok := rm.Body.(*icmp.Echo)
				if ok {
					replyMsg := icmp.Message{
						Type: ipv6.ICMPTypeEchoReply,
						Code: 0,
						Body: &icmp.Echo{ID: echo.ID, Seq: echo.Seq},
					}
					reply, _ = replyMsg.Marshal(nil)
				}
			}
		}
	}
	if reply == nil {
		// Fallback: build a minimal IPv4 echo reply
		replyMsg := icmp.Message{
			Type: ipv4.ICMPTypeEchoReply,
			Code: 0,
			Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1},
		}
		reply, _ = replyMsg.Marshal(nil)
	}
	n := copy(b, reply)
	return n, &net.IPAddr{IP: net.ParseIP("127.0.0.1")}, nil
}

func (c *offlineICMPConn) SetReadDeadline(t time.Time) error { return nil }
func (c *offlineICMPConn) Close() error                      { return nil }

func init() {
	isOffline = true
	icmpListenFn = func(network, address string) (icmpPacketConn, error) {
		return &offlineICMPConn{}, nil
	}
	icmpPingFn = func(dst *net.IPAddr, seq int) (time.Duration, error) {
		return 5 * time.Millisecond, nil
	}
	dnsLookupFn = func(addr string) (*net.IPAddr, time.Duration, error) {
		// Delegate to resolverFn so DNSLookup tests share the same mock.
		ips, err := resolverFn(context.Background(), addr)
		if err != nil {
			return nil, 0, err
		}
		if len(ips) == 0 {
			return nil, 0, errors.New("ips len is zero")
		}
		return &ips[0], 1 * time.Millisecond, nil
	}
	resolverFn = func(ctx context.Context, addr string) ([]net.IPAddr, error) {
		switch addr {
		case "example.com":
			return []net.IPAddr{{IP: net.ParseIP("1.2.3.4")}}, nil
		case "nonexistent.example":
			return nil, fmt.Errorf("lookup %s: no such host", addr)
		default:
			return nil, fmt.Errorf("offline resolver: unknown host %s", addr)
		}
	}
}
