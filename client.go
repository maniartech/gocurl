package gocurl

import "net/http"

// HTTPClient is an interface for making HTTP requests
// This allows for mocking and custom client implementations
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Ensure *http.Client implements HTTPClient
var _ HTTPClient = (*http.Client)(nil)
