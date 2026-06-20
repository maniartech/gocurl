package gocurl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileLimited(t *testing.T) {
	dir := t.TempDir()
	small := filepath.Join(dir, "small")
	if err := os.WriteFile(small, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(dir, "big")
	if err := os.WriteFile(big, make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}

	if b, err := readFileLimited(small, 10); err != nil || string(b) != "hello" {
		t.Errorf("under-cap read = %q, %v; want \"hello\", nil", b, err)
	}
	if _, err := readFileLimited(big, 10); err == nil {
		t.Error("over-cap read should error")
	}
	if _, err := readFileLimited(big, 100); err != nil {
		t.Errorf("exactly-at-cap read should succeed, got %v", err)
	}
	if _, err := readFileLimited(filepath.Join(dir, "missing"), 10); err == nil {
		t.Error("missing file should error")
	}
}

// TestParseTimeFileReadCaps confirms the convert-time @file / cookie-file reads
// fail closed past their cap (DoS guard for untrusted curl strings) while normal
// small files still parse. Caps are lowered so the test stays cheap.
func TestParseTimeFileReadCaps(t *testing.T) {
	dir := t.TempDir()

	dataFile := filepath.Join(dir, "data")
	if err := os.WriteFile(dataFile, make([]byte, 64), 0o600); err != nil {
		t.Fatal(err)
	}
	cookieFile := filepath.Join(dir, "cookies")
	if err := os.WriteFile(cookieFile, []byte("session=abc; theme=dark"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("data over cap errors", func(t *testing.T) {
		old := maxDataFileBytes
		maxDataFileBytes = 16
		defer func() { maxDataFileBytes = old }()
		_, err := ArgsToOptions([]string{"-d", "@" + dataFile, "http://x"})
		if err == nil || !strings.Contains(err.Error(), "parse-time read limit") {
			t.Fatalf("over-cap -d @file should fail closed with a read-limit error, got %v", err)
		}
	})
	t.Run("data under cap works", func(t *testing.T) {
		opts, err := ArgsToOptions([]string{"-d", "@" + dataFile, "http://x"})
		if err != nil {
			t.Fatalf("under-cap -d @file should parse, got %v", err)
		}
		if len(opts.Body) != 64 {
			t.Errorf("body length = %d, want 64", len(opts.Body))
		}
	})

	t.Run("cookie over cap errors", func(t *testing.T) {
		old := maxCookieFileBytes
		maxCookieFileBytes = 8
		defer func() { maxCookieFileBytes = old }()
		_, err := ArgsToOptions([]string{"-b", cookieFile, "http://x"})
		if err == nil || !strings.Contains(err.Error(), "parse-time read limit") {
			t.Fatalf("over-cap -b cookiefile should fail closed, got %v", err)
		}
	})
	t.Run("cookie under cap works", func(t *testing.T) {
		if _, err := ArgsToOptions([]string{"-b", cookieFile, "http://x"}); err != nil {
			t.Fatalf("under-cap -b cookiefile should parse, got %v", err)
		}
	})
}
