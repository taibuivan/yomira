// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package sec provides cryptographic primitives and identity security services.

It encapsulates sensitive operations like password hashing, token signing, and
role-based access control (RBAC) levels.

Core Components:

  - JWT: RS256-signed tokens for stateless authentication.
  - Hash: Secure password derivation using Bcrypt/Argon2.
  - Role: Hierarchy logic for privilege escalation checks.

The package enforces a strict boundary between infrastructure-level security
and high-level business logic.
*/
package sec

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// # Identity Claims

// AuthClaims represents the payload embedded inside a JWT Access Token.
type AuthClaims struct {
	jwt.RegisteredClaims

	// Custom application claims are abbreviated to keep the JWT payload small.
	UserID   string `json:"uid"`
	Username string `json:"unm"`
	Role     string `json:"rol"`
}

// IsAdmin checks if the user has administrative privileges.
func (c *AuthClaims) IsAdmin() bool {
	return UserRole(c.Role) == RoleAdmin
}

// # Token Provider (RSA)

// TokenService handles generation and verification of JWT tokens using RS256.
type TokenService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
}

// NewTokenService creates a new TokenService.
func NewTokenService(privateKeyPath, publicKeyPath, issuer string) (*TokenService, error) {

	// Load the Private Key for signing
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to read private key from %s: %w", privateKeyPath, err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to parse private key: %w", err)
	}

	// Load the Public Key for verification
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to read public key from %s: %w", publicKeyPath, err)
	}

	// Parse the public key
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to parse public key: %w", err)
	}

	return &TokenService{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
	}, nil
}

// GenerateAccessToken creates a new JWT access token for a user.
func (service *TokenService) GenerateAccessToken(userID, username, role string, timeToLive time.Duration) (string, error) {

	currentTime := time.Now()

	// Construct the claims with standard Registered claims (iss, sub, iat, exp)
	claims := AuthClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    service.issuer,
			IssuedAt:  jwt.NewNumericDate(currentTime),
			ExpiresAt: jwt.NewNumericDate(currentTime.Add(timeToLive)),
		},
		UserID:   userID,
		Username: username,
		Role:     role,
	}

	// Sign the token using the RS256 algorithm (Asymmetric)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(service.privateKey)

	if err != nil {
		return "", fmt.Errorf("auth: failed to sign token: %w", err)
	}

	return signedToken, nil
}

// VerifyToken checks the signature and validity of a JWT string.
func (service *TokenService) VerifyToken(tokenString string) (*AuthClaims, error) {

	// Parse the token and validate the signing method
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {

		// Ensure the token use RSA as the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", token.Header["alg"])
		}

		return service.publicKey, nil
	})

	// Handle parsing/validation errors (e.g. expired, malformed)
	if err != nil {
		return nil, fmt.Errorf("auth: invalid token: %w", err)
	}

	// Extract the claims and check the 'Valid' flag
	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("auth: invalid token claims")
	}

	return claims, nil
}
