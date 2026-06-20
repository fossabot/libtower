package libtower

import (
	"context"
	"crypto/tls"
	"net"
	"time"
)

// Check performs a TLS certificate check.
func (hs *HTTPS) Check(ctx context.Context) Result {
	ok, notAfter, err := hs.HTTPSCheck(ctx)
	var data CheckData
	if !notAfter.IsZero() {
		data = CertData{NotAfter: notAfter}
	}
	r := Result{OK: ok, Duration: hs.Duration, Data: data, Error: err}
	if r.OK && hs.WarnIfExpiringWithin > 0 && time.Until(notAfter) < hs.WarnIfExpiringWithin {
		r.Warning = &CertExpiringWarning{NotAfter: notAfter, Remaining: time.Until(notAfter)}
	}
	return r
}

// CertExpiringWarning is returned as Result.Warning when a certificate is expiring soon.
type CertExpiringWarning struct {
	NotAfter  time.Time
	Remaining time.Duration
}

func (w *CertExpiringWarning) Error() string {
	return "certificate expires " + w.NotAfter.Format(time.RFC3339) +
		" (" + w.Remaining.Round(time.Second).String() + " remaining)"
}

const DefaultHTTPSPort = "443"

// HTTPS type
type HTTPS struct {
	Host    string
	Port    string
	Timeout time.Duration

	InsecureSkipVerify bool

	// WarnIfExpiringWithin, when > 0, triggers a Result.Warning if the
	// earliest certificate NotAfter falls within this window from now.
	WarnIfExpiringWithin time.Duration

	Start    time.Time
	End      time.Time
	Duration time.Duration
}

// HTTPSCheck checks tls certificate is valid
func (hs *HTTPS) HTTPSCheck(ctx context.Context) (bool, time.Time, error) {
	if hs.Port == "" {
		hs.Port = DefaultHTTPSPort
	}
	address := hs.Host + ":" + hs.Port
	dialer := net.Dialer{Timeout: hs.Timeout}

	hs.Start = time.Now()
	rawConn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		hs.End = time.Now()
		hs.Duration = hs.End.Sub(hs.Start)
		return false, time.Time{}, err
	}

	tlsCfg := &tls.Config{InsecureSkipVerify: hs.InsecureSkipVerify, ServerName: hs.Host}
	conn := tls.Client(rawConn, tlsCfg)
	err = conn.Handshake()
	hs.End = time.Now()
	hs.Duration = hs.End.Sub(hs.Start)
	if err != nil {
		rawConn.Close()
		return false, time.Time{}, err
	}
	defer conn.Close()
	var NotAfter = conn.ConnectionState().PeerCertificates[0].NotAfter
	for _, cert := range conn.ConnectionState().PeerCertificates {
		if cert.NotAfter.Before(NotAfter) {
			NotAfter = cert.NotAfter
		}
	}
	return true, NotAfter, nil
}
