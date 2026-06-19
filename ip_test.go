package libtower

import (
	"net"
	"testing"
)

func TestIsIPv4(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		{"IPv4 address", net.ParseIP("192.168.1.1"), true},
		{"IPv4 localhost", net.ParseIP("127.0.0.1"), true},
		{"IPv4 zero", net.ParseIP("0.0.0.0"), true},
		{"IPv6 address", net.ParseIP("2001:db8::1"), false},
		{"IPv6 loopback", net.ParseIP("::1"), false},
		{"IPv4-mapped IPv6 (To4 returns non-nil)", net.ParseIP("::ffff:192.0.2.1"), true},
		{"nil IP", nil, false},
		{"empty IP", net.IP{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIPv4(tt.ip); got != tt.want {
				t.Errorf("isIPv4(%v) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsIPv6(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		{"IPv6 address", net.ParseIP("2001:db8::1"), true},
		{"IPv6 loopback", net.ParseIP("::1"), true},
		{"IPv6 link-local", net.ParseIP("fe80::1"), true},
		{"IPv4 address", net.ParseIP("192.168.1.1"), false},
		{"IPv4 zero", net.ParseIP("0.0.0.0"), false},
		{"IPv4-mapped IPv6 (To4 makes it v4)", net.ParseIP("::ffff:192.0.2.1"), false},
		{"nil IP", nil, false},
		{"empty IP", net.IP{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIPv6(tt.ip); got != tt.want {
				t.Errorf("isIPv6(%v) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
