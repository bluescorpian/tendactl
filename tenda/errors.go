package tenda

import (
	"errors"
	"fmt"
)

var (
	ErrBadPassword  = errors.New("incorrect router password")
	ErrNoPassword   = errors.New("no router password")
	ErrSessionLost  = errors.New("router session rejected after re-login")
	ErrUnsupported  = errors.New("endpoint not supported by this firmware")
	ErrNotConfirmed = errors.New("hazardous request not confirmed")
)

// APIError is a router reply that reports failure: a non-zero errCode or
// err_code, or a native form redirected to "<page>?<n>".
type APIError struct {
	Endpoint string
	Field    string // "errCode", "err_code" or "redirect"
	Code     int
	Body     []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: router returned %s %d", e.Endpoint, e.Field, e.Code)
}

// UnsupportedError is the router's "Form X is not defined" page: the handler
// is compiled out of this firmware.
type UnsupportedError struct {
	Endpoint string
	Form     string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("%s: %v (Form %s is not defined)", e.Endpoint, ErrUnsupported, e.Form)
}

func (e *UnsupportedError) Is(target error) bool { return target == ErrUnsupported }

// HazardError is a request on the "Never call casually" list that the
// ConfirmFunc refused, or that had no ConfirmFunc. Nothing was sent.
type HazardError struct {
	Hazard Hazard
	Err    error
}

func (e *HazardError) Error() string {
	return fmt.Sprintf("refusing %s (%s): %v", e.Hazard.Endpoint, e.Hazard.Reason, e.Err)
}

func (e *HazardError) Is(target error) bool { return target == ErrNotConfirmed }
func (e *HazardError) Unwrap() error        { return e.Err }

// DecodeError is a reply that did not have the expected shape.
type DecodeError struct {
	Endpoint string
	Body     []byte
	Err      error
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("%s: unexpected reply: %v", e.Endpoint, e.Err)
}

func (e *DecodeError) Unwrap() error { return e.Err }

// HTTPError is an unexpected HTTP status.
type HTTPError struct {
	Endpoint string
	Status   int
	Snippet  string
}

func (e *HTTPError) Error() string {
	s := fmt.Sprintf("%s: HTTP %d", e.Endpoint, e.Status)
	if e.Snippet != "" {
		s += ": " + e.Snippet
	}
	return s
}

// mapCode wraps an *APIError carrying code with target, so callers can test
// errors.Is(err, target). Other errors pass through unchanged.
func mapCode(err error, code int, target error) error {
	var ae *APIError
	if errors.As(err, &ae) && ae.Code == code {
		return fmt.Errorf("%w: %w", target, err)
	}
	return err
}
