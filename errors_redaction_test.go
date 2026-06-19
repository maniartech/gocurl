package gocurl_test

import (
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
