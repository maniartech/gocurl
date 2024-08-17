# GoCurl: Universal REST API Client for Go

> NOT - READY - YET

## Table of Contents

- [GoCurl: Universal REST API Client for Go](#gocurl-universal-rest-api-client-for-go)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
  - [Vision](#vision)
  - [Features](#features)
  - [Use Cases](#use-cases)
  - [Installation](#installation)
  - [Usage](#usage)
    - [As a Library](#as-a-library)
    - [Command-Line Interface](#command-line-interface)
  - [API Reference](#api-reference)
    - [Main Components](#main-components)
    - [Key Functions](#key-functions)
  - [Security Considerations](#security-considerations)
  - [Advanced Usage](#advanced-usage)
  - [Performance Considerations](#performance-considerations)
  - [Contributing](#contributing)
  - [License](#license)

## Introduction

GoCurl is a powerful, flexible, and secure HTTP client library for Go that aims to be the universal translator between Go applications and REST APIs of any platform. By emulating and extending the functionality of the popular curl command-line tool, GoCurl provides a familiar and intuitive interface for Go developers to interact with any REST API, regardless of the platform or language it was built in.

## Vision

In the diverse landscape of web services and microservices, many platforms expose REST APIs but don't provide official Go client libraries. This gap often forces Go developers to either write custom client code for each API they interact with or use generic HTTP clients that lack the convenience of purpose-built libraries.

GoCurl aims to bridge this gap by providing:

1. A universal client that can interact with any REST API using curl-like syntax.
2. Easy translation of curl commands (often provided in API documentation) into Go code.
3. A consistent interface for working with diverse APIs in Go projects.
4. Tools to generate Go structs from API responses for type-safe interactions.
5. Middleware and plugins to extend functionality for specific API requirements.

By doing so, GoCurl seeks to make any REST API easily consumable in Go-based projects, significantly reducing the time and effort required to integrate new services.

## Features

- Support for all HTTP methods (GET, POST, PUT, DELETE, PATCH, etc.)
- HTTP/1.1 and HTTP/2 protocol support
- Custom header management
- File upload capabilities (multipart/form-data)
- Automatic cookie handling
- Flexible authentication support (Basic, Bearer Token, OAuth, etc.)
- Proxy support (HTTP and SOCKS5)
- Custom TLS configuration
- Timeout and retry mechanisms
- Compression support (gzip, deflate)
- Variable substitution in curl-like commands
- Response parsing and struct generation
- Middleware support for request/response modification
- Plugin system for extending functionality
- Comprehensive security features

## Use Cases

1. **API Exploration**: Quickly test and explore new APIs using curl-like commands.
2. **Service Integration**: Easily integrate diverse microservices into Go applications.
3. **Legacy API Modernization**: Wrap older REST APIs with a modern Go interface.
4. **Cross-Platform Development**: Use the same client library across different Go projects interacting with various APIs.
5. **API Response Mocking**: Generate mock responses for testing based on actual API interactions.

## Installation

To use GoCurl as a library in your Go project:

```bash
go get github.com/maniartech/gocurl
```

To install the command-line tool:

```bash
go install github.com/maniartech/gocurl/cmd/gocurl@latest
```

## Usage

### As a Library

```go
import "github.com/maniartech/gocurl"

func main() {
    variables := gocurl.Variables{
        "api_key": os.Getenv("API_KEY"),
    }

    response, err := gocurl.Request([]string{
        "curl",
        "-X", "GET",
        "-H", "Authorization: Bearer ${api_key}",
        "https://api.example.com/data",
    }, variables)

    if err != nil {
        log.Fatalf("Error: %v", err)
    }
    fmt.Println(response)

    // Parse response into a struct
    var data MyDataStruct
    if err := gocurl.ParseJSON(response, &data); err != nil {
        log.Fatalf("Error parsing response: %v", err)
    }
    // Use the structured data...
}
```

### Command-Line Interface

```bash
gocurl -X GET -H "Authorization: Bearer $API_KEY" https://api.example.com/data
```

## API Reference

### Main Components

- `CurlCommand`: Represents a curl command with all its options.
- `RequestOptions`: Contains all the options for making an HTTP request.
- `Variables`: A map type for variable substitution in commands.
- `Middleware`: Interface for creating request/response modification middleware.
- `Plugin`: Interface for creating plugins to extend functionality.

### Key Functions

- `ParseCurlCommand(args []string, variables Variables) (*RequestOptions, error)`
- `Request(args []string, variables Variables) (string, error)`
- `(c *CurlCommand) Execute() (*http.Response, error)`
- `ParseJSON(data string, v interface{}) error`
- `GenerateStruct(jsonData string) (string, error)`

(Detailed API documentation to be generated and linked)

## Security Considerations

GoCurl implements several security measures:

1. Input Validation: Strict validation of input arguments and Variables values.
2. Escaping: Proper escaping of substituted values to prevent injection attacks.
3. Sensitive Data Handling: Redaction of sensitive information in logs.
4. TLS Security: Support for custom TLS configurations.
5. Authentication: Flexible support for various authentication methods.

Users should be aware of:
- Proper handling of sensitive data in Variables
- Risks associated with following redirects to untrusted hosts
- Importance of using HTTPS for secure communications

## Advanced Usage

1. Creating Custom Middleware
2. Developing Plugins for Specific APIs
3. Generating Go Structs from API Responses
4. Implementing Complex Authentication Flows
5. Handling Paginated API Responses

(Detailed examples for each to be provided)

## Performance Considerations

- Connection Pooling
- Keep-Alive Connections
- Efficient Memory Usage for Large Responses
- Concurrent API Requests

## Contributing

We welcome contributions! Please see our [CONTRIBUTING.md](CONTRIBUTING.md) file for details on submitting pull requests, suggesting improvements, or reporting bugs.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

GoCurl is committed to evolving with the needs of the Go community and the ever-changing landscape of web APIs. We encourage feedback, contributions, and collaboration to make GoCurl the go-to solution for consuming REST APIs in Go projects.