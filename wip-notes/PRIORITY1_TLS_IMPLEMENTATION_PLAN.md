# Priority 1 TLS/Proxy Features Implementation Plan

## Objective

Implement curl-compatible TLS version control, cipher suite selection, and proxy TLS authentication to achieve **80-90% curl HTTP/HTTPS parity**.

---

## Features to Implement

### 1. TLS Version Control Flags

```bash
# Set minimum TLS version
gocurl --tlsv1.2 https://example.com
gocurl --tlsv1.3 https://example.com

# Set maximum TLS version
gocurl --tls-max 1.3 https://example.com

# Combined
gocurl --tlsv1.2 --tls-max 1.3 https://example.com
```

**curl equivalent:**
```bash
curl --tlsv1.2 --tls-max 1.3 https://example.com
```

### 2. Cipher Suite Selection

```bash
# Single cipher
gocurl --ciphers "ECDHE-RSA-AES256-GCM-SHA384" https://example.com

# Multiple ciphers (colon-separated, curl style)
gocurl --ciphers "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256" https://example.com

# TLS 1.3 ciphers
gocurl --tls13-ciphers "TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256" https://example.com
```

**curl equivalent:**
```bash
curl --ciphers "ECDHE-RSA-AES256-GCM-SHA384" https://example.com
curl --tls13-ciphers "TLS_AES_256_GCM_SHA384" https://example.com
```

### 3. Proxy TLS Authentication

```bash
# HTTPS proxy with client certificate
gocurl --proxy https://proxy.example.com:8080 \
       --proxy-cert client.pem \
       --proxy-key client.key \
       https://example.com

# HTTPS proxy with CA cert
gocurl --proxy https://proxy.example.com:8080 \
       --proxy-cacert ca.pem \
       https://example.com

# HTTPS proxy - skip verification
gocurl --proxy https://proxy.example.com:8080 \
       --proxy-insecure \
       https://example.com
```

**curl equivalent:**
```bash
curl -x https://proxy.example.com:8080 \
     --proxy-cert client.pem \
     --proxy-key client.key \
     https://example.com
```

---

## Implementation Steps

### Phase 1: Data Structures (options.go, proxy/types.go)

#### A. Update `options/options.go`

Add new fields to `RequestOptions`:

```go
// TLS version control
TLSMinVersion uint16   // tls.VersionTLS12, tls.VersionTLS13, etc.
TLSMaxVersion uint16   // tls.VersionTLS13, etc.

// Cipher suite control
CipherSuites      []uint16  // TLS 1.2 cipher suites
TLS13CipherSuites []uint16  // TLS 1.3 cipher suites
```

#### B. Update `proxy/types.go`

Add to `ProxyConfig`:

```go
// Proxy TLS authentication
ClientCert string  // Path to client certificate
ClientKey  string  // Path to client key
CACert     string  // Path to CA certificate
Insecure   bool    // Skip TLS verification for proxy
```

### Phase 2: Parser Updates (cmd/parser.go)

Add new flag handlers:

```go
// TLS version flags
case "--tlsv1", "--tlsv1.0":
    o.TLSMinVersion = tls.VersionTLS10
case "--tlsv1.1":
    o.TLSMinVersion = tls.VersionTLS11
case "--tlsv1.2":
    o.TLSMinVersion = tls.VersionTLS12
case "--tlsv1.3":
    o.TLSMinVersion = tls.VersionTLS13
case "--tls-max":
    // Parse next arg: "1.0", "1.1", "1.2", "1.3"
    o.TLSMaxVersion = parseTLSMaxVersion(nextArg)

// Cipher suite flags
case "--ciphers":
    o.CipherSuites = parseCipherSuites(nextArg)
case "--tls13-ciphers":
    o.TLS13CipherSuites = parseTLS13CipherSuites(nextArg)

// Proxy TLS flags
case "--proxy-cert":
    // Need to store in ProxyConfig
    o.ProxyCert = nextArg
case "--proxy-key":
    o.ProxyKey = nextArg
case "--proxy-cacert":
    o.ProxyCACert = nextArg
case "--proxy-insecure":
    o.ProxyInsecure = true
```

### Phase 3: Cipher Suite Mapping (new file: tls_utils.go)

Create `tls_utils.go` for cipher suite mapping:

```go
package gocurl

import (
    "crypto/tls"
    "fmt"
    "strings"
)

// Cipher suite name mapping (OpenSSL names -> Go constants)
var cipherSuiteMap = map[string]uint16{
    // TLS 1.2 ECDHE
    "ECDHE-RSA-AES256-GCM-SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
    "ECDHE-RSA-AES128-GCM-SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
    "ECDHE-ECDSA-AES256-GCM-SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
    "ECDHE-ECDSA-AES128-GCM-SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,

    // TLS 1.2 CHACHA20
    "ECDHE-RSA-CHACHA20-POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
    "ECDHE-ECDSA-CHACHA20-POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,

    // TLS 1.2 CBC (legacy)
    "ECDHE-RSA-AES256-CBC-SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
    "ECDHE-RSA-AES128-CBC-SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
    "ECDHE-ECDSA-AES256-CBC-SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
    "ECDHE-ECDSA-AES128-CBC-SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,

    // RSA (legacy)
    "AES256-GCM-SHA384":             tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
    "AES128-GCM-SHA256":             tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
}

// TLS 1.3 cipher suite mapping
var tls13CipherSuiteMap = map[string]uint16{
    "TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
    "TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
    "TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,
}

// ParseCipherSuites parses colon-separated cipher suite names
func ParseCipherSuites(cipherStr string) ([]uint16, error) {
    if cipherStr == "" {
        return nil, nil
    }

    names := strings.Split(cipherStr, ":")
    suites := make([]uint16, 0, len(names))

    for _, name := range names {
        name = strings.TrimSpace(name)
        if suite, ok := cipherSuiteMap[name]; ok {
            suites = append(suites, suite)
        } else {
            return nil, fmt.Errorf("unknown cipher suite: %s", name)
        }
    }

    return suites, nil
}

// ParseTLS13CipherSuites parses TLS 1.3 cipher suite names
func ParseTLS13CipherSuites(cipherStr string) ([]uint16, error) {
    if cipherStr == "" {
        return nil, nil
    }

    names := strings.Split(cipherStr, ":")
    suites := make([]uint16, 0, len(names))

    for _, name := range names {
        name = strings.TrimSpace(name)
        if suite, ok := tls13CipherSuiteMap[name]; ok {
            suites = append(suites, suite)
        } else {
            return nil, fmt.Errorf("unknown TLS 1.3 cipher suite: %s", name)
        }
    }

    return suites, nil
}

// ParseTLSMaxVersion parses TLS max version string
func ParseTLSMaxVersion(version string) (uint16, error) {
    switch version {
    case "1.0":
        return tls.VersionTLS10, nil
    case "1.1":
        return tls.VersionTLS11, nil
    case "1.2":
        return tls.VersionTLS12, nil
    case "1.3":
        return tls.VersionTLS13, nil
    default:
        return 0, fmt.Errorf("invalid TLS version: %s (use 1.0, 1.1, 1.2, or 1.3)", version)
    }
}
```

### Phase 4: Apply TLS Configuration (convert.go)

Update `createTLSConfig()` to apply new settings:

```go
func createTLSConfig(o *options.RequestOptions) (*tls.Config, error) {
    tlsConfig := &tls.Config{
        InsecureSkipVerify: o.Insecure,
    }

    // Apply TLS version constraints
    if o.TLSMinVersion != 0 {
        tlsConfig.MinVersion = o.TLSMinVersion
    }
    if o.TLSMaxVersion != 0 {
        tlsConfig.MaxVersion = o.TLSMaxVersion
    }

    // Apply cipher suites
    if len(o.CipherSuites) > 0 {
        tlsConfig.CipherSuites = o.CipherSuites
    }

    // Apply TLS 1.3 cipher suites (Go 1.21+)
    if len(o.TLS13CipherSuites) > 0 {
        // Note: TLS 1.3 cipher suites are not configurable in older Go versions
        // This will require checking Go version or using build tags
    }

    // Existing cert loading code...
    if o.CertFile != "" && o.KeyFile != "" {
        cert, err := tls.LoadX509KeyPair(o.CertFile, o.KeyFile)
        if err != nil {
            return nil, fmt.Errorf("failed to load client certificate: %w", err)
        }
        tlsConfig.Certificates = []tls.Certificate{cert}
    }

    // Existing CA cert loading code...
    if o.CAFile != "" {
        caCert, err := os.ReadFile(o.CAFile)
        if err != nil {
            return nil, fmt.Errorf("failed to read CA certificate: %w", err)
        }
        caCertPool := x509.NewCertPool()
        if !caCertPool.AppendCertsFromPEM(caCert) {
            return nil, fmt.Errorf("failed to parse CA certificate")
        }
        tlsConfig.RootCAs = caCertPool
    }

    return tlsConfig, nil
}
```

### Phase 5: Proxy TLS Configuration (proxy/httpproxy.go)

Update proxy creation to support TLS:

```go
func CreateHTTPProxyTransport(config *ProxyConfig) (*http.Transport, error) {
    proxyURL, err := url.Parse(config.Address)
    if err != nil {
        return nil, fmt.Errorf("invalid proxy URL: %w", err)
    }

    transport := &http.Transport{
        Proxy: http.ProxyURL(proxyURL),
        // ... other settings
    }

    // If proxy is HTTPS, configure TLS
    if proxyURL.Scheme == "https" {
        tlsConfig := &tls.Config{
            InsecureSkipVerify: config.Insecure,
        }

        // Load proxy client certificate
        if config.ClientCert != "" && config.ClientKey != "" {
            cert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
            if err != nil {
                return nil, fmt.Errorf("failed to load proxy client cert: %w", err)
            }
            tlsConfig.Certificates = []tls.Certificate{cert}
        }

        // Load proxy CA certificate
        if config.CACert != "" {
            caCert, err := os.ReadFile(config.CACert)
            if err != nil {
                return nil, fmt.Errorf("failed to read proxy CA cert: %w", err)
            }
            caCertPool := x509.NewCertPool()
            if !caCertPool.AppendCertsFromPEM(caCert) {
                return nil, fmt.Errorf("failed to parse proxy CA cert")
            }
            tlsConfig.RootCAs = caCertPool
        }

        // Apply TLS config to proxy connections
        transport.TLSClientConfig = tlsConfig
    }

    return transport, nil
}
```

### Phase 6: Testing Strategy

#### A. TLS Version Tests

```go
func TestTLSVersionControl(t *testing.T) {
    tests := []struct {
        name       string
        minVersion uint16
        maxVersion uint16
        shouldWork bool
    }{
        {"TLS 1.2 min", tls.VersionTLS12, 0, true},
        {"TLS 1.3 only", tls.VersionTLS13, tls.VersionTLS13, true},
        {"TLS 1.0-1.2", tls.VersionTLS10, tls.VersionTLS12, true},
    }
    // Test against servers with different TLS versions
}
```

#### B. Cipher Suite Tests

```go
func TestCipherSuites(t *testing.T) {
    // Test parsing
    suites, err := ParseCipherSuites("ECDHE-RSA-AES256-GCM-SHA384")
    assert.NoError(t, err)
    assert.Len(t, suites, 1)

    // Test invalid cipher
    _, err = ParseCipherSuites("INVALID-CIPHER")
    assert.Error(t, err)
}
```

#### C. Proxy TLS Tests

```go
func TestProxyTLS(t *testing.T) {
    // Requires test HTTPS proxy with client cert auth
    // Can use mitmproxy or squid with TLS
}
```

### Phase 7: Documentation Updates

#### A. CLI Help Text

Add to parser help:

```
TLS Options:
  --tlsv1.0          Use TLS 1.0 or higher
  --tlsv1.1          Use TLS 1.1 or higher
  --tlsv1.2          Use TLS 1.2 or higher
  --tlsv1.3          Use TLS 1.3 or higher
  --tls-max <ver>    Maximum TLS version (1.0, 1.1, 1.2, 1.3)
  --ciphers <list>   TLS 1.2 cipher suites (colon-separated)
  --tls13-ciphers    TLS 1.3 cipher suites (colon-separated)

Proxy TLS Options:
  --proxy-cert       Client certificate for HTTPS proxy
  --proxy-key        Private key for HTTPS proxy
  --proxy-cacert     CA certificate for HTTPS proxy
  --proxy-insecure   Skip TLS verification for proxy
```

#### B. Book Updates

Add sections to:
- Chapter 4 (CLI): TLS version control examples
- Chapter 5 (Builder Pattern): TLS configuration API
- Security chapter: Cipher suite selection guide

---

## Implementation Order

1. ✅ **Day 1: Data structures**
   - Update `options.go` with TLS fields
   - Update `proxy/types.go` with proxy TLS fields

2. ✅ **Day 2: Cipher suite mapping**
   - Create `tls_utils.go`
   - Implement cipher name parsing
   - Add comprehensive cipher suite map

3. ✅ **Day 3: Parser updates**
   - Add TLS version flags to `cmd/parser.go`
   - Add cipher suite flags
   - Add proxy TLS flags

4. ✅ **Day 4: TLS config application**
   - Update `convert.go` createTLSConfig()
   - Update proxy TLS in `proxy/httpproxy.go`

5. ✅ **Day 5: Testing**
   - Write unit tests for parsing
   - Write integration tests with TLS servers
   - Test with different cipher suites

---

## Expected Outcome

After implementation:

| Feature Category | Before | After |
|-----------------|--------|-------|
| **TLS Version Control** | ❌ 0% | ✅ 100% |
| **Cipher Suite Control** | ❌ 0% | ✅ 100% |
| **Proxy TLS Auth** | ❌ 0% | ✅ 100% |
| **Overall TLS Score** | 50% | **95%** |
| **Overall HTTP/HTTPS Score** | 65% | **85%** |

### Curl Parity Achievement

```
✅ HTTP Core:        100% (unchanged)
✅ HTTP/2:           100% (unchanged)
✅ TLS Core:         100% (unchanged)
✅ TLS Advanced:     20% → 95% (+75%)
✅ Proxy Core:       100% (unchanged)
✅ Proxy Advanced:   10% → 80% (+70%)
⚠️ Authentication:   33% (unchanged - future work)

Overall: 65% → 85% (+20%)
```

---

## Risk Assessment

### Low Risk
- ✅ TLS version flags (straightforward uint16 assignment)
- ✅ Cipher suite parsing (deterministic mapping)

### Medium Risk
- ⚠️ TLS 1.3 cipher suites (Go version compatibility)
- ⚠️ Proxy TLS (need to test with real HTTPS proxies)

### Mitigation
- Add Go version checks for TLS 1.3 features
- Create comprehensive test suite with real servers
- Document any Go version requirements

---

## Success Criteria

- [ ] All TLS version flags work (`--tlsv1.2`, `--tlsv1.3`, `--tls-max`)
- [ ] Cipher suites can be selected (`--ciphers`)
- [ ] HTTPS proxies work with client certs (`--proxy-cert`)
- [ ] All tests pass
- [ ] Documentation updated
- [ ] Book updated with examples
- [ ] Help text includes new flags

**Target: 85% curl HTTP/HTTPS compatibility** ✅
