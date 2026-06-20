// Separate module so the book's example programs (each `package main`) do not
// pollute the library module's `go build/vet/test ./...` or its pkg.go.dev page.
module github.com/maniartech/gocurl/book2

go 1.22.3

require github.com/maniartech/gocurl v0.0.0

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/text v0.19.0 // indirect
)

replace github.com/maniartech/gocurl => ../
