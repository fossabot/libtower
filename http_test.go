package libtower

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPStatusSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	hsr := &HTTP{URL: srv.URL, Method: "GET"}
	err := hsr.HTTPStatus()
	if err != nil {
		t.Fatalf("HTTPStatus() error: %v", err)
	}
	if hsr.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", hsr.StatusCode, http.StatusOK)
	}
	if hsr.Status != "200 OK" {
		t.Errorf("Status = %q, want %q", hsr.Status, "200 OK")
	}
	if hsr.Proto != "HTTP/1.1" {
		t.Errorf("Proto = %q, want HTTP/1.1", hsr.Proto)
	}
	if hsr.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", hsr.Duration)
	}
}

func TestHTTPStatusNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	hsr := &HTTP{URL: srv.URL, Method: "GET"}
	err := hsr.HTTPStatus()
	if err != nil {
		t.Fatalf("HTTPStatus() error: %v", err)
	}
	if hsr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", hsr.StatusCode, http.StatusNotFound)
	}
}

func TestHTTPStatusInvalidURL(t *testing.T) {
	hsr := &HTTP{URL: "://bogus", Method: "GET"}
	err := hsr.HTTPStatus()
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestHTTPStatusContextError(t *testing.T) {
	// A URL that will fail to connect — nothing listening on this port
	hsr := &HTTP{URL: "http://127.0.0.1:19998", Method: "GET"}
	err := hsr.HTTPStatus()
	if err == nil {
		t.Error("expected connection error, got nil")
	}
}

func TestTraceSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Use "localhost" hostname (instead of 127.0.0.1) so DNS resolution hooks fire.
	// Extract the port from the test server URL and build a localhost URL.
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	localhostURL := "http://localhost:" + port

	ht := &HTTPTrace{URL: localhostURL, Method: "GET"}
	err := ht.Trace()
	if err != nil {
		t.Fatalf("Trace() error: %v", err)
	}
	if ht.Total <= 0 {
		t.Errorf("Total = %v, want > 0", ht.Total)
	}
	// Connect should be non-zero
	if ht.Connect <= 0 {
		t.Errorf("Connect = %v, want > 0", ht.Connect)
	}
}

func TestTraceHTTPS(t *testing.T) {
	// httptest.NewTLSServer triggers TLS handshake hooks in httptrace.
	// Use the server's client transport so cert verification passes.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()
	http.DefaultTransport = srv.Client().Transport

	ht := &HTTPTrace{URL: srv.URL, Method: "GET"}
	err := ht.Trace()
	if err != nil {
		t.Fatalf("Trace() error: %v", err)
	}
	if ht.Total <= 0 {
		t.Errorf("Total = %v, want > 0", ht.Total)
	}
	// TLS handshake should be non-zero for HTTPS URLs
	if ht.TLSHandshake <= 0 {
		t.Errorf("TLSHandshake = %v, want > 0", ht.TLSHandshake)
	}
}

func TestTraceConnectionError(t *testing.T) {
	ht := &HTTPTrace{URL: "http://127.0.0.1:19999", Method: "GET"}
	err := ht.Trace()
	if err == nil {
		t.Error("expected connection error, got nil")
	}
}

func TestTraceInvalidURL(t *testing.T) {
	ht := &HTTPTrace{URL: "://bogus", Method: "GET"}
	err := ht.Trace()
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}
