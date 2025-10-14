package options

import (
  "strings"
  "testing"
)

// TestBuilder_Validate tests the builder's Validate method
func TestBuilder_Validate_ValidOptions(t *testing.T) {
  builder := NewRequestOptionsBuilder().
    SetMethod("GET").
    SetURL("https://example.com").
    AddHeader("Content-Type", "application/json")

  err := builder.Validate()
  if err != nil {
    t.Errorf("Validate() should not return error for valid options, got: %v", err)
  }
}

func TestBuilder_Validate_InvalidMethod(t *testing.T) {
  builder := NewRequestOptionsBuilder().
    SetMethod("INVALID").
    SetURL("https://example.com")

  err := builder.Validate()
  if err == nil {
    t.Error("Validate() should return error for invalid HTTP method")
  }
  if !strings.Contains(err.Error(), "invalid HTTP method") {
    t.Errorf("Validate() error should mention 'invalid HTTP method', got: %v", err)
  }
}

func TestBuilder_Validate_EmptyURL(t *testing.T) {
  builder := NewRequestOptionsBuilder().
    SetMethod("GET")

  err := builder.Validate()
  if err == nil {
    t.Error("Validate() should return error for empty URL")
  }
  if !strings.Contains(err.Error(), "cannot be empty") {
    t.Errorf("Validate() error should mention 'cannot be empty', got: %v", err)
  }
}

func TestBuilder_Validate_ForbiddenHeader(t *testing.T) {
  builder := NewRequestOptionsBuilder().
    SetMethod("GET").
    SetURL("https://example.com").
    AddHeader("Host", "evil.com")

  err := builder.Validate()
  if err == nil {
    t.Error("Validate() should return error for forbidden header")
  }
  if !strings.Contains(err.Error(), "forbidden header") {
    t.Errorf("Validate() error should mention 'forbidden header', got: %v", err)
  }
}

func TestBuilder_Validate_BodyTooLarge(t *testing.T) {
  largeBody := strings.Repeat("a", 11*1024*1024) // 11MB

  builder := NewRequestOptionsBuilder().
    SetMethod("POST").
    SetURL("https://example.com").
    SetBody(largeBody)

  err := builder.Validate()
  if err == nil {
    t.Error("Validate() should return error for body exceeding limit")
  }
  if !strings.Contains(err.Error(), "too large") {
    t.Errorf("Validate() error should mention 'too large', got: %v", err)
  }
}

func TestBuilder_Validate_InsecureAuth(t *testing.T) {
  builder := NewRequestOptionsBuilder().
    SetMethod("GET").
    SetURL("http://example.com"). // HTTP not HTTPS
    SetBasicAuth("user", "pass")

  err := builder.Validate()
  if err == nil {
    t.Error("Validate() should return error for BasicAuth over HTTP")
  }
  if !strings.Contains(err.Error(), "insecure") {
    t.Errorf("Validate() error should mention 'insecure', got: %v", err)
  }
}

func TestBuilder_Validate_SecureAuth(t *testing.T) {
  builder := NewRequestOptionsBuilder().
    SetMethod("GET").
    SetURL("https://example.com"). // HTTPS
    SetBasicAuth("user", "pass")

  err := builder.Validate()
  if err != nil {
    t.Errorf("Validate() should not return error for BasicAuth over HTTPS, got: %v", err)
  }
}
