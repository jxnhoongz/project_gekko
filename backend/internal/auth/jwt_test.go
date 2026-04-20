package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAndVerify(t *testing.T) {
	signer := NewJWTSigner("test-secret-xyz", time.Hour)

	tok, err := signer.Issue(42, "admin@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	claims, err := signer.Verify(tok)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.AdminID)
	assert.Equal(t, "admin@example.com", claims.Email)
}

func TestVerify_wrongSecret(t *testing.T) {
	a := NewJWTSigner("secret-a", time.Hour)
	b := NewJWTSigner("secret-b", time.Hour)

	tok, err := a.Issue(1, "x@y.z")
	require.NoError(t, err)

	_, err = b.Verify(tok)
	require.Error(t, err)
}

func TestVerify_expired(t *testing.T) {
	signer := NewJWTSigner("s", -time.Second)
	tok, err := signer.Issue(1, "x@y.z")
	require.NoError(t, err)

	_, err = signer.Verify(tok)
	require.Error(t, err)
}

func TestVerify_garbage(t *testing.T) {
	signer := NewJWTSigner("s", time.Hour)
	_, err := signer.Verify("not.a.jwt")
	require.Error(t, err)
}
