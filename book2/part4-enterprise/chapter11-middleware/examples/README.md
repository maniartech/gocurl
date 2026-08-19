# Middleware examples

`01-production-composition` is a hermetic executable example of the managed production
pipeline. Run it from `book2` with:

```bash
go run ./part4-enterprise/chapter11-middleware/examples/01-production-composition
```

Expected output includes `status=200`, `body="ready"`, and `retries=1`. The example's
loopback allow-list is test-only; the comments explain why copying it into production
would weaken the SSRF boundary.
