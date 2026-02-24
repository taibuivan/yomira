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

// # Password Security (Bcrypt)

// HashPassword hashes a plain-text password using the bcrypt algorithm.
func HashPassword(plainTextPassword string) (string, error) {

	// Default cost (10) provides a good balance between security and performance
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(plainTextPassword), bcrypt.DefaultCost)

	if err != nil {
		return "", fmt.Errorf("auth: failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// CheckPasswordHash compares a plain-text password with its hashed version.
func CheckPasswordHash(plainTextPassword, existingHash string) bool {

	// Bcrypt handles salt automatically. comparison is constant-time to prevent timing attacks.
	err := bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(plainTextPassword))

	return err == nil
}

// # Token Security (CSPRNG & SHA-256)

// GenerateSecureToken creates a cryptographically secure random token.
// Used for refresh tokens, password reset tokens, etc.
func GenerateSecureToken(length int) (string, error) {

	// Use crypto/rand for cryptographically strong random numbers
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("sec: failed to generate random token: %w", err)
	}

	// Return a URL-safe, unpadded base64 string for cookie/URL compatibility
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HashToken generates a SHA-256 hash of a string.
// Used to safely store tokens in the database without saving the raw values.
func HashToken(token string) string {

	// We hash volatile tokens (like Refresh Tokens) before DB storage
	// to prevent leakage if the database is compromised.
	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])
}
