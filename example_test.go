package libtower_test

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/mismatched/libtower"
)

func ExampleTCP_TCPPortCheck() {
	u, _ := url.Parse("tcp://example.com:443")
	tcp := &libtower.TCP{URL: u, Timeout: 5 * time.Second}
	ok, err := tcp.TCPPortCheck(context.Background())
	fmt.Println("ok:", ok, "err:", err)
}

func ExampleDNSLookup() {
	ip, dur, err := libtower.DNSLookup("example.com")
	fmt.Println("ip:", ip, "duration:", dur, "err:", err)
}

func ExampleHTTPS_Check() {
	hs := &libtower.HTTPS{
		Host:                  "example.com",
		Timeout:               5 * time.Second,
		WarnIfExpiringWithin:  30 * 24 * time.Hour,
	}
	r := hs.Check(context.Background())
	fmt.Println("ok:", r.OK)
	if r.Data != nil {
		certData := r.Data.(libtower.CertData)
		fmt.Println("notAfter:", certData.NotAfter.Format(time.RFC3339))
	}
	if r.Warning != nil {
		fmt.Println("warning:", r.Warning)
	}
}

func ExampleChecker() {
	u, _ := url.Parse("tcp://example.com:443")
	checks := []libtower.Checker{
		&libtower.TCP{URL: u, Timeout: time.Second},
		&libtower.DNS{ADDR: "example.com"},
	}
	for _, c := range checks {
		r := c.Check(context.Background())
		fmt.Printf("%T ok=%v duration=%v\n", c, r.OK, r.Duration)
	}
}

func ExamplePing() {
	ip, dur, err := libtower.Ping("example.com", 1)
	fmt.Println("ip:", ip, "rtt:", dur, "err:", err)
}

func ExampleWebSocket_WSCheck() {
	ws := &libtower.WebSocket{
		URL:     "ws://echo.example.com",
		Timeout: 5 * time.Second,
	}
	err := ws.WSCheck(context.Background())
	fmt.Println("err:", err)
}
