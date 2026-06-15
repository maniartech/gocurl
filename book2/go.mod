// Separate module so the book's example programs (each `package main`) do not
// pollute the library module's `go build/vet/test ./...` or its pkg.go.dev page.
module github.com/maniartech/gocurl/book2

go 1.22.3

require github.com/maniartech/gocurl v0.0.0

replace github.com/maniartech/gocurl => ../
