package libtower

import (
	"context"
	"net"
	"net/url"
	"time"

	"golang.org/x/net/websocket"
)

// WebSocket type
type WebSocket struct {
	URL     string
	Timeout time.Duration

	Start    time.Time
	End      time.Time
	Duration time.Duration
}

// wsHandshakeFn establishes a WebSocket connection to the given URL.
// Tests swap this for offline execution.
var wsHandshakeFn = func(conn net.Conn, rawURL string) (*websocket.Conn, error) {
	cfg, err := websocket.NewConfig(rawURL, "http://localhost/")
	if err != nil {
		return nil, err
	}
	return websocket.NewClient(cfg, conn)
}

// WSCheck performs a WebSocket health check by dialing and verifying the upgrade.
func (ws *WebSocket) WSCheck(ctx context.Context) error {
	u, err := url.Parse(ws.URL)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		u.Scheme = "ws"
	}

	host := u.Host
	if u.Port() == "" {
		if u.Scheme == "wss" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	dialer := net.Dialer{Timeout: ws.Timeout}
	ws.Start = time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		ws.End = time.Now()
		ws.Duration = ws.End.Sub(ws.Start)
		return err
	}

	wsConn, err := wsHandshakeFn(conn, u.String())
	ws.End = time.Now()
	ws.Duration = ws.End.Sub(ws.Start)
	if err != nil {
		conn.Close()
		return err
	}
	defer wsConn.Close()
	return nil
}

// Check performs a WebSocket health check.
func (ws *WebSocket) Check(ctx context.Context) Result {
	err := ws.WSCheck(ctx)
	return Result{OK: err == nil, Duration: ws.Duration, Error: err}
}
