# benchcmp — competitive benchmarks

This is a **separate Go module** holding the cross-client benchmark arms that compare
gocurl against popular Go HTTP clients ([resty](https://github.com/go-resty/resty),
[req](https://github.com/imroc/req)).

It is isolated on purpose: resty and req pull a large transitive dependency graph
(quic-go, utls, ginkgo, …). Keeping these arms in their own module means a gocurl
**library** user never transitively depends on any of it. The root module's cleanliness
is enforced by `TestNoVendorDepsInRootModule` in the parent package.

## Running

```bash
cd benchcmp
go test -run='^$' -bench='BenchmarkCmp' -benchmem -count=10 .
```

All arms run over one shared in-process `httptest` server with **identical** transport
tuning (`benchFairTransport`, mirroring gocurl's default Client transport), so the numbers
compare client overhead rather than connection-pool configuration.

## Honesty rules

- Same server, same fair transport, body drained on every arm.
- resty and req **buffer** the full response body by default; gocurl **streams**. That
  semantic difference is real and is noted in [`../docs/benchmarking.md`](../docs/benchmarking.md).
- Results are reported as-measured — wins **and** losses. gocurl is currently the heaviest
  on `B/op`; we publish that rather than hide it.

See [`../docs/benchmarking.md`](../docs/benchmarking.md) for the full methodology, the
fairness guard, and the results table.
