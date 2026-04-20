package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordLen = 8

func HashPassword(plain string) (string, error) {
	if len(plain) < minPasswordLen {
		return "", errors.New("password must be at least 8 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
