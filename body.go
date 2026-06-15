package gocurl

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/maniartech/gocurl/options"
)

// BytesBody returns a rewindable BodySource backed by an in-memory byte slice.
func BytesBody(b []byte) options.BodySource { return &bytesBody{data: b} }

// StringBody returns a rewindable BodySource backed by a string.
func StringBody(s string) options.BodySource { return &bytesBody{data: []byte(s)} }

type bytesBody struct{ data []byte }

func (b *bytesBody) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.data)), nil
}
func (b *bytesBody) Len() (int64, bool) { return int64(len(b.data)), true }
func (b *bytesBody) Rewindable() bool   { return true }

// FileBody returns a rewindable BodySource that streams a file from disk
// (re-opened for each retry instead of being buffered in memory).
func FileBody(path string) options.BodySource { return &fileBody{path: path} }

type fileBody struct{ path string }

func (f *fileBody) Open() (io.ReadCloser, error) { return os.Open(f.path) }
func (f *fileBody) Len() (int64, bool) {
	fi, err := os.Stat(f.path)
	if err != nil {
		return 0, false
	}
	return fi.Size(), true
}
func (f *fileBody) Rewindable() bool { return true }

// ReaderBody returns a NON-rewindable BodySource that streams from r once. It
// cannot be replayed for retries; use BytesBody/FileBody when retries are needed.
func ReaderBody(r io.Reader) options.BodySource { return &readerBody{r: r} }

type readerBody struct {
	r    io.Reader
	used bool
}

func (b *readerBody) Open() (io.ReadCloser, error) {
	if b.used {
		return nil, fmt.Errorf("reader body already consumed (not rewindable)")
	}
	b.used = true
	if rc, ok := b.r.(io.ReadCloser); ok {
		return rc, nil
	}
	return io.NopCloser(b.r), nil
}
func (b *readerBody) Len() (int64, bool) { return 0, false }
func (b *readerBody) Rewindable() bool   { return false }

// MultipartFile describes one file part of a multipart/form-data body. Prefer
// Path (re-openable, so the body is rewindable for retries); Reader is a
// fallback that makes the body non-rewindable.
type MultipartFile struct {
	Field    string
	FileName string
	Path     string
	Reader   io.Reader
}

// MultipartBody returns a streaming multipart/form-data BodySource. The body is
// produced lazily through an io.Pipe; closing the returned reader unblocks the
// writer goroutine, so an aborted/cancelled request never leaks it.
func MultipartBody(fields map[string]string, files ...MultipartFile) options.BodySource {
	boundary := multipart.NewWriter(io.Discard).Boundary()
	rewindable := true
	for _, f := range files {
		if f.Path == "" { // a Reader part can only be consumed once
			rewindable = false
			break
		}
	}
	return &multipartBody{fields: fields, files: files, boundary: boundary, rewindable: rewindable}
}

type multipartBody struct {
	fields     map[string]string
	files      []MultipartFile
	boundary   string
	rewindable bool
}

func (m *multipartBody) Open() (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go func() {
		w := multipart.NewWriter(pw)
		_ = w.SetBoundary(m.boundary)
		if err := m.writeParts(w); err != nil {
			// If the reader was closed early, pw writes already fail; propagate.
			_ = pw.CloseWithError(err)
			return
		}
		// w.Close writes the trailing boundary; propagate its error (or nil).
		_ = pw.CloseWithError(w.Close())
	}()
	return pr, nil
}

func (m *multipartBody) writeParts(w *multipart.Writer) error {
	for k, v := range m.fields {
		if err := w.WriteField(k, v); err != nil {
			return err
		}
	}
	for _, f := range m.files {
		part, err := w.CreateFormFile(f.Field, f.FileName)
		if err != nil {
			return err
		}
		src, closeFn, err := openPartSource(f)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(part, src)
		if closeFn != nil {
			closeFn()
		}
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func openPartSource(f MultipartFile) (io.Reader, func(), error) {
	if f.Path != "" {
		file, err := os.Open(f.Path)
		if err != nil {
			return nil, nil, err
		}
		return file, func() { file.Close() }, nil
	}
	return f.Reader, nil, nil
}

func (m *multipartBody) Len() (int64, bool) { return 0, false }
func (m *multipartBody) Rewindable() bool   { return m.rewindable }
func (m *multipartBody) ContentType() string {
	return "multipart/form-data; boundary=" + m.boundary
}
