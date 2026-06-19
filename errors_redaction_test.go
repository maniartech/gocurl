package gocurl_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/maniartech/gocurl"
)

// TestRedactURL_RepeatedSensitiveParams is the regression for the
// single-occurrence redaction bug: every occurrence of a sensitive query
// parameter must be redacted, not just the first.
func TestRedactURL_RepeatedSensitiveParams(t *testing.T) {
	in := "http://h.example/p?api_key=FIRSTKEY&z=1&api_key=SECONDKEY&token=FIRSTTOK&q=2&token=SECONDTOK"
	out := gocurl.RedactURL(in)

	for _, leak := range []string{"FIRSTKEY", "SECONDKEY", "FIRSTTOK", "SECONDTOK"} {
		if strings.Contains(out, leak) {
			t.Errorf("RedactURL leaked %q: %s", leak, out)
		}
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("RedactURL produced no redaction: %s", out)
	}
}

func TestRedactURL_StripsUserinfo(t *testing.T) {
	out := gocurl.RedactURL("https://alice:s3cret@example.com/path")
	if strings.Contains(out, "s3cret") {
		t.Errorf("RedactURL leaked basic-auth password: %s", out)
	}
}

// TestRedactCommand_UserFlagForms covers all four -u/--user spellings; the
// password segment must always be redacted while the username is kept.
func TestRedactCommand_UserFlagForms(t *testing.T) {
	cases := []string{
		"curl -u alice:topSecretPw https://example.com",
		"curl --user alice:topSecretPw https://example.com",
		"curl --user=alice:topSecretPw https://example.com",
		"curl -u=alice:topSecretPw https://example.com",
	}
	for _, cmd := range cases {
		out := gocurl.RedactCommand(cmd)
		if strings.Contains(out, "topSecretPw") {
			t.Errorf("RedactCommand leaked password for %q: %s", cmd, out)
		}
		if !strings.Contains(out, "alice") {
			t.Errorf("RedactCommand dropped the username for %q: %s", cmd, out)
		}
		if !strings.Contains(out, "[REDACTED]") {
			t.Errorf("RedactCommand produced no redaction for %q: %s", cmd, out)
		}
	}
}

// TestRedactCommand_CookieFlag covers -b/--cookie; the cookie value is a session
// credential and must be redacted (the Cookie: header form was already handled).
func TestRedactCommand_CookieFlag(t *testing.T) {
	cases := []string{
		"curl -b SESSIONID=supersecretcookie https://example.com",
		"curl --cookie SESSIONID=supersecretcookie https://example.com",
		"curl --cookie=SESSIONID=supersecretcookie https://example.com",
	}
	for _, cmd := range cases {
		out := gocurl.RedactCommand(cmd)
		if strings.Contains(out, "supersecretcookie") {
			t.Errorf("RedactCommand leaked cookie value for %q: %s", cmd, out)
		}
		if !strings.Contains(out, "[REDACTED]") {
			t.Errorf("RedactCommand produced no redaction for %q: %s", cmd, out)
		}
	}
}

// TestIsSensitiveHeader_M7Additions locks in the canonical-set additions: the
// AWS STS session token (bearer-equivalent) and the CSRF token.
func TestIsSensitiveHeader_M7Additions(t *testing.T) {
	for _, h := range []string{"X-Amz-Security-Token", "x-amz-security-token", "X-CSRF-Token", "x-csrf-token"} {
		if !gocurl.IsSensitiveHeader(h) {
			t.Errorf("IsSensitiveHeader(%q) = false, want true", h)
		}
	}
}

// TestIsSensitiveHeader_VendorHeuristic covers the open set of vendor auth headers
// caught by the credential-suffix/content heuristic, and confirms benign headers
// are not over-redacted.
func TestIsSensitiveHeader_VendorHeuristic(t *testing.T) {
	for _, h := range []string{
		"X-Goog-Api-Key", "Private-Token", "X-Vault-Token", "X-Functions-Key",
		"X-Access-Token", "X-Gitlab-Token", "X-Auth-Key", "Apikey",
	} {
		if !gocurl.IsSensitiveHeader(h) {
			t.Errorf("IsSensitiveHeader(%q) = false, want true (vendor auth header)", h)
		}
	}
	for _, h := range []string{"Content-Type", "Accept", "User-Agent", "Content-Length", "Content-Encoding"} {
		if gocurl.IsSensitiveHeader(h) {
			t.Errorf("IsSensitiveHeader(%q) = true, want false (benign)", h)
		}
	}
}

// TestRedactCommand_FirstTokenFlag is the regression for the leading-space bug:
// redaction must fire even when the credential flag is the FIRST token (e.g. args
// joined without a leading space, as the CLI now does via ParseError).
func TestRedactCommand_FirstTokenFlag(t *testing.T) {
	cases := []struct{ cmd, leak string }{
		{"-u user:secretpw https://x", "secretpw"},
		{"--user user:secretpw https://x", "secretpw"},
		{"-u=user:secretpw https://x", "secretpw"},
		{"-b SESSIONID=supersecret https://x", "supersecret"},
		{"--cookie SESSIONID=supersecret https://x", "supersecret"},
	}
	for _, tc := range cases {
		out := gocurl.RedactCommand(tc.cmd)
		if strings.Contains(out, tc.leak) {
			t.Errorf("RedactCommand(%q) leaked %q: %s", tc.cmd, tc.leak, out)
		}
		if !strings.Contains(out, "[REDACTED]") {
			t.Errorf("RedactCommand(%q) produced no redaction: %s", tc.cmd, out)
		}
	}
}

// TestCurlArgsWithVars_ParseErrorRedactedAndTyped is the regression for the
// missed variadic-with-vars path: its parse failure must be a typed KindParse
// error with credentials (userinfo + sensitive query params) redacted.
func TestCurlArgsWithVars_ParseErrorRedactedAndTyped(t *testing.T) {
	_, err := gocurl.CurlArgsWithVars(context.Background(), gocurl.Variables{},
		"https://first/", "https://bob:hunter2@second/?token=abc123")
	if err == nil {
		t.Fatal("expected a parse error for two positional URLs")
	}
	if strings.Contains(err.Error(), "hunter2") || strings.Contains(err.Error(), "abc123") {
		t.Errorf("CurlArgsWithVars leaked credentials in error: %v", err)
	}
	if !errors.Is(err, gocurl.ErrParse) {
		t.Errorf("CurlArgsWithVars parse failure should be KindParse, got: %v", err)
	}
}
