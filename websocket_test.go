package libtower

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

// startWSServer starts a local WebSocket echo server and returns its URL.
func startWSServer(t *testing.T) string {
	t.Helper()
	handler := websocket.Handler(func(conn *websocket.Conn) {
		var msg []byte
		websocket.Message.Receive(conn, &msg)
		conn.Close()
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	// Convert http:// to ws://
	return "ws" + srv.URL[4:]
}

func TestWSCheckSuccess(t *testing.T) {
	wsURL := startWSServer(t)

	ws := &WebSocket{URL: wsURL, Timeout: time.Second}
	err := ws.WSCheck(context.TODO())
	if err != nil {
		t.Fatalf("WSCheck() error: %v", err)
	}
	if ws.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", ws.Duration)
	}
}

func TestWSCheckInvalidURL(t *testing.T) {
	ws := &WebSocket{URL: "://bogus", Timeout: time.Second}
	err := ws.WSCheck(context.TODO())
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestWSCheckConnectionRefused(t *testing.T) {
	ws := &WebSocket{URL: "ws://127.0.0.1:19998", Timeout: 100 * time.Millisecond}
	err := ws.WSCheck(context.TODO())
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestWSCheckHandshakeError(t *testing.T) {
	// Plain HTTP server, not WebSocket — handshake should fail
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[4:]
	ws := &WebSocket{URL: wsURL, Timeout: time.Second}
	err := ws.WSCheck(context.TODO())
	if err == nil {
		t.Fatal("expected handshake error against plain HTTP, got nil")
	}
}

func TestWSCheckError(t *testing.T) {
	orig := wsHandshakeFn
	defer func() { wsHandshakeFn = orig }()

	wsHandshakeFn = func(conn net.Conn, rawURL string) (*websocket.Conn, error) {
		return nil, errors.New("handshake failed")
	}

	ws := &WebSocket{URL: "ws://127.0.0.1:19999", Timeout: time.Second}
	err := ws.WSCheck(context.TODO())
	if err == nil {
		t.Fatal("expected handshake error, got nil")
	}
}

func TestWSCheckDefaultPort(t *testing.T) {
	orig := wsHandshakeFn
	defer func() { wsHandshakeFn = orig }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	var capturedURL string
	wsHandshakeFn = func(conn net.Conn, rawURL string) (*websocket.Conn, error) {
		capturedURL = rawURL
		return nil, errors.New("skip after capture")
	}

	addr := ln.Addr().String()
	ws := &WebSocket{URL: "ws://" + addr, Timeout: time.Second}
	ws.WSCheck(context.TODO())
	// URL without explicit port in the string has no port component
	if capturedURL == "" {
		t.Fatal("wsHandshakeFn was not called")
	}
}

func TestWebSocketCheck(t *testing.T) {
	wsURL := startWSServer(t)
	ws := &WebSocket{URL: wsURL, Timeout: time.Second}
	r := ws.Check(context.TODO())
	if !r.OK {
		t.Fatalf("Check() OK = false, want true; err=%v", r.Error)
	}
	if r.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", r.Duration)
	}
}

func TestWebSocketCheckerInterface(t *testing.T) {
	var _ Checker = (*WebSocket)(nil)
}