# User-Agent Implementation Summary

## Changes Made

✅ **gocurl now follows curl's behavior by always sending a User-Agent header**

### Files Modified

1. **version.go** (NEW)
   - Added `Version` variable for build-time version injection
   - Default: `"dev"`
   - Can be set at build time: `go build -ldflags "-X github.com/maniartech/gocurl.Version=1.0.0"`

2. **process.go**
   - Updated `applyHeaders()` function
   - Always sets User-Agent header in format: `gocurl/<VERSION>`
   - Falls back to custom User-Agent if explicitly set via `opts.UserAgent`

3. **process2_test.go**
   - Added test for default User-Agent behavior
   - Existing custom User-Agent test updated
   - Both tests passing ✅

4. **USER_AGENT_CURL_COMPATIBILITY.md** (NEW)
   - Comprehensive documentation
   - Examples and comparisons
   - Implementation details

## Behavior

### Before
```bash
# No User-Agent header sent by default
# Would use Go's http.Client default only if header was added
```

### After
```bash
$ gocurl https://httpbin.org/user-agent
{
  "user-agent": "gocurl/dev"
}

# Matches curl's behavior:
$ curl https://httpbin.org/user-agent
{
  "user-agent": "curl/7.85.0"
}
```

## API Compatibility

✅ **Backward Compatible** - Custom User-Agent still works:

```go
// Still works exactly as before
opts := options.NewRequestOptionsBuilder().
    SetUserAgent("MyApp/1.0").
    Build()
```

```bash
# Still works exactly as before
gocurl -A "MyApp/1.0" https://api.example.com
```

## Test Results

```bash
$ go test -run=TestCustomUserAgent -v
=== RUN   TestCustomUserAgent
=== RUN   TestCustomUserAgent/Default_User-Agent_follows_curl_behavior
=== RUN   TestCustomUserAgent/Custom_User-Agent_string
--- PASS: TestCustomUserAgent (0.00s)
    --- PASS: TestCustomUserAgent/Default_User-Agent_follows_curl_behavior (0.00s)
    --- PASS: TestCustomUserAgent/Custom_User-Agent_string (0.00s)
PASS
```

✅ All tests passing
✅ No regressions in existing functionality

## Version Management

The version can be set during build:

```bash
# For releases
go build -ldflags "-X github.com/maniartech/gocurl.Version=1.0.0"

# For development (default)
go build  # Uses "dev"
```

This allows:
- Development builds to show `gocurl/dev`
- Release builds to show actual version like `gocurl/1.0.0`
- CI/CD pipelines to inject version from tags

## Why This Change

1. **Curl Compatibility**: Matches curl's well-established behavior
2. **HTTP Best Practice**: User-Agent should always be sent
3. **Server Logging**: Helps servers identify client type
4. **API Analytics**: Many APIs track User-Agent for metrics
5. **Industry Standard**: Python requests, curl, wget all send default User-Agent

## Impact

- ✅ **No breaking changes** - custom User-Agent still works
- ✅ **Improves curl parity** - closer to curl's behavior
- ✅ **Better server identification** - servers can log gocurl usage
- ✅ **HTTP best practice** - follows RFC recommendations

## Next Steps

1. ✅ Implementation complete
2. ✅ Tests passing
3. ✅ Documentation created
4. 📝 Consider updating README.md to mention default User-Agent
5. 📝 Consider adding version command to CLI: `gocurl --version`

## Related Files

- `version.go` - Version variable
- `process.go` - Default User-Agent logic
- `process2_test.go` - Tests
- `wip-notes/USER_AGENT_CURL_COMPATIBILITY.md` - Full documentation
