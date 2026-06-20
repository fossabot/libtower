package libtower

import (
	"context"
	"errors"
	"net"
	"net/http/httptrace"
	"time"

	"github.com/miekg/dns"
)

// DNS type
type DNS struct {
	ADDR    string
	Timeout time.Duration

	Start    time.Time
	End      time.Time
	Duration time.Duration
}

// resolverFn resolves hostnames via DNS. Tests swap this to point at a local server.
var resolverFn = func(ctx context.Context, addr string) ([]net.IPAddr, error) {
	return (&net.Resolver{}).LookupIPAddr(ctx, addr)
}

// Check performs a DNS lookup check.
func (d *DNS) Check(ctx context.Context) Result {
	ip, dur, err := DNSLookupContext(ctx, d.ADDR)
	return Result{
		OK:       err == nil,
		Duration: dur,
		Data:     DNSData{IP: ip},
		Error:    err,
	}
}

// DNSLookup func
func DNSLookup(addr string) (*net.IPAddr, time.Duration, error) {
	return DNSLookupContext(context.Background(), addr)
}

// DNSLookupContext resolves addr via the default resolver, respecting ctx cancellation.
func DNSLookupContext(ctx context.Context, addr string) (*net.IPAddr, time.Duration, error) {
	var dnsTime time.Time
	var dnsDuration time.Duration
	traceDNS := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) {
			dnsTime = time.Now()
		},
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			dnsDuration = time.Since(dnsTime)
		},
	}
	ctx = httptrace.WithClientTrace(ctx, traceDNS)
	ips, err := resolverFn(ctx, addr)
	if err != nil {
		return nil, 0, err
	}
	if len(ips) == 0 {
		return nil, 0, errors.New("ips len is zero")
	}
	return &ips[0], dnsDuration, nil
}

// DNSLookupFrom func
func DNSLookupFrom(addr string, server string) (*net.IPAddr, time.Duration, error) {
	return DNSLookupFromContext(context.Background(), addr, server)
}

// DNSLookupFromContext performs a direct DNS query to server, respecting ctx cancellation.
func DNSLookupFromContext(ctx context.Context, addr string, server string) (*net.IPAddr, time.Duration, error) {
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		host = server
		port = "53"
	}
	serverIP := net.ParseIP(host)
	if serverIP == nil {
		return new(net.IPAddr), 0, errors.New("failed to parse server ip address")
	}
	serverAddress := net.JoinHostPort(host, port)

	msg := dns.Msg{}
	msg.Id = dns.Id()
	msg.RecursionDesired = true
	msg.Question = []dns.Question{dns.Question{Name: dns.Fqdn(addr), Qtype: dns.TypeA, Qclass: dns.ClassINET}}

	client := dns.Client{Net: "udp"}
	resp, rtt, err := client.ExchangeContext(ctx, &msg, serverAddress)

	if err != nil {
		return nil, 0, errors.New("dns exchange error: " + err.Error())
	}
	if resp == nil {
		return nil, 0, errors.New("response is nil")
	}
	if resp.Rcode != dns.RcodeSuccess {
		return nil, 0, errors.New(dns.RcodeToString[resp.Rcode])
	}
	for _, record := range resp.Answer {
		if t, ok := record.(*dns.A); ok {
			ipAddress := net.IPAddr{IP: t.A}
			return &ipAddress, rtt, nil
		}
	}
	return nil, 0, errors.New("no A record found in DNS response")
}
