# Exercise 2: Variable Expansion System

Build a multi-environment configuration system using automatic and explicit variable expansion.

## Objectives

- Master environment variable expansion
- Understand WithVars functions
- Build multi-environment configuration
- Implement secure secret management

## Scenario

You're building a deployment tool that needs to connect to different API environments (dev, staging, production). Each environment has different URLs, API keys, and configuration. You need to implement a system that:

1. Loads configuration from environment variables
2. Supports explicit variable maps for multi-environment
3. Handles secrets securely
4. Makes API calls using the configured variables

## Requirements

### Part 1: Environment Variable Expansion (20 points)

Implement `loadFromEnvironment()` that:
- Sets required environment variables (API_URL, API_KEY, ENVIRONMENT)
- Makes requests using automatic variable expansion
- Validates that variables are expanded correctly

**Expected Environment Variables:**
```
API_URL=https://api.example.com
API_KEY=secret-key-12345
ENVIRONMENT=production
```

### Part 2: Explicit Variable Maps (30 points)

Implement `Environment` struct and `makeRequest()` method that:
- Stores environment-specific configuration
- Uses WithVars functions for explicit control
- Does NOT use environment variables (even if set)

**Environments to implement:**
- Development: `dev.example.com`, `dev-api-key`
- Staging: `staging.example.com`, `staging-api-key`
- Production: `prod.example.com`, `prod-api-key`

### Part 3: Multi-Environment Manager (30 points)

Implement `EnvironmentManager` that:
- Stores multiple environment configurations
- Switches between environments
- Makes requests to correct environment

### Part 4: Secure Secret Handling (20 points)

Implement:
- `loadSecretsFromVault()` - Simulates loading from secure vault
- `maskSecret()` - Masks secrets in logs
- Proper secret handling in all API calls

## Starter Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/maniartech/gocurl"
)

// Environment represents a deployment environment
type Environment struct {
    Name   string
    APIUrl string
    APIKey string
}

// EnvironmentManager manages multiple environments
type EnvironmentManager struct {
    environments map[string]*Environment
    current      string
}

// TODO 1: Implement loadFromEnvironment
// Use automatic environment variable expansion
func loadFromEnvironment() error {
    // Set environment variables
    os.Setenv("API_URL", "https://httpbin.org")
    os.Setenv("API_KEY", "env-secret-key")
    os.Setenv("ENDPOINT", "/headers")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // TODO: Make request using environment variables
    // Use ${API_URL}${ENDPOINT} and X-API-Key: ${API_KEY}

    return nil
}

// TODO 2: Implement NewEnvironment
func NewEnvironment(name, apiUrl, apiKey string) *Environment {
    // TODO: Create and return Environment
    return nil
}

// TODO 3: Implement makeRequest method on Environment
// Use WithVars functions with explicit variable map
func (e *Environment) makeRequest(endpoint string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // TODO: Create variable map
    // TODO: Use CurlCommandWithVars or similar

    return nil
}

// TODO 4: Implement NewEnvironmentManager
func NewEnvironmentManager() *EnvironmentManager {
    return &EnvironmentManager{
        environments: make(map[string]*Environment),
    }
}

// TODO 5: Implement AddEnvironment
func (em *EnvironmentManager) AddEnvironment(env *Environment) {
    // TODO: Add environment to map
}

// TODO 6: Implement SwitchTo
func (em *EnvironmentManager) SwitchTo(name string) error {
    // TODO: Switch current environment
    return nil
}

// TODO 7: Implement MakeRequest (uses current environment)
func (em *EnvironmentManager) MakeRequest(endpoint string) error {
    // TODO: Get current environment and make request
    return nil
}

// TODO 8: Implement loadSecretsFromVault
func loadSecretsFromVault(vaultKey string) (string, error) {
    // Simulate loading from HashiCorp Vault, AWS Secrets Manager, etc.
    secrets := map[string]string{
        "dev-vault":     "dev-secret-xyz",
        "staging-vault": "staging-secret-abc",
        "prod-vault":    "prod-secret-123",
    }

    // TODO: Return secret from map or error if not found
    return "", nil
}

// TODO 9: Implement maskSecret
func maskSecret(secret string) string {
    // TODO: Mask secret for logging (show only first 4 chars)
    return "***"
}

func main() {
    fmt.Println("🔐 Exercise 2: Variable Expansion System\n")

    // Test 1: Environment variable expansion
    fmt.Println("Test 1: Environment Variable Expansion")
    if err := loadFromEnvironment(); err != nil {
        log.Printf("   ❌ Error: %v\n", err)
    } else {
        fmt.Println("   ✅ Environment variables loaded and used")
    }

    // Test 2: Explicit variable maps
    fmt.Println("\nTest 2: Explicit Variable Maps")
    dev := NewEnvironment("dev", "https://httpbin.org", "dev-api-key")
    if err := dev.makeRequest("/get"); err != nil {
        log.Printf("   ❌ Error: %v\n", err)
    } else {
        fmt.Printf("   ✅ Request made to %s environment\n", dev.Name)
    }

    // Test 3: Environment manager
    fmt.Println("\nTest 3: Multi-Environment Manager")
    manager := NewEnvironmentManager()
    manager.AddEnvironment(NewEnvironment("dev", "https://httpbin.org", "dev-key"))
    manager.AddEnvironment(NewEnvironment("staging", "https://httpbin.org", "staging-key"))
    manager.AddEnvironment(NewEnvironment("prod", "https://httpbin.org", "prod-key"))

    // Switch environments and make requests
    for _, env := range []string{"dev", "staging", "prod"} {
        if err := manager.SwitchTo(env); err != nil {
            log.Printf("   ❌ Error switching to %s: %v\n", env, err)
            continue
        }
        if err := manager.MakeRequest("/get"); err != nil {
            log.Printf("   ❌ Error in %s: %v\n", env, err)
        } else {
            fmt.Printf("   ✅ Request successful in %s\n", env)
        }
    }

    // Test 4: Secure secrets
    fmt.Println("\nTest 4: Secure Secret Handling")
    secret, err := loadSecretsFromVault("prod-vault")
    if err != nil {
        log.Printf("   ❌ Error loading secret: %v\n", err)
    } else {
        masked := maskSecret(secret)
        fmt.Printf("   ✅ Secret loaded: %s\n", masked)
        fmt.Printf("   💡 Original secret never logged!\n")
    }

    fmt.Println("\n" + repeatString("=", 60))
    fmt.Println("Exercise complete! Check your implementation.")
    fmt.Println(repeatString("=", 60))
}

func repeatString(s string, count int) string {
    result := ""
    for i := 0; i < count; i++ {
        result += s
    }
    return result
}
```

## Self-Check Criteria

### Part 1: Environment Variables
- ✅ Uses CurlString() or CurlStringCommand()
- ✅ Variables use `${VAR}` syntax
- ✅ Automatic expansion from os.Getenv()

### Part 2: Explicit Variables
- ✅ Uses CurlCommandWithVars() or CurlWithVars()
- ✅ Creates `gocurl.Variables{}` map
- ✅ Does NOT use environment variables

### Part 3: Environment Manager
- ✅ Stores multiple environments
- ✅ Switches active environment
- ✅ Routes requests correctly

### Part 4: Secret Handling
- ✅ Loads from secure source
- ✅ Masks in logs
- ✅ Never prints full secrets

## Expected Output

```
Test 1: Environment Variable Expansion
   ✅ Environment variables loaded and used

Test 2: Explicit Variable Maps
   ✅ Request made to dev environment

Test 3: Multi-Environment Manager
   ✅ Request successful in dev
   ✅ Request successful in staging
   ✅ Request successful in prod

Test 4: Secure Secret Handling
   ✅ Secret loaded: prod***
   💡 Original secret never logged!
```

## Bonus Challenges

1. Add configuration file loading (JSON/YAML)
2. Implement environment inheritance (staging extends dev)
3. Add request caching per environment
4. Implement API key rotation
5. Add environment validation before requests

## Common Pitfalls

⚠️ **Pitfall 1:** Using environment variables when WithVars is required
- WithVars functions ignore environment variables
- Must use explicit variable maps

⚠️ **Pitfall 2:** Logging full secrets
- Always mask secrets before logging
- Use secure vault for production

⚠️ **Pitfall 3:** Hard-coding credentials
- Never hard-code in source code
- Use environment or vault

## Resources

- Chapter 3 section on variable expansion
- Example 02-variable-expansion
- Example 06-function-categories (WithVars usage)
