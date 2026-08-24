package core

import (
	"testing"
)

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name       string
		auth       string
		tenantID   string
		wantValid  bool
		wantErr    bool
	}{
		{
			name:      "valid bearer token",
			auth:      "Bearer test-api-key-12345678",
			tenantID:  "tenant-1",
			wantValid: true,
		},
		{
			name:      "valid token with special chars",
			auth:      "Bearer abc.def-ghi_123:jkl",
			tenantID:  "tenant-1",
			wantValid: true,
		},
		{
			name:      "missing auth header",
			auth:      "",
			tenantID:  "tenant-1",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "missing tenant ID",
			auth:      "Bearer test-api-key-12345678",
			tenantID:  "",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "not bearer scheme",
			auth:      "Basic dXNlcjpwYXNz",
			tenantID:  "tenant-1",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "empty bearer token",
			auth:      "Bearer ",
			tenantID:  "tenant-1",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "token too short",
			auth:      "Bearer abc",
			tenantID:  "tenant-1",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "token with invalid characters",
			auth:      "Bearer test token with spaces",
			tenantID:  "tenant-1",
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAPIKey(tt.auth, tt.tenantID)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateAPIKey() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
			if tt.wantErr && result.Error == "" {
				t.Error("ValidateAPIKey() expected error but got none")
			}
		})
	}
}

func TestSanitizeTenantID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"tenant-1", "tenant-1"},
		{"tenant_1", "tenant_1"},
		{"tenant.vault.io", "tenant.vault.io"},
		{"Tenant123", "Tenant123"},
		{"tenant<script>", "tenantscript"},
		{"tenant; DROP TABLE", "tenantDROPTABLE"},
		{"tenant/../../../etc", "tenant......etc"},
		{"", ""},
		{"valid-id_123.test", "valid-id_123.test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeTenantID(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeTenantID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSecureCompare(t *testing.T) {
	if !SecureCompare("same", "same") {
		t.Error("SecureCompare('same', 'same') should return true")
	}
	if SecureCompare("a", "b") {
		t.Error("SecureCompare('a', 'b') should return false")
	}
	if !SecureCompare("", "") {
		t.Error("SecureCompare('', '') should return true (empty strings are equal)")
	}
}

func TestIsValidTokenChar(t *testing.T) {
	valid := []rune{'a', 'z', 'A', 'Z', '0', '9', '-', '_', '.', ':'}
	for _, c := range valid {
		if !isValidTokenChar(c) {
			t.Errorf("isValidTokenChar(%q) should be true", c)
		}
	}

	invalid := []rune{' ', '!', '@', '#', '/', '\\', '\n', '\t'}
	for _, c := range invalid {
		if isValidTokenChar(c) {
			t.Errorf("isValidTokenChar(%q) should be false", c)
		}
	}
}
