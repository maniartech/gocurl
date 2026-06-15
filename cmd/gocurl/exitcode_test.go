package main

import (
	"errors"
	"testing"

	"github.com/maniartech/gocurl"
)

// TestGetExitCode_Kind verifies the Kind-based curl-compatible exit codes.
func TestGetExitCode_Kind(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"server status", gocurl.ServerStatusError("http://h", 404), 22},
		{"timeout", gocurl.TimeoutError("http://h", nil), 28},
		{"connect", gocurl.ConnectError("http://h", nil), 7},
		{"tls", gocurl.TLSError("http://h", nil), 35},
		{"parse", gocurl.ParseError("curl", errors.New("bad")), 2},
		{"validation", gocurl.ValidationError("URL", errors.New("required")), 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := getExitCode(tc.err); got != tc.want {
				t.Errorf("getExitCode(%s) = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}
