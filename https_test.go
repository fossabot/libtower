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
