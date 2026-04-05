package cache

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"mycourse-io-be/cache_clients"
	"mycourse-io-be/dto"
)

// UserMeTTL is the Redis TTL for cached GET /api/v1/me payloads and login invalid-credentials keys.
const UserMeTTL = time.Minute

// LoginEmailUserIDTTL is how long we remember a successful email→user_id resolution for login (avoids repeated lookups by email).
const LoginEmailUserIDTTL = 30 * time.Second

const (
	redisKeyUserMePrefix           = "mycourse:user:me:"
	redisKeyLoginInvalidPrefix     = "mycourse:auth:login:invalid:"
	redisKeyLoginUserByEmailPrefix = "mycourse:auth:login:user_by_email:"
)

// NormalizeLoginEmail trims and lower-cases an email for cache key use.
func NormalizeLoginEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func redisUserMeKey(userID uint) string {
	return redisKeyUserMePrefix + strconv.FormatUint(uint64(userID), 10)
}

func redisLoginInvalidKey(normEmail string) string {
	return redisKeyLoginInvalidPrefix + normEmail
}

func redisLoginUserByEmailKey(normEmail string) string {
	return redisKeyLoginUserByEmailPrefix + normEmail
}

// GetCachedLoginUserID returns the cached internal user id for a normalized login email, if any.
func GetCachedLoginUserID(ctx context.Context, normEmail string) (uint, bool) {
	if !cache_clients.RedisAvailable() || normEmail == "" {
		return 0, false
	}
	s, err := cache_clients.Redis.Get(ctx, redisLoginUserByEmailKey(normEmail)).Result()
	if err != nil {
		return 0, false
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

// SetCachedLoginUserID stores a short-lived email→user_id mapping after Postgres resolved the account.
func SetCachedLoginUserID(ctx context.Context, normEmail string, userID uint) {
	if !cache_clients.RedisAvailable() || normEmail == "" {
		return
	}
	_ = cache_clients.Redis.Set(ctx, redisLoginUserByEmailKey(normEmail),
		strconv.FormatUint(uint64(userID), 10), LoginEmailUserIDTTL).Err()
}

// GetCachedUserMe returns a cached dto.MeResponse when Redis has a valid entry for userID.
func GetCachedUserMe(ctx context.Context, userID uint) (*dto.MeResponse, bool) {
	if !cache_clients.RedisAvailable() {
		return nil, false
	}
	data, err := cache_clients.Redis.Get(ctx, redisUserMeKey(userID)).Bytes()
	if err != nil {
		return nil, false
	}
	var me dto.MeResponse
	if err := json.Unmarshal(data, &me); err != nil {
		return nil, false
	}
	if me.UserID != userID {
		return nil, false
	}
	return &me, true
}

// SetCachedUserMe stores the /me JSON payload under mycourse:user:me:{id}.
func SetCachedUserMe(ctx context.Context, me *dto.MeResponse) {
	if !cache_clients.RedisAvailable() || me == nil {
		return
	}
	data, err := json.Marshal(me)
	if err != nil {
		return
	}
	_ = cache_clients.Redis.Set(ctx, redisUserMeKey(me.UserID), data, UserMeTTL).Err()
}

// LoginInvalidCached is true when this normalized email was recently rejected with InvalidCredentials.
func LoginInvalidCached(ctx context.Context, normEmail string) bool {
	if !cache_clients.RedisAvailable() || normEmail == "" {
		return false
	}
	n, err := cache_clients.Redis.Exists(ctx, redisLoginInvalidKey(normEmail)).Result()
	return err == nil && n > 0
}

// SetLoginInvalidCache records a short-lived invalid login outcome for normEmail.
func SetLoginInvalidCache(ctx context.Context, normEmail string) {
	if !cache_clients.RedisAvailable() || normEmail == "" {
		return
	}
	_ = cache_clients.Redis.Set(ctx, redisLoginInvalidKey(normEmail), "1", UserMeTTL).Err()
}

// DelLoginInvalidCache clears the invalid-login key (e.g. after successful login).
func DelLoginInvalidCache(ctx context.Context, normEmail string) {
	if !cache_clients.RedisAvailable() || normEmail == "" {
		return
	}
	_ = cache_clients.Redis.Del(ctx, redisLoginInvalidKey(normEmail)).Err()
}
