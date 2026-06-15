package gocurl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
)

// fakeNetErr is a synthetic net.Error for classification tests.
type fakeNetErr struct{ timeout bool }

func (f fakeNetErr) Error() string   { return "fake net error" }
func (f fakeNetErr) Timeout() bool   { return f.timeout }
func (f fakeNetErr) Temporary() bool { return false }

func TestKind_String(t *testing.T) {
	cases := map[Kind]string{
		KindUnknown:        "unknown",
		KindParse:          "parse",
		KindValidation:     "validation",
		KindConnect:        "connect",
		KindTLS:            "tls",
		KindTimeout:        "timeout",
		KindCanceled:       "canceled",
		KindServerStatus:   "server status",
		KindRetryExhausted: "retry exhausted",
		KindBodyRead:       "body read",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", int(k), got, want)
		}
	}
}

func TestClassifyTransportError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want Kind
	}{
		{"nil", nil, KindUnknown},
		{"canceled", context.Canceled, KindCanceled},
		{"deadline", context.DeadlineExceeded, KindTimeout},
		{"wrapped deadline", fmt.Errorf("boom: %w", context.DeadlineExceeded), KindTimeout},
		{"net timeout", fakeNetErr{timeout: true}, KindTimeout},
		{"dial opError", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, KindConnect},
		{"dns error", &net.DNSError{Err: "no such host", Name: "x"}, KindConnect},
		{"tls cert verify", &tls.CertificateVerificationError{Err: errors.New("bad cert")}, KindTLS},
		{"x509 string", errors.New("x509: certificate signed by unknown authority"), KindTLS},
		{"cert pin string", errors.New("certificate pin verification failed: no matching fingerprint found"), KindTLS},
		{"unknown", errors.New("something else"), KindUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyTransportError(tc.err); got != tc.want {
				t.Errorf("classifyTransportError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestGocurlError_Classification(t *testing.T) {
	cases := []struct {
		name      string
		err       *GocurlError
		timeout   bool
		temporary bool
		retryable bool
	}{
		{"connect", ConnectError("http://h", errors.New("refused")).(*GocurlError), false, true, true},
		{"timeout", TimeoutError("http://h", context.DeadlineExceeded).(*GocurlError), true, true, true},
		{"tls", TLSError("http://h", errors.New("bad cert")).(*GocurlError), false, false, false},
		{"canceled", CanceledError("http://h", context.Canceled).(*GocurlError), false, false, false},
		{"validation", ValidationError("URL", errors.New("required")).(*GocurlError), false, false, false},
		{"parse", ParseError("curl", errors.New("bad")).(*GocurlError), false, false, false},
		{"status 503 retryable", ServerStatusError("http://h", 503).(*GocurlError), false, true, true},
		{"status 404 not retryable", ServerStatusError("http://h", 404).(*GocurlError), false, false, false},
		{"body read", BodyReadError("http://h", errors.New("eof")).(*GocurlError), false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Timeout(); got != tc.timeout {
				t.Errorf("Timeout() = %v, want %v", got, tc.timeout)
			}
			if got := tc.err.Temporary(); got != tc.temporary {
				t.Errorf("Temporary() = %v, want %v", got, tc.temporary)
			}
			if got := tc.err.Retryable(); got != tc.retryable {
				t.Errorf("Retryable() = %v, want %v", got, tc.retryable)
			}
		})
	}
}

func TestGocurlError_NilSafety(t *testing.T) {
	var e *GocurlError
	if e.Timeout() || e.Temporary() || e.Retryable() {
		t.Error("nil *GocurlError classification methods must all be false")
	}
}

func TestSentinels_ErrorsIs(t *testing.T) {
	if !errors.Is(TimeoutError("http://h", nil), ErrTimeout) {
		t.Error("timeout error should match ErrTimeout")
	}
	if !errors.Is(ConnectError("http://h", nil), ErrConnect) {
		t.Error("connect error should match ErrConnect")
	}
	if !errors.Is(ServerStatusError("http://h", 500), ErrServerStatus) {
		t.Error("server-status error should match ErrServerStatus")
	}
	if errors.Is(TimeoutError("http://h", nil), ErrConnect) {
		t.Error("timeout error must not match ErrConnect")
	}
}

func TestErrorsIs_ContextThroughUnwrap(t *testing.T) {
	err := TimeoutError("http://h", context.DeadlineExceeded)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("errors.Is should resolve context.DeadlineExceeded through Unwrap")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Error("errors.Is should match ErrTimeout sentinel")
	}
}

func TestRetryExhausted_Chaining(t *testing.T) {
	inner := classifyToError(fmt.Errorf("attempt failed: %w", context.DeadlineExceeded))
	err := RetryError("http://h", 3, inner)

	if KindOf(err) != KindRetryExhausted {
		t.Errorf("KindOf = %v, want KindRetryExhausted", KindOf(err))
	}
	var ge *GocurlError
	if !errors.As(err, &ge) || ge.Attempt != 3 {
		t.Errorf("expected Attempt=3, got %+v", ge)
	}
	// The last attempt timed out, so chain-walking helpers still resolve.
	if !errors.Is(err, ErrTimeout) {
		t.Error("errors.Is(retryExhausted, ErrTimeout) should be true via the chain")
	}
	if !IsTimeout(err) {
		t.Error("IsTimeout should walk the chain to the timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("errors.Is(retryExhausted, context.DeadlineExceeded) should be true")
	}
}

func TestPackageHelpers(t *testing.T) {
	if KindOf(errors.New("plain")) != KindUnknown {
		t.Error("KindOf(non-gocurl) should be KindUnknown")
	}
	if !IsRetryable(ConnectError("http://h", nil)) {
		t.Error("IsRetryable(connect) should be true")
	}
	if IsRetryable(TLSError("http://h", nil)) {
		t.Error("IsRetryable(tls) should be false")
	}
	if !IsTemporary(ServerStatusError("http://h", 503)) {
		t.Error("IsTemporary(503) should be true")
	}
	if IsTemporary(ServerStatusError("http://h", 400)) {
		t.Error("IsTemporary(400) should be false")
	}
	if !IsTimeout(TimeoutError("http://h", nil)) {
		t.Error("IsTimeout(timeout) should be true")
	}
}

func TestScrubErrorString_Direct(t *testing.T) {
	in := `Get "https://user:s3cr3tpw@host/p?api_key=SUPERSECRET": dial`
	out := scrubErrorString(in)
	if strings.Contains(out, "s3cr3tpw") {
		t.Errorf("userinfo password not redacted: %q", out)
	}
	if strings.Contains(out, "SUPERSECRET") {
		t.Errorf("api_key not redacted: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("expected [REDACTED] markers: %q", out)
	}
}

func TestError_ScrubsWrappedCredentials(t *testing.T) {
	// A wrapped stdlib error embeds a full URL with credentials; the Error()
	// backstop must scrub them from the final string.
	wrapped := fmt.Errorf(`Get "https://bob:hunter2@host/p?token=abc123def": EOF`)
	err := RequestError("http://h", wrapped)
	s := err.Error()
	if strings.Contains(s, "hunter2") {
		t.Errorf("password leaked through Error(): %q", s)
	}
	if strings.Contains(s, "abc123def") {
		t.Errorf("token leaked through Error(): %q", s)
	}
}

func TestError_BackCompatShapeAndStatus(t *testing.T) {
	// Struct-literal (no Kind) keeps the legacy message shape.
	legacy := &GocurlError{Op: "request", URL: "https://example.com/api", Err: errors.New("connection refused")}
	s := legacy.Error()
	for _, want := range []string{"request", "https://example.com/api", "connection refused"} {
		if !strings.Contains(s, want) {
			t.Errorf("legacy Error() %q missing %q", s, want)
		}
	}
	// Server-status appends the code to the op segment.
	ss := ServerStatusError("https://example.com", 503).Error()
	if !strings.Contains(ss, "server status (503)") {
		t.Errorf("server-status Error() = %q, want it to contain 'server status (503)'", ss)
	}
}

func TestClassifyToError(t *testing.T) {
	if classifyToError(nil) != nil {
		t.Error("classifyToError(nil) should be nil")
	}
	// Already-typed errors pass through unchanged.
	orig := ConnectError("http://h", nil)
	if got := classifyToError(orig); got != orig {
		t.Error("classifyToError should pass through an existing GocurlError")
	}
	// Raw transport errors get classified, without a URL.
	got := classifyToError(context.DeadlineExceeded)
	var ge *GocurlError
	if !errors.As(got, &ge) || ge.Kind != KindTimeout || ge.URL != "" {
		t.Errorf("classifyToError(deadline) = %+v, want KindTimeout with no URL", ge)
	}
}

func TestSentinel_ErrorString(t *testing.T) {
	if !strings.Contains(ErrTimeout.Error(), "timeout") {
		t.Errorf("sentinel Error() = %q", ErrTimeout.Error())
	}
}

func TestTemporary_RetryExhaustedRecursion(t *testing.T) {
	// A retry-exhausted error wrapping a temporary inner is itself temporary.
	inner := ServerStatusError("http://h", 503) // retryable/temporary
	err := RetryError("http://h", 2, inner)
	if !IsTemporary(err) {
		t.Error("retry-exhausted wrapping a 503 should be temporary via recursion")
	}
	// Wrapping a non-temporary inner is not temporary.
	err2 := RetryError("http://h", 2, TLSError("http://h", nil))
	if IsTemporary(err2) {
		t.Error("retry-exhausted wrapping a TLS error should not be temporary")
	}
}

func TestHelpers_NonGocurlError(t *testing.T) {
	plain := errors.New("plain")
	if IsTemporary(plain) || IsRetryable(plain) || IsTimeout(plain) {
		t.Error("classification helpers must be false for a non-gocurl error")
	}
}

func TestTimeout_WrappedNetError(t *testing.T) {
	// Kind is not KindTimeout, but the wrapped net.Error reports a timeout.
	e := &GocurlError{Kind: KindConnect, Err: fakeNetErr{timeout: true}}
	if !e.Timeout() {
		t.Error("Timeout() should consult a wrapped net.Error")
	}
}

func TestWrapTransportError(t *testing.T) {
	if wrapTransportError("http://h", nil) != nil {
		t.Error("wrapTransportError(nil) should be nil")
	}
	// Already-typed (e.g. retry-exhausted) passes through.
	typed := RetryError("http://h", 2, nil)
	if got := wrapTransportError("http://h", typed); got != typed {
		t.Error("wrapTransportError should pass through a typed error")
	}
	// Raw connect error is classified.
	raw := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	got := wrapTransportError("http://h", raw)
	if KindOf(got) != KindConnect {
		t.Errorf("wrapTransportError(dial) Kind = %v, want KindConnect", KindOf(got))
	}
}
