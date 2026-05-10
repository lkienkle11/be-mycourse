package constants

import "time"

// UserMeTTL is the Redis TTL for cached GET /api/v1/me payloads and login invalid-credentials keys.
const UserMeTTL = time.Minute

// LoginEmailUserIDTTL is how long we cache a successful email→user_id resolution for login.
const LoginEmailUserIDTTL = 30 * time.Second

const (
	RedisKeyUserMePrefix           = "mycourse:user:me:"
	RedisKeyLoginInvalidPrefix     = "mycourse:auth:login:invalid:"
	RedisKeyLoginUserByEmailPrefix = "mycourse:auth:login:user_by_email:"
	// RedisKeyRegisterConfirmEmailWindowPrefix — ZSET of successful-send timestamps (score = unix ms) per user id.
	RedisKeyRegisterConfirmEmailWindowPrefix = "mycourse:auth:register:confirm_email_window:"
)
