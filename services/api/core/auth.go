package core

import (
	"crypto/subtle"
	"fmt"
	"strings"
)

// AuthResult contains the result of an authentication check.
type AuthResult struct {
	Valid    bool
	TenantID string
	Actor    string
	Error    string
}

// ValidateAPIKey validates an API key format and returns the tenant ID.
// In production, this should validate against a database or external auth provider.
// For now, it validates the bearer token format and extracts tenant context.
func ValidateAPIKey(authorization string, tenantID string) AuthResult {
	if authorization == "" {
		return AuthResult{Error: "missing Authorization header"}
	}

	if !strings.HasPrefix(authorization, "Bearer ") {
		return AuthResult{Error: "Authorization header must use Bearer scheme"}
	}

	token := strings.TrimPrefix(authorization, "Bearer ")
	if token == "" {
		return AuthResult{Error: "empty bearer token"}
	}

	// Validate token length (reject obviously invalid tokens)
	if len(token) < 16 {
		return AuthResult{Error: "bearer token too short (minimum 16 characters)"}
	}

	// Reject tokens with suspicious characters
	for _, c := range token {
		if !isValidTokenChar(c) {
			return AuthResult{Error: "bearer token contains invalid characters"}
		}
	}

	// In production, validate against JWT or API key store
	// For now, accept well-formed tokens and bind to tenant
	if tenantID == "" {
		return AuthResult{Error: "tenant ID required"}
	}

	return AuthResult{Valid: true, TenantID: tenantID}
}

// ValidateJWT validates a JWT token (placeholder for production implementation).
// In production, replace with proper JWT validation using a library like golang-jwt.
func ValidateJWT(token string, secret string) (map[string]interface{}, error) {
	if secret == "" {
		return nil, fmt.Errorf("JWT secret not configured")
	}

	// Placeholder: in production, parse and validate JWT
	// using golang-jwt/jwt/v5 or similar library
	_ = token
	_ = secret

	return nil, fmt.Errorf("JWT validation not implemented — use API key authentication")
}

// SecureCompare performs a constant-time string comparison.
func SecureCompare(a, b string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// SanitizeTenantID removes potentially dangerous characters from tenant IDs.
func SanitizeTenantID(id string) string {
	// Only allow alphanumeric, hyphens, underscores, dots
	var cleaned strings.Builder
	for _, c := range id {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			cleaned.WriteRune(c)
		}
	}
	return cleaned.String()
}

// isValidTokenChar checks if a character is valid in a bearer token.
func isValidTokenChar(c rune) bool {
	// Allow alphanumeric, hyphens, underscores, dots, colons
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':'
}
