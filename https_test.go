package libtower

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"
)

func generateLocalhostCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(1 * time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair: %v", err)
	}
	return cert
}

func newLocalTLSServer(t *testing.T) (net.Listener, string, string) {
	t.Helper()
	cert := generateLocalhostCert(t)
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsCfg)
	if err != nil {
		t.Fatalf("tls.Listen: %v", err)
	}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		tlsConn := tls.Server(conn, tlsCfg)
		tlsConn.Handshake()
		tlsConn.Close()
	}()
	host, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}
	return ln, host, port
}

func TestHTTPSCheckSuccess(t *testing.T) {
	ctx := context.TODO()
	ln, host, port := newLocalTLSServer(t)
	defer ln.Close()

	hs := &HTTPS{Host: host, Port: port, Timeout: time.Second, InsecureSkipVerify: true}
	ok, notAfter, err := hs.HTTPSCheck(ctx)
	if err != nil {
		t.Fatalf("HTTPSCheck() error: %v", err)
	}
	if !ok {
		t.Error("HTTPSCheck() = false, want true")
	}
	if notAfter.IsZero() {
		t.Error("NotAfter is zero, want a valid time")
	}
	if notAfter.Before(time.Now()) {
		t.Errorf("NotAfter %v is in the past", notAfter)
	}
	if hs.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", hs.Duration)
	}
}

func TestHTTPSCheckRefused(t *testing.T) {
	ctx := context.TODO()
	ln, host, port := newLocalTLSServer(t)
	ln.Close()

	hs := &HTTPS{Host: host, Port: port, Timeout: time.Second}
	ok, _, err := hs.HTTPSCheck(ctx)
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
	if ok {
		t.Error("HTTPSCheck() = true, want false")
	}
}

func TestHTTPSCheckDefaultPort(t *testing.T) {
	ctx := context.TODO()
	// Port empty — function defaults to "443" and mutation is visible even on error
	hs := &HTTPS{Host: "127.0.0.1", Timeout: time.Second}
	_, _, err := hs.HTTPSCheck(ctx)
	if err == nil {
		t.Skip("something running on 127.0.0.1:443 — skipping")
	}
	if hs.Port != DefaultHTTPSPort {
		t.Errorf("Port = %q, want %q", hs.Port, DefaultHTTPSPort)
	}
}

func TestHTTPSCheckCertExpiryWarning(t *testing.T) {
	ctx := context.TODO()
	ln, host, port := newLocalTLSServer(t)
	defer ln.Close()

	// Set a large warning window — the test cert is valid for 1 hour, so it triggers
	hs := &HTTPS{
		Host:                  host,
		Port:                  port,
		Timeout:               time.Second,
		InsecureSkipVerify:    true,
		WarnIfExpiringWithin: 2 * time.Hour,
	}
	r := hs.Check(ctx)
	if !r.OK {
		t.Fatalf("Check() OK = false, want true; err=%v", r.Error)
	}
	if r.Warning == nil {
		t.Fatal("Warning = nil, want non-nil (cert expiring within 2h)")
	}
	cw, ok := r.Warning.(*CertExpiringWarning)
	if !ok {
		t.Fatalf("Warning type = %T, want *CertExpiringWarning", r.Warning)
	}
	if cw.NotAfter.IsZero() {
		t.Error("NotAfter is zero")
	}
	if cw.Remaining <= 0 {
		t.Errorf("Remaining = %v, want > 0", cw.Remaining)
	}
}

func TestHTTPSCheckCertExpiryNoWarning(t *testing.T) {
	ctx := context.TODO()
	ln, host, port := newLocalTLSServer(t)
	defer ln.Close()

	hs := &HTTPS{
		Host:                  host,
		Port:                  port,
		Timeout:               time.Second,
		InsecureSkipVerify:    true,
		WarnIfExpiringWithin: 0, // disabled
	}
	r := hs.Check(ctx)
	if !r.OK {
		t.Fatalf("Check() OK = false, want true; err=%v", r.Error)
	}
	if r.Warning != nil {
		t.Errorf("Warning = %v, want nil (WarnIfExpiringWithin=0)", r.Warning)
	}
}

func TestCertExpiringWarningError(t *testing.T) {
	w := &CertExpiringWarning{
		NotAfter:  time.Now().Add(30 * time.Minute),
		Remaining: 30 * time.Minute,
	}
	s := w.Error()
	if s == "" {
		t.Error("Error() returned empty string")
	}
	if !strings.Contains(s, "30m") && !strings.Contains(s, "certificate") {
		t.Errorf("Error() = %q, want expiry message", s)
	}
}

func TestHTTPSCheckHandshakeError(t *testing.T) {
	ctx := context.TODO()

	// Plain TCP server — TLS handshake will fail, covering rawConn.Close()
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

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	hs := &HTTPS{Host: host, Port: port, Timeout: time.Second}
	ok, _, err := hs.HTTPSCheck(ctx)
	if err == nil {
		t.Fatal("expected TLS handshake error against plain TCP, got nil")
	}
	if ok {
		t.Error("HTTPSCheck() = true, want false")
	}
}
