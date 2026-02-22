// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package sec

import (
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
