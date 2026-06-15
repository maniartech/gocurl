package tests

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/maniartech/gocurl"
)

// sizeServer records how many body bytes it received.
type sizeServer struct {
	*httptest.Server
	mu       sync.Mutex
	received int64
	body     []byte
}

func newSizeServer(t *testing.T, keepBody bool) *sizeServer {
	t.Helper()
	s := &sizeServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if keepBody {
			b, _ := io.ReadAll(r.Body)
			s.mu.Lock()
			s.received, s.body = int64(len(b)), b
			s.mu.Unlock()
		} else {
			n, _ := io.Copy(io.Discard, r.Body)
			s.mu.Lock()
			s.received = n
			s.mu.Unlock()
		}
		fmt.Fprintf(w, "got %d", s.received)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestStreaming_FileUpload(t *testing.T) {
	payload := strings.Repeat("0123456789", 200000) // ~2MB
	p := filepath.Join(t.TempDir(), "big.bin")
	if err := os.WriteFile(p, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := newSizeServer(t, false)

	c, _ := gocurl.New()
	defer c.Close()
	req, err := gocurl.NewRequest("POST", srv.URL, gocurl.Stream(gocurl.FileBody(p)))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if srv.received != int64(len(payload)) {
		t.Errorf("server received %d bytes, want %d", srv.received, len(payload))
	}
}

func TestStreaming_UploadFileFlag(t *testing.T) {
	p := filepath.Join(t.TempDir(), "up.txt")
	if err := os.WriteFile(p, []byte("upload-via-T"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := newSizeServer(t, true)

	c, _ := gocurl.New()
	defer c.Close()
	// Forward slashes so the path survives shell-style tokenization on Windows.
	cmd := "curl -T " + filepath.ToSlash(p) + " " + srv.URL
	if _, _, err := c.CurlString(context.Background(), cmd); err != nil {
		t.Fatal(err)
	}
	if string(srv.body) != "upload-via-T" {
		t.Errorf("server body = %q", srv.body)
	}
}

func TestStreaming_MultipartUpload(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(p, []byte("CONTENT"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotField, gotFileName, gotFileBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		gotField = r.FormValue("field")
		f, hdr, err := r.FormFile("doc")
		if err == nil {
			defer f.Close()
			gotFileName = hdr.Filename
			b, _ := io.ReadAll(f)
			gotFileBody = string(b)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c, _ := gocurl.New()
	defer c.Close()
	req, err := gocurl.NewRequest("POST", srv.URL, gocurl.Stream(
		gocurl.MultipartBody(
			map[string]string{"field": "hello"},
			gocurl.MultipartFile{Field: "doc", FileName: "a.txt", Path: p},
		),
	))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if gotField != "hello" || gotFileName != "a.txt" || gotFileBody != "CONTENT" {
		t.Errorf("multipart mismatch: field=%q file=%q body=%q", gotField, gotFileName, gotFileBody)
	}
}

func TestStreaming_NonRewindableReaderUpload(t *testing.T) {
	srv := newSizeServer(t, true)
	c, _ := gocurl.New()
	defer c.Close()
	// A non-rewindable reader body works on the default (no-retry) client.
	req, err := gocurl.NewRequest("POST", srv.URL, gocurl.Stream(gocurl.ReaderBody(strings.NewReader("stream-once"))))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if string(srv.body) != "stream-once" {
		t.Errorf("server body = %q", srv.body)
	}
}

func TestStreaming_ChunkedResponseConsumption(t *testing.T) {
	const lines = 20
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Error("server does not support flushing")
		}
		for i := 0; i < lines; i++ {
			fmt.Fprintf(w, "line-%d\n", i)
			if ok {
				fl.Flush()
			}
		}
	}))
	defer srv.Close()

	c, _ := gocurl.New()
	defer c.Close()
	resp, err := c.Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Read the live stream incrementally.
	count := 0
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if !strings.HasPrefix(sc.Text(), "line-") {
			t.Errorf("unexpected line: %q", sc.Text())
		}
		count++
	}
	if count != lines {
		t.Errorf("read %d lines, want %d", count, lines)
	}
}
