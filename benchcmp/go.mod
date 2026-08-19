// Module benchcmp holds the competitive benchmark arms (resty, req) that compare
// gocurl against popular Go HTTP clients. It is a SEPARATE module on purpose: its
// vendor dependencies (resty, req, and their transitive graph) never enter the
// root gocurl module's require graph, so `go mod tidy` on the library stays clean
// (enforced by TestNoVendorDepsInRootModule). Run:
//
//	cd benchcmp && go test -bench . -benchmem -run '^$'
module github.com/maniartech/gocurl/benchcmp

go 1.25.0

require (
	github.com/go-resty/resty/v2 v2.16.5
	github.com/imroc/req/v3 v3.61.0
	github.com/maniartech/gocurl v0.0.0
)

require (
	github.com/andybalholm/brotli v1.2.2 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/icholy/digest v1.2.0 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.61.0 // indirect
	github.com/refraction-networking/utls v1.8.2 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace github.com/maniartech/gocurl => ../
