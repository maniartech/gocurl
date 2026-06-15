package options

import "io"

// BodySource is the single abstraction for a request body. It lets the client
// stream bodies (files, readers, multipart) instead of buffering them, while
// still supporting replay for retries via Rewindable/Open.
//
// See specs/05-streaming-and-bodies.md.
type BodySource interface {
	// Open returns a fresh reader over the body. For a Rewindable source it may
	// be called multiple times (e.g. for retries); each call yields the full
	// body from the start. The caller closes the returned ReadCloser.
	Open() (io.ReadCloser, error)

	// Len reports the body length in bytes and whether it is known. When known,
	// it is used to set Content-Length.
	Len() (int64, bool)

	// Rewindable reports whether Open can be called again to replay the body
	// (so retries do not need to buffer it).
	Rewindable() bool
}

// ContentTyper is an optional interface a BodySource may implement to supply a
// Content-Type (e.g. multipart sets its multipart/form-data boundary).
type ContentTyper interface {
	ContentType() string
}
