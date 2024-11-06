package middlewares

import "net/http"

// MiddlewareFunc is a function type for request middleware.
type MiddlewareFunc func(*http.Request) (*http.Request, error)
