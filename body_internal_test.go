package gocurl

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/gocurl/options"
)

func readAllClose(t *testing.T, rc io.ReadCloser) string {
	t.Helper()
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}

func TestBytesAndStringBody(t *testing.T) {
	for _, bs := range []options.BodySource{BytesBody([]byte("payload")), StringBody("payload")} {
		if !bs.Rewindable() {
			t.Error("bytes/string body should be rewindable")
		}
		if n, ok := bs.Len(); !ok || n != 7 {
			t.Errorf("Len = %d,%v want 7,true", n, ok)
		}
		// Rewindable: two independent opens.
		rc1, _ := bs.Open()
		rc2, _ := bs.Open()
		if readAllClose(t, rc1) != "payload" || readAllClose(t, rc2) != "payload" {
			t.Error("content mismatch across opens")
		}
	}
}

func TestFileBody(t *testing.T) {
	p := filepath.Join(t.TempDir(), "up.txt")
	if err := os.WriteFile(p, []byte("file-contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	fb := FileBody(p)
	if !fb.Rewindable() {
		t.Error("file body should be rewindable")
	}
	if n, ok := fb.Len(); !ok || n != int64(len("file-contents")) {
		t.Errorf("Len = %d,%v", n, ok)
	}
	rc, err := fb.Open()
	if err != nil {
		t.Fatal(err)
	}
	if readAllClose(t, rc) != "file-contents" {
		t.Error("content mismatch")
	}

	// Missing file: Open errors, Len reports unknown.
	mb := FileBody(filepath.Join(t.TempDir(), "missing"))
	if _, err := mb.Open(); err == nil {
		t.Error("expected open error for missing file")
	}
	if _, ok := mb.Len(); ok {
		t.Error("Len should be unknown for missing file")
	}
}

func TestReaderBody(t *testing.T) {
	rb := ReaderBody(strings.NewReader("once"))
	if rb.Rewindable() {
		t.Error("reader body must not be rewindable")
	}
	if _, ok := rb.Len(); ok {
		t.Error("reader body length should be unknown")
	}
	rc, err := rb.Open()
	if err != nil {
		t.Fatal(err)
	}
	if readAllClose(t, rc) != "once" {
		t.Error("content mismatch")
	}
	if _, err := rb.Open(); err == nil {
		t.Error("second Open of a reader body must error")
	}
}

func TestMultipartBody_RoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(p, []byte("FILE"), 0o600); err != nil {
		t.Fatal(err)
	}
	mb := MultipartBody(map[string]string{"field": "value"},
		MultipartFile{Field: "doc", FileName: "a.txt", Path: p})

	if !mb.Rewindable() {
		t.Error("path-based multipart should be rewindable")
	}
	ct := mb.(options.ContentTyper).ContentType()
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("content type %q: %v", ct, err)
	}

	rc, err := mb.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	mr := multipart.NewReader(rc, params["boundary"])
	form, err := mr.ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	if form.Value["field"][0] != "value" {
		t.Errorf("field = %v", form.Value["field"])
	}
	if len(form.File["doc"]) != 1 || form.File["doc"][0].Filename != "a.txt" {
		t.Errorf("file part missing: %+v", form.File)
	}
}

func TestMultipartBody_ReaderNotRewindable(t *testing.T) {
	mb := MultipartBody(nil, MultipartFile{Field: "f", FileName: "x", Reader: strings.NewReader("data")})
	if mb.Rewindable() {
		t.Error("reader-backed multipart must not be rewindable")
	}
}

func TestMultipartBody_MissingFileErrors(t *testing.T) {
	mb := MultipartBody(nil, MultipartFile{Field: "f", FileName: "x", Path: filepath.Join(t.TempDir(), "missing")})
	rc, err := mb.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, readErr := io.ReadAll(rc)
	rc.Close()
	if readErr == nil {
		t.Error("expected an error reading a multipart body whose file is missing")
	}
}

func TestReaderBody_ReadCloserPassthrough(t *testing.T) {
	rb := ReaderBody(io.NopCloser(strings.NewReader("rc")))
	rc, err := rb.Open()
	if err != nil {
		t.Fatal(err)
	}
	if readAllClose(t, rc) != "rc" {
		t.Error("ReadCloser passthrough content mismatch")
	}
}

func TestMultipartBody_ReaderPartRoundTrip(t *testing.T) {
	mb := MultipartBody(nil, MultipartFile{Field: "doc", FileName: "r.txt", Reader: strings.NewReader("READERDATA")})
	ct := mb.(options.ContentTyper).ContentType()
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatal(err)
	}
	rc, err := mb.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	form, err := multipart.NewReader(rc, params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(form.File["doc"]) != 1 {
		t.Fatalf("file part missing: %+v", form.File)
	}
	f, _ := form.File["doc"][0].Open()
	defer f.Close()
	b, _ := io.ReadAll(f)
	if string(b) != "READERDATA" {
		t.Errorf("reader part content = %q", b)
	}
}

func TestMultipartBody_CancellationNoLeak(t *testing.T) {
	// Open many multipart bodies and close them early without reading; the
	// writer goroutines must exit (closing the pipe reader unblocks them).
	base := runtime.NumGoroutine()
	for i := 0; i < 50; i++ {
		mb := MultipartBody(map[string]string{"k": strings.Repeat("v", 100000)})
		rc, err := mb.Open()
		if err != nil {
			t.Fatal(err)
		}
		_ = rc.Close() // close before reading -> writer must unblock and exit
	}
	// Allow goroutines to wind down.
	for i := 0; i < 50; i++ {
		if runtime.NumGoroutine() <= base+5 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("goroutines did not wind down: base=%d now=%d (possible writer-goroutine leak)", base, runtime.NumGoroutine())
}

func TestCreateRequest_BodyStreamSetsLenAndGetBody(t *testing.T) {
	opts := options.NewRequestOptions("https://example.com")
	opts.Method = "POST"
	opts.Headers = nil
	opts.BodyStream = BytesBody([]byte("12345"))

	req, err := createRequest(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if req.ContentLength != 5 {
		t.Errorf("ContentLength = %d, want 5", req.ContentLength)
	}
	if req.GetBody == nil {
		t.Error("GetBody should be set for a rewindable body source")
	}
	// GetBody yields a fresh, full body.
	rc, err := req.GetBody()
	if err != nil {
		t.Fatal(err)
	}
	if readAllClose(t, rc) != "12345" {
		t.Error("GetBody content mismatch")
	}
}

func TestCreateRequest_MultipartContentType(t *testing.T) {
	opts := options.NewRequestOptions("https://example.com")
	opts.Method = "POST"
	opts.BodyStream = MultipartBody(map[string]string{"a": "b"})

	req, err := createRequest(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if ct := req.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data; boundary=") {
		t.Errorf("Content-Type = %q", ct)
	}
	// Non-rewindable check is for reader-based; this path-less multipart is rewindable.
	if req.GetBody == nil {
		t.Error("rewindable multipart should set GetBody")
	}
}
