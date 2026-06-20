package libtower

import (
	"context"
	"net/http"
	"time"
)

// Check performs an HTTP status check.
func (hsr *HTTP) Check(ctx context.Context) Result {
	err := hsr.HTTPStatusContext(ctx)
	return Result{OK: err == nil, Duration: hsr.Duration, Error: err}
}

// HTTPStatus check
func (hsr *HTTP) HTTPStatus() error {
	return hsr.HTTPStatusContext(context.Background())
}

// HTTPStatusContext performs an HTTP request, respecting ctx cancellation.
func (hsr *HTTP) HTTPStatusContext(ctx context.Context) error {
	// setup client
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, hsr.Method, hsr.URL, nil)
	if err != nil {
		return err
	}
	hsr.Start = time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	hsr.End = time.Now()

	hsr.Duration = hsr.End.Sub(hsr.Start)
	hsr.StatusCode = resp.StatusCode
	// TODO : add response body
	hsr.Status, hsr.Proto, hsr.ProtoMajor, hsr.ProtoMinor, hsr.Header, hsr.ContentLength, hsr.TransferEncoding, hsr.Close, hsr.Uncompressed, hsr.Trailer =
		resp.Status, resp.Proto, resp.ProtoMajor, resp.ProtoMinor, resp.Header, resp.ContentLength, resp.TransferEncoding, resp.Close, resp.Uncompressed, resp.Trailer

	return err
}
