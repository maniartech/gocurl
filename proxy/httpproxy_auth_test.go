package proxy

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildProxyURL_UsernameOnly(t *testing.T) {
	hp := &HTTPProxy{Address: "proxy.example:8080", Username: "alice"}
	got, err := hp.buildProxyURL()
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "http://alice@proxy.example:8080" {
		t.Errorf("buildProxyURL = %q, want userinfo with empty password", got.String())
	}
}

func TestBuildProxyURL_UserAndPassword(t *testing.T) {
	hp := &HTTPProxy{Address: "proxy.example:8080", Username: "alice", Password: "s3cret"}
	got, err := hp.buildProxyURL()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.String(), "alice:s3cret@proxy.example:8080") {
		t.Errorf("buildProxyURL = %q, want user:pass@addr", got.String())
	}
}

func TestBuildProxyURL_NoCredentials(t *testing.T) {
	hp := &HTTPProxy{Address: "proxy.example:8080"}
	got, err := hp.buildProxyURL()
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "http://proxy.example:8080" {
		t.Errorf("buildProxyURL = %q, want no userinfo", got.String())
	}
}

// TestBuildProxyURL_CredentialsWithSpaces is the regression for the QueryEscape
// bug: a space in the username/password must round-trip to the literal value
// (QueryEscape encoded it as '+', which does NOT decode back in userinfo, so the
// proxy received corrupted credentials and the HTTP-proxy and CONNECT paths
// disagreed). url.UserPassword encodes space as %20 and decodes cleanly.
func TestBuildProxyURL_CredentialsWithSpaces(t *testing.T) {
	hp := &HTTPProxy{Address: "proxy.example:8080", Username: "user name", Password: "p ss"}
	got, err := hp.buildProxyURL()
	if err != nil {
		t.Fatal(err)
	}
	if u := got.User.Username(); u != "user name" {
		t.Errorf("username = %q, want %q (literal, no '+'-mangling)", u, "user name")
	}
	pw, _ := got.User.Password()
	if pw != "p ss" {
		t.Errorf("password = %q, want %q (literal, no '+'-mangling)", pw, "p ss")
	}
	// The HTTP-proxy path (via url.User) and the CONNECT path
	// (createConnectRequest, raw Username:Password) must agree on the wire.
	req := hp.createConnectRequest("target.example:443")
	decoded, derr := base64.StdEncoding.DecodeString(
		strings.TrimPrefix(req.Header.Get("Proxy-Authorization"), "Basic "))
	if derr != nil {
		t.Fatal(derr)
	}
	if string(decoded) != "user name:p ss" {
		t.Errorf("CONNECT credentials = %q, want %q (must match HTTP-proxy path)", decoded, "user name:p ss")
	}
}

// TestBuildProxyURL_MalformedAddressDoesNotLeakPassword is the regression for the
// proxy-URL parse-error leak: a malformed address must produce an error that
// contains only the address, never the credentials.
func TestBuildProxyURL_MalformedAddressDoesNotLeakPassword(t *testing.T) {
	hp := &HTTPProxy{Address: "pro xy:8080", Username: "alice", Password: "topSecretPw"}
	_, err := hp.buildProxyURL()
	if err == nil {
		t.Fatal("expected an error for a malformed proxy address")
	}
	if strings.Contains(err.Error(), "topSecretPw") {
		t.Errorf("proxy parse error leaked the password: %v", err)
	}
}

func TestCreateConnectRequest_UsernameOnly(t *testing.T) {
	hp := &HTTPProxy{Address: "proxy.example:8080", Username: "alice"}
	req := hp.createConnectRequest("target.example:443")
	got := req.Header.Get("Proxy-Authorization")
	if got == "" {
		t.Fatal("Proxy-Authorization must be set for username-only proxy auth")
	}
	const prefix = "Basic "
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("Proxy-Authorization = %q, want Basic scheme", got)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(got, prefix))
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "alice:" {
		t.Errorf("decoded credentials = %q, want %q (RFC 7617 empty password)", decoded, "alice:")
	}
}

func TestCreateConnectRequest_NoUsername(t *testing.T) {
	hp := &HTTPProxy{Address: "proxy.example:8080"}
	req := hp.createConnectRequest("target.example:443")
	if got := req.Header.Get("Proxy-Authorization"); got != "" {
		t.Errorf("Proxy-Authorization = %q, want empty when no username", got)
	}
}
