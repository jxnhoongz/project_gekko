package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerify(t *testing.T) {
	plain := "hunter2-longer-than-8-chars"
	hash, err := HashPassword(plain)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, plain, hash)

	assert.True(t, VerifyPassword(hash, plain))
	assert.False(t, VerifyPassword(hash, "wrong-password"))
}

func TestHashPassword_tooShort(t *testing.T) {
	_, err := HashPassword("short")
	require.Error(t, err)
}

func TestVerifyPassword_malformedHash(t *testing.T) {
	assert.False(t, VerifyPassword("not-a-bcrypt-hash", "anything"))
}
