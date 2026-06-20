package libtower

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"time"
)

// TCP type
type TCP struct {
	URL     *url.URL
	Timeout time.Duration

	CertFile       string
	PrivateKeyFile string

	Start    time.Time
	End      time.Time
	Duration time.Duration
}

// TCPPortCheck checks if a tcp port is open
func (tr *TCP) TCPPortCheck(ctx context.Context) (bool, error) {
	dialer := net.Dialer{Timeout: tr.Timeout}
	tr.Start = time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", tr.URL.Host)
	tr.End = time.Now()
	tr.Duration = tr.End.Sub(tr.Start)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	return true, nil
}

// Check performs a TCP port check.
func (tr *TCP) Check(ctx context.Context) Result {
	ok, err := tr.TCPPortCheck(ctx)
	return Result{OK: ok, Duration: tr.Duration, Error: err}
}

// TLSPortCheck check if a scured tcp port is open
func (tr *TCP) TLSPortCheck(ctx context.Context) (bool, error) {
	cert, err := tls.LoadX509KeyPair(tr.CertFile, tr.PrivateKeyFile)
	if err != nil {
		return false, err
	}
	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}

	dialer := net.Dialer{Timeout: tr.Timeout}
	tr.Start = time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", tr.URL.Host)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &config)
	err = tlsConn.Handshake()
	if err != nil {
		return false, err
	}
	tr.End = time.Now()
	tr.Duration = tr.End.Sub(tr.Start)

	return true, nil
}
