// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package auth

import "time"

// # Authentication Constraints

const (
	// AccessTokenTTL is the duration a JWT access token remains valid.
	// We keep it short (15m) to minimize the impact of a leaked token.
	AccessTokenTTL = 15 * time.Minute

	// RefreshTokenTTL is the duration a session/refresh token remains valid.
	// Long-lived (30 days) to provide a good user experience.
	RefreshTokenTTL = 30 * 24 * time.Hour

	// RefreshTokenLength is the byte length of the random secure token.
	RefreshTokenLength = 32

	// ResetTokenTTL is the duration a password reset token remains valid.
	// Short-lived (1 hour) for security.
	ResetTokenTTL = 1 * time.Hour

	// ResetTokenLength is the byte length of the random password reset token.
	ResetTokenLength = 32

	// VerificationTokenTTL is the duration an email verification token remains valid.
	// Long-lived (24 hours) as users might not check email immediately.
	VerificationTokenTTL = 24 * time.Hour

	// VerificationTokenLength is the byte length of the random verification token.
	VerificationTokenLength = 32
)
