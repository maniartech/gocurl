# Book Examples Testing System

Unified testing system for all GoCurl book examples in Part 1 (Foundations) and Part 2 (API Approaches).

## Quick Start

### Test All Examples (Recommended)
```bash
./scripts/test-book-examples.sh
```

### Test Specific Part
```bash
./scripts/test-book-examples.sh -p 2        # Test Part 2 only
```

### Test Specific Chapter
```bash
./scripts/test-book-examples.sh -p 2 -c 6   # Test Chapter 6 of Part 2
```

### Verbose Output with Timing
```bash
./scripts/test-book-examples.sh -v
```

---

## Testing Methods

The system supports two testing methods:

### 1. Go-based Testing (Default, Recommended)
- **Faster:** Parallel compilation
- **Better output:** Colored, formatted results
- **More features:** Timing, filters, verbose mode
- **Requires:** Go installed

```bash
./scripts/test-book-examples.sh -m go
# OR
go run scripts/test-examples.go
```

### 2. Bash-based Testing (Fallback)
- **Universal:** Works without Go installation
- **Simple:** Pure bash implementation
- **Portable:** Runs anywhere with bash

```bash
./scripts/test-book-examples.sh -m bash
```

### 3. Auto-detection (Default)
- Automatically uses Go if available
- Falls back to bash if Go not found

```bash
./scripts/test-book-examples.sh          # Auto-detects
./scripts/test-book-examples.sh -m auto  # Explicit auto
```

---

## Command-Line Options

### Unified Launcher (`test-book-examples.sh`)

```bash
./scripts/test-book-examples.sh [OPTIONS]
```

| Option | Description | Example |
|--------|-------------|---------|
| `-m, --method <auto\|bash\|go>` | Test method (default: auto) | `-m go` |
| `-r, --run` | Run examples (not just compile) | `-r` |
| `-v, --verbose` | Verbose output with timing | `-v` |
| `-p, --part <1\|2>` | Test only specific part | `-p 2` |
| `-c, --chapter <N>` | Test only specific chapter | `-c 6` |
| `-h, --help` | Show help message | `-h` |

### Go Test Program (`test-examples.go`)

```bash
go run scripts/test-examples.go [FLAGS]
```

| Flag | Description | Example |
|------|-------------|---------|
| `-part <N>` | Test only part 1 or 2 | `-part=2` |
| `-chapter <N>` | Test only specific chapter | `-chapter=6` |
| `-run` | Run examples (not just compile) | `-run` |
| `-verbose` | Show timing and details | `-verbose` |

---

## Usage Examples

### Basic Testing

```bash
# Test everything (auto-detect method)
./scripts/test-book-examples.sh

# Test using Go specifically
./scripts/test-book-examples.sh -m go

# Test using bash only
./scripts/test-book-examples.sh -m bash
```

### Filtered Testing

```bash
# Test only Part 1
./scripts/test-book-examples.sh -p 1

# Test only Part 2
./scripts/test-book-examples.sh -p 2

# Test only Chapter 5
./scripts/test-book-examples.sh -c 5

# Test Chapter 6 of Part 2
./scripts/test-book-examples.sh -p 2 -c 6
```

### Advanced Testing

```bash
# Verbose mode (shows timing)
./scripts/test-book-examples.sh -v

# Actually RUN examples (not just compile)
# Warning: Makes network calls!
./scripts/test-book-examples.sh -r

# Run Chapter 6 examples with verbose output
./scripts/test-book-examples.sh -p 2 -c 6 -r -v
```

### Direct Go Usage

```bash
# Using go run directly
go run scripts/test-examples.go
go run scripts/test-examples.go -part=2
go run scripts/test-examples.go -chapter=6 -verbose
go run scripts/test-examples.go -part=2 -chapter=7 -run
```

---

## Output Examples

### Success Output
```
==================================================
Testing All Book Examples
==================================================

Testing Part 2: API Approaches
-----------------------------------
[1] part2-api-approaches/chapter06-json-apis/examples/01-basic-json ... PASS ✓
[2] part2-api-approaches/chapter06-json-apis/examples/02-json-array ... PASS ✓
...

==================================================
Test Summary
==================================================

Total Examples:   17
Passed:           17 ✓
Failed:           0 ✗
Skipped:          0 ⊘

Success Rate: 100%

✅ All examples compiled successfully!
```

### Verbose Output (with timing)
```
[1] part2-api-approaches/chapter06-json-apis/examples/01-basic-json ... PASS ✓ (856ms)
[2] part2-api-approaches/chapter06-json-apis/examples/02-json-array ... PASS ✓ (842ms)
...
Total Build Time: 4.215s
```

### Failure Output
```
[5] part1-foundations/chapter02-installation/examples/broken-example ... FAIL ✗
     undefined: SomeFunction

Failed Examples:
  ❌ part1-foundations/chapter02-installation/examples/broken-example
```

---

## Files

| File | Description |
|------|-------------|
| `test-book-examples.sh` | Unified launcher (bash + Go) |
| `test-examples.go` | Go-based test program |
| `test-all-examples.sh` | Legacy bash-only (deprecated) |

---

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Test Book Examples

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Test All Examples
        run: ./scripts/test-book-examples.sh -m go
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Testing book examples..."
./scripts/test-book-examples.sh -m go

if [ $? -ne 0 ]; then
    echo "❌ Example tests failed! Commit aborted."
    exit 1
fi
```

---

## Testing Results

**Latest Test Run:** October 17, 2025

| Part | Chapters | Examples | Status |
|------|----------|----------|--------|
| Part 1 | 4 | 30 | ✅ All Pass |
| Part 2 | 3 | 17 | ✅ All Pass |
| **Total** | **7** | **47** | **✅ 100%** |

- **Compilation Success:** 40/40 testable examples (100%)
- **Skipped:** 7 CLI examples (shell scripts, no main.go)
- **Failed:** 0

See [TESTING_REPORT.md](../book2/TESTING_REPORT.md) for detailed results.

---

## Troubleshooting

### "Command not found: go"
- Install Go from https://go.dev/dl/
- Or use bash method: `./scripts/test-book-examples.sh -m bash`

### "Permission denied"
```bash
chmod +x scripts/test-book-examples.sh
```

### "No such file or directory: book2/"
```bash
# Make sure you're in the gocurl root directory
cd /path/to/gocurl
./scripts/test-book-examples.sh
```

### Example fails to compile
1. Check if dependencies are installed: `go mod download`
2. Run verbose mode to see full error: `-v`
3. Test individual example: `cd book2/.../example && go build`

---

## Contributing

When adding new examples:

1. Create example in appropriate chapter directory
2. Include `main.go` with runnable code
3. Test locally: `./scripts/test-book-examples.sh -c <chapter>`
4. Ensure it passes before committing

---

## Performance

### Compilation Times (Approximate)

| Examples | Go Method | Bash Method |
|----------|-----------|-------------|
| 1 example | ~0.8s | ~1.0s |
| 17 examples (Part 2) | ~14s | ~17s |
| 47 examples (All) | ~35s | ~47s |

**Recommendation:** Use Go method for faster testing.

---

## Advanced Usage

### Custom Filters

```bash
# Test multiple chapters (using bash for loop)
for ch in 5 6 7; do
    echo "Testing Chapter $ch..."
    ./scripts/test-book-examples.sh -c $ch
done

# Test with custom timeout
timeout 60s ./scripts/test-book-examples.sh
```

### Parallel Testing

```bash
# Test parts in parallel (bash)
./scripts/test-book-examples.sh -p 1 &
./scripts/test-book-examples.sh -p 2 &
wait
```

---

## License

Part of the GoCurl book project.

## Maintainers

- GitHub Copilot
- ManiarTech Team
