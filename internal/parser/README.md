# GoCurl Command Parsing and Tokenization

## Overview

The GoCurl library provides a robust system for parsing and tokenizing curl commands, allowing Go applications to easily interpret and execute curl-like requests. This document outlines the requirements, functionality, and usage of the command parsing and tokenization components.

## Features

- Parse curl-like command strings into structured request options
- Support for variable substitution within command strings
- Tokenization of complex command strings, respecting quotes and escapes
- Handling of various curl options and flags
- Conversion of parsed commands into executable request structures

## Requirements

- Go 1.15 or higher
- No external dependencies

## Command Parser

### Functionality

The command parser is responsible for converting a curl-like command string into a structured set of request options. It performs the following tasks:

1. Tokenizes the input string into discrete elements (flags, values, variables)
2. Interprets curl flags and their corresponding values
3. Substitutes variables with their actual values from a provided variables map
4. Constructs a `RequestOptions` struct that can be used to execute the request

### Supported Curl Options

The parser recognizes and interprets the following curl options:

- `-X, --request`: HTTP method (GET, POST, PUT, etc.)
- `-H, --header`: HTTP headers
- `-d, --data`: Request body for POST/PUT requests
- `-F, --form`: Form data
- `-u, --user`: Basic authentication
- `--max-time`: Request timeout
- `-L, --location`: Follow redirects
- `--max-redirs`: Maximum number of redirects to follow
- `--compressed`: Request compressed response
- `--cert`: Client certificate file
- `--key`: Private key file
- `-k, --insecure`: Allow insecure SSL connections
- `-A, --user-agent`: User agent string

### Variable Substitution

The parser supports variable substitution within the command string. Variables can be specified in two formats:

- `$VARIABLE_NAME`
- `${VARIABLE_NAME}`

Variables can be used in any part of the command string, including URLs, headers, and data.

## Tokenizer

### Functionality

The tokenizer breaks down the input command string into a series of tokens, each representing a distinct element of the command. It handles:

- Curl flags (e.g., `-X`, `-H`)
- Values (including quoted strings)
- Variables (in both `$VAR` and `${VAR}` formats)

### Token Types

The tokenizer produces three types of tokens:

1. `TokenFlag`: Represents a curl option flag
2. `TokenValue`: Represents a value or argument
3. `TokenVariable`: Represents a variable to be substituted

### Handling of Special Cases

The tokenizer is designed to handle several special cases:

- Quoted strings: Preserves spaces and special characters within quotes
- Escaped characters: Properly handles escaped quotes and other characters
- Mixed content: Correctly tokenizes strings that mix variables with other content

## Usage

### Parsing a Curl Command

To parse a curl command string:

```go
command := "curl -X POST -H 'Content-Type: $CONTENT_TYPE' -d '{\"key\":\"$VALUE\"}' $API_URL/data"
variables := gocurl.Variables{
    "CONTENT_TYPE": "application/json",
    "VALUE": "example",
    "API_URL": "https://api.example.com",
}

requestOptions, err := gocurl.ParseCurlCommand(command, variables)
if err != nil {
    // Handle error
}

// Use requestOptions to make the HTTP request
```

### Accessing Parsed Data

The `RequestOptions` struct provides access to all parsed data:

```go
fmt.Println(requestOptions.Method)  // "POST"
fmt.Println(requestOptions.URL)     // "https://api.example.com/data"
fmt.Println(requestOptions.Headers) // map[string]string{"Content-Type": "application/json"}
fmt.Println(requestOptions.Body)    // "{\"key\":\"example\"}"
```

## Error Handling

The parser and tokenizer provide detailed error messages for various failure scenarios:

- Malformed command strings
- Undefined variables
- Invalid option usage

Errors are returned as standard Go error types and should be checked after calling `ParseCurlCommand`.

## Limitations

- The parser does not support all curl options. Unsupported options are ignored.
- Variable substitution occurs at parse time. Dynamic variable changes after parsing are not reflected.
- The parser does not execute the HTTP request. It only provides the structured data for making the request.

## Contributing

Contributions to improve the command parsing and tokenization system are welcome. Please submit pull requests with tests for new features or bug fixes.

## License

This project is licensed under the MIT License - see the LICENSE file for details.