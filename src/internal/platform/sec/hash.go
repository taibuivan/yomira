// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package sec

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain-text password using the bcrypt algorithm.
func HashPassword(plainTextPassword string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(plainTextPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("auth: failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// CheckPasswordHash compares a plain-text password with its hashed version.
func CheckPasswordHash(plainTextPassword, existingHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(plainTextPassword))
	return err == nil
}

// GenerateSecureToken creates a cryptographically secure random token.
// Used for refresh tokens, password reset tokens, etc.
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("sec: failed to generate random token: %w", err)
	}
	// Return a URL-safe, unpadded base64 string
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HashToken generates a SHA-256 hash of a string.
// Used to safely store tokens in the database without saving the raw value.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
