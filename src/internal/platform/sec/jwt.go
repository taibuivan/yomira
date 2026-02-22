// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package auth provides cryptographic primitives and token management.
//
// # Architecture
//
// This package isolates security-sensitive code (Hashing, JWT Signing) from
// the domain logic. It acts as an Infrastructure service injected into the
// Application layer via the [TokenProvider] interface.
package sec

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthClaims represents the payload embedded inside a JWT Access Token.
//
// # Why custom claims?
//
// By embedding the UserID, Username, and Role directly inside the JWT,
// the [middleware.Authenticate] can reconstruct the active user context
// WITHOUT querying the database on every single API request. This provides
// massive read-scalability.
type AuthClaims struct {
	jwt.RegisteredClaims

	// Custom application claims are abbreviated to keep the JWT payload small.
	UserID   string `json:"uid"`
	Username string `json:"unm"`
	Role     string `json:"rol"`
}

// TokenService handles generation and verification of JWT tokens using RS256.
type TokenService struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
}

// NewTokenService creates a new TokenService.
// It reads RSA keys from the provided filesystem paths.
func NewTokenService(privateKeyPath, publicKeyPath, issuer string) (*TokenService, error) {
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to read private key from %s: %w", privateKeyPath, err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to parse private key: %w", err)
	}

	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("auth: failed to read public key from %s: %w", publicKeyPath, err)
	}

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

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(service.privateKey)
	if err != nil {
		return "", fmt.Errorf("auth: failed to sign token: %w", err)
	}

	return signedToken, nil
}

// VerifyToken checks the signature and validity of a JWT string.
func (service *TokenService) VerifyToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method: %v", token.Header["alg"])
		}
		return service.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("auth: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*AuthClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("auth: invalid token claims")
	}

	return claims, nil
}
