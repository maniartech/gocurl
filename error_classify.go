package gocurl

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"regexp"
	"strings"
)

// Kind classifies a GocurlError. It is the machine-readable discriminator that
// callers branch on instead of string-matching error messages; the legacy Op
// string is retained for backward-compatible message output.
//
// See specs/08-error-model.md.
type Kind int

const (
	// KindUnknown is the zero value: an error that was not (or could not be)
	// classified. A struct-literal GocurlError built without a constructor
	// reports KindUnknown.
	KindUnknown Kind = iota
	// KindParse — tokenize/convert of a curl command failed.
	KindParse
	// KindValidation — validateRequestOptions rejected the prepared request.
	KindValidation
	// KindConnect — dial / connection refused / DNS / proxy connect failure.
	KindConnect
	// KindTLS — handshake, certificate verification, or pin mismatch.
	KindTLS
	// KindTimeout — a deadline was exceeded (connect, per-attempt, or overall).
	KindTimeout
	// KindCanceled — the context was canceled by the caller.
	KindCanceled
	// KindServerStatus — a 4xx/5xx response surfaced as an error (opt-in only,
	// via WithFailOnStatus / -f).
	KindServerStatus
	// KindRetryExhausted — all retry attempts failed.
	KindRetryExhausted
	// KindBodyRead — reading/decoding the response body failed (incl. over-limit).
	KindBodyRead
)

// String returns the lowercase, human-readable name of the kind. It also serves
// as the default Op segment for errors built from the typed constructors.
func (k Kind) String() string {
	switch k {
	case KindParse:
		return "parse"
	case KindValidation:
		return "validation"
	case KindConnect:
		return "connect"
	case KindTLS:
		return "tls"
	case KindTimeout:
		return "timeout"
	case KindCanceled:
		return "canceled"
	case KindServerStatus:
		return "server status"
	case KindRetryExhausted:
		return "retry exhausted"
	case KindBodyRead:
		return "body read"
	default:
		return "unknown"
	}
}

// kindSentinel is the value behind the package-level Err* sentinels. It carries a
// Kind so errors.Is(err, ErrTimeout) can match any GocurlError of that kind in
// the chain (see GocurlError.Is).
type kindSentinel struct{ kind Kind }

func (k *kindSentinel) Error() string { return "gocurl: " + k.kind.String() }

// Sentinel errors for use with errors.Is. They never wrap anything; they exist
// purely so callers can write errors.Is(err, gocurl.ErrTimeout) without
// importing or type-asserting the concrete type.
var (
	ErrParse          error = &kindSentinel{KindParse}
	ErrValidation     error = &kindSentinel{KindValidation}
	ErrConnect        error = &kindSentinel{KindConnect}
	ErrTLS            error = &kindSentinel{KindTLS}
	ErrTimeout        error = &kindSentinel{KindTimeout}
	ErrCanceled       error = &kindSentinel{KindCanceled}
	ErrServerStatus   error = &kindSentinel{KindServerStatus}
	ErrRetryExhausted error = &kindSentinel{KindRetryExhausted}
	ErrBodyRead       error = &kindSentinel{KindBodyRead}
)

// KindOf returns the Kind of the first (outermost) GocurlError in err's chain,
// or KindUnknown if there is none.
func KindOf(err error) Kind {
	var ge *GocurlError
	if errors.As(err, &ge) {
		return ge.Kind
	}
	return KindUnknown
}

// IsTimeout reports whether err (anywhere in its chain) was caused by a
// deadline/timeout. It matches a KindTimeout GocurlError as well as a bare
// context.DeadlineExceeded, so it resolves through a retry-exhausted wrapper.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout) || errors.Is(err, context.DeadlineExceeded)
}

// IsTemporary reports whether the outermost GocurlError considers the failure
// plausibly transient. Advisory only — not a retry decision (see IsRetryable).
func IsTemporary(err error) bool {
	var ge *GocurlError
	if errors.As(err, &ge) {
		return ge.Temporary()
	}
	return false
}

// IsRetryable reports whether gocurl's resilience layer considers the outermost
// GocurlError safe to retry for an idempotent request.
func IsRetryable(err error) bool {
	var ge *GocurlError
	if errors.As(err, &ge) {
		return ge.Retryable()
	}
	return false
}

// classifyTransportError inspects a transport-layer error (typically from
// client.Do) and returns the most specific Kind. The order matters: context
// outcomes first, then timeouts, then TLS, then connect/DNS.
func classifyTransportError(err error) Kind {
	if err == nil {
		return KindUnknown
	}
	switch {
	case errors.Is(err, context.Canceled):
		return KindCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return KindTimeout
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return KindTimeout
	}

	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return KindTLS
	}
	var recErr tls.RecordHeaderError
	if errors.As(err, &recErr) {
		return KindTLS
	}
	if isTLSError(err) {
		return KindTLS
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return KindConnect
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return KindConnect
	}

	return KindUnknown
}

// isTLSError is a string-level backstop for TLS failures that do not surface as
// a typed *tls.CertificateVerificationError — notably a VerifyPeerCertificate
// callback rejection (certificate pinning) which propagates as a plain error.
func isTLSError(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "x509:") ||
		strings.Contains(s, "tls:") ||
		strings.Contains(s, "certificate")
}

// classifyKind returns the Kind of err, classifying a raw (not-yet-wrapped)
// transport error when the chain has not produced a typed GocurlError. This is
// used by instrumentation, which runs inside the middleware chain — before
// wrapTransportError classifies the error at the Client.Do boundary.
func classifyKind(err error) Kind {
	if k := KindOf(err); k != KindUnknown {
		return k
	}
	return classifyTransportError(err)
}

// classifyToError wraps a transport error into a typed GocurlError carrying its
// classified Kind (without a URL — used as the inner error of a retry-exhausted
// wrapper, which supplies the URL). Errors already typed pass through unchanged.
func classifyToError(err error) error {
	if err == nil {
		return nil
	}
	var ge *GocurlError
	if errors.As(err, &ge) {
		return err
	}
	k := classifyTransportError(err)
	return &GocurlError{Op: k.String(), Kind: k, Err: err}
}

// wrapTransportError normalizes a transport/execution error returned from the
// request engine into a typed GocurlError. Errors that are already typed (e.g. a
// retry-exhausted error from the retry loop, or a validation error) pass through
// unchanged; everything else is classified via RequestError.
func wrapTransportError(url string, err error) error {
	if err == nil {
		return nil
	}
	var ge *GocurlError
	if errors.As(err, &ge) {
		return err
	}
	return RequestError(url, err)
}

// userinfoCredRe matches a "scheme://user:password@" segment so the password can
// be redacted from error strings that embed a full URL (e.g. Go's *url.Error).
var userinfoCredRe = regexp.MustCompile(`://([^/@\s:]+):[^@/\s]+@`)

// sensitiveParamRe matches sensitive query parameters anywhere in a string. The
// alternatives are ordered longest-first so api_key/apikey win over key. Unlike
// redactURLParams it redacts EVERY occurrence, which matters because a wrapped
// stdlib error often repeats the URL.
var sensitiveParamRe = regexp.MustCompile(`(?i)(api_key|apikey|secret|password|token|key)=[^&\s"'<>]+`)

// scrubErrorString is the final-pass backstop run inside GocurlError.Error() on
// the joined message. It redacts credentials that may leak in from wrapped
// stdlib errors (sensitive query params and userinfo passwords).
func scrubErrorString(msg string) string {
	msg = sensitiveParamRe.ReplaceAllString(msg, "$1=[REDACTED]")
	msg = userinfoCredRe.ReplaceAllString(msg, "://$1:[REDACTED]@")
	return msg
}
