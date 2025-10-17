package gocurl

import (
	"crypto/tls"
	"testing"
)

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []uint16
		wantErr   bool
		errSubstr string
	}{
		{
			name:  "single cipher suite",
			input: "ECDHE-RSA-AES256-GCM-SHA384",
			want:  []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
		},
		{
			name:  "multiple cipher suites",
			input: "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256",
			want: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
		{
			name:  "with whitespace",
			input: " ECDHE-RSA-AES256-GCM-SHA384 : ECDHE-RSA-AES128-GCM-SHA256 ",
			want: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:      "unknown cipher suite",
			input:     "INVALID-CIPHER",
			wantErr:   true,
			errSubstr: "unknown cipher suite",
		},
		{
			name:      "one valid one invalid",
			input:     "ECDHE-RSA-AES256-GCM-SHA384:INVALID-CIPHER",
			wantErr:   true,
			errSubstr: "unknown cipher suite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCipherSuites(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCipherSuites() expected error, got nil")
					return
				}
				if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ParseCipherSuites() error = %v, want substring %v", err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseCipherSuites() unexpected error = %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ParseCipherSuites() got %d suites, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseCipherSuites() got[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseTLS13CipherSuites(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      []uint16
		wantErr   bool
		errSubstr string
	}{
		{
			name:  "single TLS 1.3 cipher",
			input: "TLS_AES_256_GCM_SHA384",
			want:  []uint16{tls.TLS_AES_256_GCM_SHA384},
		},
		{
			name:  "multiple TLS 1.3 ciphers",
			input: "TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256",
			want: []uint16{
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_AES_128_GCM_SHA256,
			},
		},
		{
			name:  "all TLS 1.3 ciphers",
			input: "TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256",
			want: []uint16{
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:      "unknown TLS 1.3 cipher",
			input:     "TLS_INVALID_CIPHER",
			wantErr:   true,
			errSubstr: "unknown TLS 1.3 cipher suite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTLS13CipherSuites(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseTLS13CipherSuites() expected error, got nil")
					return
				}
				if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ParseTLS13CipherSuites() error = %v, want substring %v", err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseTLS13CipherSuites() unexpected error = %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("ParseTLS13CipherSuites() got %d suites, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseTLS13CipherSuites() got[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint16
		wantErr bool
	}{
		{"TLS 1.0", "1.0", tls.VersionTLS10, false},
		{"TLS 1.1", "1.1", tls.VersionTLS11, false},
		{"TLS 1.2", "1.2", tls.VersionTLS12, false},
		{"TLS 1.3", "1.3", tls.VersionTLS13, false},
		{"invalid version", "2.0", 0, true},
		{"invalid format", "v1.2", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTLSVersion(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseTLSVersion() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseTLSVersion() unexpected error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ParseTLSVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSupportedCipherSuites(t *testing.T) {
	suites := GetSupportedCipherSuites()

	if len(suites) == 0 {
		t.Error("GetSupportedCipherSuites() returned empty list")
	}

	// Check that some expected ciphers are present
	expected := []string{
		"ECDHE-RSA-AES256-GCM-SHA384",
		"ECDHE-RSA-AES128-GCM-SHA256",
		"ECDHE-RSA-CHACHA20-POLY1305",
	}

	for _, exp := range expected {
		found := false
		for _, suite := range suites {
			if suite == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetSupportedCipherSuites() missing expected cipher: %s", exp)
		}
	}
}

func TestGetSupportedTLS13CipherSuites(t *testing.T) {
	suites := GetSupportedTLS13CipherSuites()

	if len(suites) == 0 {
		t.Error("GetSupportedTLS13CipherSuites() returned empty list")
	}

	// Check that expected TLS 1.3 ciphers are present
	expected := []string{
		"TLS_AES_256_GCM_SHA384",
		"TLS_AES_128_GCM_SHA256",
		"TLS_CHACHA20_POLY1305_SHA256",
	}

	for _, exp := range expected {
		found := false
		for _, suite := range suites {
			if suite == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetSupportedTLS13CipherSuites() missing expected cipher: %s", exp)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
