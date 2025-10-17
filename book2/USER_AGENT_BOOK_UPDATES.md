# Book Documentation Updates - User-Agent Default Behavior

## Summary

Updated the book documentation to naturally reflect that gocurl now sends a default User-Agent header, following curl's behavior.

## Files Updated

### 1. Chapter 4 - CLI (chapter04-cli/chapter.md)

**Location: Line 253 - Custom User-Agent Section**

**Before:**
```markdown
### Custom User-Agent (-A, --user-agent)

```bash
gocurl -A "MyApp/1.0" https://httpbin.org/user-agent
```
```

**After:**
```markdown
### Custom User-Agent (-A, --user-agent)

By default, gocurl sends `gocurl/dev` as the User-Agent (matching curl's behavior of sending `curl/VERSION`). You can customize this:

```bash
gocurl -A "MyApp/1.0" https://httpbin.org/user-agent
```
```

**Rationale:** Naturally introduces the default behavior right before showing how to customize it.

---

**Location: Line 88 - Example Output**

**Before:**
```json
"User-Agent": "Go-http-client/2.0"
```

**After:**
```json
"User-Agent": "gocurl/dev"
```

**Rationale:** Updates example output to reflect actual default behavior.

---

### 2. Chapter 5 - Builder Pattern (chapter05-builder-pattern/chapter.md)

**Location: Line 336 - UserAgent Field Description**

**Before:**
```markdown
**UserAgent** (`string`)
- Custom User-Agent header
- Overrides default GoCurl user agent
- Example: `UserAgent: "MyApp/1.0"`
```

**After:**
```markdown
**UserAgent** (`string`)
- Custom User-Agent header
- Default: `gocurl/dev` (or `gocurl/VERSION` in releases)
- Overrides the default User-Agent
- Example: `UserAgent: "MyApp/1.0"`
```

**Rationale:** Documents the default value inline with the field description.

---

### 3. API Reference (API_REFERENCE.md)

**Location: Line 987 - SetUserAgent Method**

**Before:**
```markdown
Sets User-Agent header.
```

**After:**
```markdown
Sets custom User-Agent header. If not set, defaults to `gocurl/dev` (or `gocurl/VERSION` in releases), following curl's behavior.
```

**Rationale:** API reference should be explicit about default behavior.

---

## Integration Approach

✅ **Natural and Immersive** - Updates blend seamlessly into existing content
✅ **Not Over-Emphasized** - Brief mentions where relevant, not highlighting as "new feature"
✅ **Context-Appropriate** - Default mentioned when discussing customization
✅ **Example-Accurate** - Output examples reflect actual behavior
✅ **Curl Parity Emphasized** - Frames it as "matching curl's behavior" (a positive)

## What We Did NOT Do (Intentionally)

❌ Didn't add a dedicated "New Feature" section
❌ Didn't create a migration guide (it's backward compatible)
❌ Didn't add callout boxes or warnings
❌ Didn't update every mention of User-Agent
❌ Didn't add to README or introduction (too prominent)

## Locations Where User-Agent is Mentioned (No Changes Needed)

### Chapter 4 - CLI
- Line 131: Verbose output example already shows `gocurl/1.0.0` ✅ (correct for release)
- Line 221: Custom header example - context is about custom headers, not defaults ✅

### Chapter 5 - Builder Pattern
- Line 523: `SetUserAgent()` method listing - updated in API_REFERENCE ✅
- Exercise examples - show custom values (appropriate for exercises) ✅

### Exercises
- Multiple exercises use custom User-Agent values
- **Decision:** Leave as-is - exercises teach customization, not defaults ✅

## Verification

All changes follow the book's style guide:
- ✅ Brief and informative
- ✅ Code examples are accurate
- ✅ Integrated naturally into flow
- ✅ Consistent terminology ("gocurl/dev" vs "gocurl/VERSION")
- ✅ References curl for context (familiar to readers)

## Future Considerations

When the book version number is determined:
- Update Line 131 in Chapter 4 if version changes from 1.0.0
- Update build instructions to show version injection
- Consider adding `--version` flag documentation when implemented

## Related Documentation

- `wip-notes/USER_AGENT_CURL_COMPATIBILITY.md` - Technical deep-dive
- `wip-notes/USER_AGENT_IMPLEMENTATION_SUMMARY.md` - Implementation details
- `version.go` - Version variable source code
- `process.go` - Default User-Agent implementation

## Impact Assessment

**Reader Experience:**
- ✅ Accurate information from the start
- ✅ No confusion about default behavior
- ✅ Natural curl parity reinforcement
- ✅ Clear how to customize when needed

**Backward Compatibility:**
- ✅ No breaking changes to document
- ✅ Custom User-Agent examples still work
- ✅ No migration needed

**Book Quality:**
- ✅ Examples match actual behavior
- ✅ Consistent with curl (book's theme)
- ✅ Professional documentation standard
