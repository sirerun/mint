package gcp

import (
	"context"
	"testing"
)

// Compile-time interface check.
var _ SecretClient = (*SecretManagerAdapter)(nil)

func TestSecretManagerAdapterImplementsSecretClient(t *testing.T) {
	// Verified at compile time by the var _ line above.
}

func TestNewSecretManagerAdapterSignature(t *testing.T) {
	// Verify the constructor accepts a context and returns the expected types.
	var fn func(context.Context) (*SecretManagerAdapter, error) = NewSecretManagerAdapter
	_ = fn
}
