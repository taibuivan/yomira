// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/taibuivan/yomira/internal/platform/apperr"
)

// RedisResetTokenRepository implements ResetTokenRepository using Redis.
type RedisResetTokenRepository struct {
	client *redis.Client
}

// NewResetTokenRepository creates a new Redis-backed ResetTokenRepository.
func NewResetTokenRepository(client *redis.Client) *RedisResetTokenRepository {
	return &RedisResetTokenRepository{client: client}
}

/*
Set stores a reset token with its associated userID and TTL.

Parameters:
  - context: context.Context
  - token: string
  - userID: string
  - ttl: time.Duration

Returns:
  - error: Execution errors
*/
func (repository *RedisResetTokenRepository) Set(context context.Context, token string, userID string, ttl time.Duration) error {

	// Use constants for key prefix
	key := fmt.Sprintf("auth:reset_token:%s", token)

	// Set the token with TTL
	if err := repository.client.Set(context, key, userID, ttl).Err(); err != nil {
		return fmt.Errorf("redis_reset_token_set_failed: %w", err)
	}

	// Return nil on success
	return nil
}

/*
Get retrieves the userID for a given token.

Description: Returns apperr.NotFound if the token is absent or expired.

Parameters:
  - context: context.Context
  - token: string

Returns:
  - string: Original UserID
  - error: apperr.NotFound or connectivity errors
*/
func (repository *RedisResetTokenRepository) Get(context context.Context, token string) (string, error) {

	// Use constants for key prefix
	key := fmt.Sprintf("auth:reset_token:%s", token)

	// Get the token from Redis
	userID, err := repository.client.Get(context, key).Result()

	// Handle errors
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", apperr.NotFound("Reset token is invalid or expired")
		}
		return "", fmt.Errorf("redis_reset_token_get_failed: %w", err)
	}

	// Return the userID
	return userID, nil
}

/*
Delete removes the token from Redis.

Parameters:
  - context: context.Context
  - token: string

Returns:
  - error: Deletion failures
*/
func (repository *RedisResetTokenRepository) Delete(context context.Context, token string) error {

	// Use constants for key prefix
	key := fmt.Sprintf("auth:reset_token:%s", token)

	// Delete the token from Redis
	if err := repository.client.Del(context, key).Err(); err != nil {
		return fmt.Errorf("redis_reset_token_delete_failed: %w", err)
	}

	// Return nil on success
	return nil
}

// # Verification Token Repository

// RedisVerificationTokenRepository implements VerificationTokenRepository using Redis.
type RedisVerificationTokenRepository struct {
	client *redis.Client
}

// NewVerificationTokenRepository creates a new Redis-backed VerificationTokenRepository.
func NewVerificationTokenRepository(client *redis.Client) *RedisVerificationTokenRepository {
	return &RedisVerificationTokenRepository{client: client}
}

/*
Set stores a verification token with its associated userID and TTL.

Parameters:
  - context: context.Context
  - token: string
  - userID: string
  - ttl: time.Duration

Returns:
  - error: Storage failures
*/
func (repository *RedisVerificationTokenRepository) Set(context context.Context, token string, userID string, ttl time.Duration) error {

	// Use constants for key prefix
	key := fmt.Sprintf("auth:verify_token:%s", token)

	// Set the token with TTL
	if err := repository.client.Set(context, key, userID, ttl).Err(); err != nil {
		return fmt.Errorf("redis_verify_token_set_failed: %w", err)
	}

	// Return nil on success
	return nil
}

/*
Get retrieves the userID for a given token.

Description: Returns apperr.NotFound if the token is not present.

Parameters:
  - context: context.Context
  - token: string

Returns:
  - string: UserID
  - error: Resolution failures
*/
func (repository *RedisVerificationTokenRepository) Get(context context.Context, token string) (string, error) {

	// Use constants for key prefix
	key := fmt.Sprintf("auth:verify_token:%s", token)

	// Get the token from Redis
	userID, err := repository.client.Get(context, key).Result()

	// Handle errors
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", apperr.NotFound("Verification token is invalid or expired")
		}
		return "", fmt.Errorf("redis_verify_token_get_failed: %w", err)
	}

	// Return the userID
	return userID, nil
}

/*
Delete removes the token from Redis.

Parameters:
  - context: context.Context
  - token: string

Returns:
  - error: Execution failures
*/
func (repository *RedisVerificationTokenRepository) Delete(context context.Context, token string) error {

	// Use constants for key prefix
	key := fmt.Sprintf("auth:verify_token:%s", token)

	// Delete the token from Redis
	if err := repository.client.Del(context, key).Err(); err != nil {
		return fmt.Errorf("redis_verify_token_delete_failed: %w", err)
	}

	// Return nil on success
	return nil
}
