package libtower

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"testing"
	"time"
)

func TestTCPPortCheckSuccess(t *testing.T) {
	ctx := context.TODO()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()
	// Accept in background so dial doesn't get refused
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	addr := ln.Addr().String()
	u, err := url.Parse("tcp://" + addr)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	tr := &TCP{URL: u, Timeout: time.Second}
	ok, err := tr.TCPPortCheck(ctx)
	if err != nil {
		t.Fatalf("TCPPortCheck() error: %v", err)
	}
	if !ok {
		t.Error("TCPPortCheck() = false, want true")
	}
	if tr.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", tr.Duration)
	}
}

func TestTCPPortCheckRefused(t *testing.T) {
	ctx := context.TODO()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close() // close immediately — nothing accepting

	u, err := url.Parse("tcp://" + addr)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	tr := &TCP{URL: u, Timeout: time.Second}
	ok, err := tr.TCPPortCheck(ctx)
	if err == nil {
		t.Fatal("expected connection refused error, got nil")
	}
	if ok {
		t.Error("TCPPortCheck() = true, want false")
	}
}

func TestTLSPortCheckSuccess(t *testing.T) {
	ctx := context.TODO()

	cert, err := tls.LoadX509KeyPair("data/certs/server.pem", "data/certs/server.key")
	if err != nil {
		t.Fatalf("failed to load server cert: %v", err)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatalf("failed to listen TLS: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Complete TLS handshake so client can finish
		tlsConn := tls.Server(conn, tlsCfg)
		tlsConn.Handshake()
		tlsConn.Close()
	}()

	addr := ln.Addr().String()
	u, err := url.Parse("tcp://" + addr)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	tr := &TCP{
		URL:            u,
		Timeout:        time.Second,
		CertFile:       "data/certs/client.pem",
		PrivateKeyFile: "data/certs/client.key",
	}
	ok, err := tr.TLSPortCheck(ctx)
	if err != nil {
		t.Fatalf("TLSPortCheck() error: %v", err)
	}
	if !ok {
		t.Error("TLSPortCheck() = false, want true")
	}
	if tr.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", tr.Duration)
	}
}

func TestTLSPortCheckBadCert(t *testing.T) {
	ctx := context.TODO()

	u, err := url.Parse("tcp://127.0.0.1:443")
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	tr := &TCP{
		URL:            u,
		Timeout:        time.Second,
		CertFile:       "data/certs/doesnotexist.pem",
		PrivateKeyFile: "data/certs/doesnotexist.key",
	}
	_, err = tr.TLSPortCheck(ctx)
	if err == nil {
		t.Fatal("expected error for bad cert path, got nil")
	}
}

func TestTLSPortCheckRefused(t *testing.T) {
	ctx := context.TODO()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	u, err := url.Parse("tcp://" + addr)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	tr := &TCP{
		URL:            u,
		Timeout:        time.Second,
		CertFile:       "data/certs/client.pem",
		PrivateKeyFile: "data/certs/client.key",
	}
	ok, err := tr.TLSPortCheck(ctx)
	if err == nil {
		t.Fatal("expected connection refused error, got nil")
	}
	if ok {
		t.Error("TLSPortCheck() = true, want false")
	}
}

func TestTLSPortCheckHandshakeError(t *testing.T) {
	ctx := context.TODO()

	// Plain TCP server — TLS handshake will fail
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			buf := make([]byte, 1024)
			conn.Read(buf) // read ClientHello, then close
			conn.Close()
		}
	}()

	addr := ln.Addr().String()
	u, err := url.Parse("tcp://" + addr)
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	tr := &TCP{
		URL:            u,
		Timeout:        time.Second,
		CertFile:       "data/certs/client.pem",
		PrivateKeyFile: "data/certs/client.key",
	}
	ok, err := tr.TLSPortCheck(ctx)
	if err == nil {
		t.Fatal("expected TLS handshake error against plain TCP, got nil")
	}
	if ok {
		t.Error("TLSPortCheck() = true, want false")
	}
}
