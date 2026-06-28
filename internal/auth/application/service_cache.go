package application

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/uuidx"
)

func (s *AuthService) redisKey(prefix string, id interface{}) string {
	switch v := id.(type) {
	case string:
		return prefix + v
	case uint:
		return prefix + strconv.FormatUint(uint64(v), 10)
	}
	return prefix
}

func (s *AuthService) getCachedMe(ctx context.Context, userID string) (*domain.MeProfile, bool) {
	if s.redis == nil {
		return nil, false
	}
	data, err := s.redis.Get(ctx, s.redisKey(RedisKeyUserMePrefix, userID)).Bytes()
	if err != nil {
		return nil, false
	}
	var me domain.MeProfile
	if err := json.Unmarshal(data, &me); err != nil || me.UserID != userID {
		return nil, false
	}
	return &me, true
}

func (s *AuthService) setCachedMe(ctx context.Context, me *domain.MeProfile) {
	if s.redis == nil || me == nil {
		return
	}
	data, _ := json.Marshal(me)
	_ = s.redis.Set(ctx, s.redisKey(RedisKeyUserMePrefix, me.UserID), data, UserMeTTL).Err()
}

func (s *AuthService) delCachedMe(ctx context.Context, userID string) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, s.redisKey(RedisKeyUserMePrefix, userID)).Err()
	}
}

// InvalidateUserMeCache drops the cached /me payload for userID (e.g. after RBAC role change).
func (s *AuthService) InvalidateUserMeCache(ctx context.Context, userID string) {
	s.delCachedMe(ctx, userID)
}

func (s *AuthService) getCachedLoginUserID(ctx context.Context, normEmail string) (string, bool) {
	if s.redis == nil || normEmail == "" {
		return "", false
	}
	v, err := s.redis.Get(ctx, s.redisKey(RedisKeyLoginUserByEmailPrefix, normEmail)).Result()
	if err != nil || v == "" {
		return "", false
	}
	// Accept legacy JSON entries from a prior cache format; only user_id is used.
	var legacy struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(v), &legacy); err == nil && legacy.UserID != "" {
		return legacy.UserID, true
	}
	return v, true
}

func (s *AuthService) setCachedLoginUserID(ctx context.Context, normEmail, userID string) {
	if s.redis == nil || normEmail == "" || userID == "" {
		return
	}
	_ = s.redis.Set(ctx, s.redisKey(RedisKeyLoginUserByEmailPrefix, normEmail), userID, LoginEmailUserIDTTL).Err()
}

func (s *AuthService) loginInvalidCached(ctx context.Context, normEmail string) bool {
	if s.redis == nil || normEmail == "" {
		return false
	}
	n, err := s.redis.Exists(ctx, s.redisKey(RedisKeyLoginInvalidPrefix, normEmail)).Result()
	return err == nil && n > 0
}

func (s *AuthService) setLoginInvalidCache(ctx context.Context, normEmail string) {
	if s.redis == nil || normEmail == "" {
		return
	}
	_ = s.redis.Set(ctx, s.redisKey(RedisKeyLoginInvalidPrefix, normEmail), "1", UserMeTTL).Err()
}

func (s *AuthService) delLoginInvalidCache(ctx context.Context, normEmail string) {
	if s.redis != nil && normEmail != "" {
		_ = s.redis.Del(ctx, s.redisKey(RedisKeyLoginInvalidPrefix, normEmail)).Err()
	}
}

func (s *AuthService) delLoginUserByEmail(ctx context.Context, normEmail string) {
	if s.redis != nil && normEmail != "" {
		_ = s.redis.Del(ctx, s.redisKey(RedisKeyLoginUserByEmailPrefix, normEmail)).Err()
	}
}

func (s *AuthService) delRegisterWindowKey(ctx context.Context, userID string) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, s.redisKey(RedisKeyRegisterConfirmEmailWindowPrefix, userID)).Err()
	}
}

func (s *AuthService) tryReserveEmailSend(ctx context.Context, userID string) (bool, time.Duration, string, error) {
	if s.redis == nil {
		return true, 0, "", nil
	}
	member := uuidx.NewV4()
	now := time.Now().UnixMilli()
	win := RegisterConfirmationEmailWindow.Milliseconds()
	max := int64(MaxRegisterConfirmationEmailsPerWindow)
	script := redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local max = tonumber(ARGV[3])
local member = ARGV[4]
local minscore = now - window
redis.call('ZREMRANGEBYSCORE', key, '-inf', minscore)
redis.call('ZADD', key, now, member)
local n = redis.call('ZCARD', key)
redis.call('PEXPIRE', key, window + 120000)
if n > max then
  redis.call('ZREM', key, member)
  local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
  local retry = 0
  if oldest[2] ~= nil then
    retry = tonumber(oldest[2]) + window - now
    if retry < 0 then retry = 0 end
  end
  return {0, retry}
end
return {1, 0}
`)
	key := RedisKeyRegisterConfirmEmailWindowPrefix + userID
	r, err := script.Run(ctx, s.redis, []string{key}, now, win, max, member).Slice()
	if err != nil {
		return false, 0, "", err
	}
	if len(r) < 2 {
		return false, 0, "", nil
	}
	flag, _ := toInt64(r[0])
	retryMs, _ := toInt64(r[1])
	if flag == 0 {
		return false, time.Duration(retryMs) * time.Millisecond, "", nil
	}
	return true, 0, member, nil
}

func (s *AuthService) releaseEmailSendReservation(ctx context.Context, userID string, reservationID string) {
	if s.redis == nil || reservationID == "" {
		return
	}
	key := RedisKeyRegisterConfirmEmailWindowPrefix + userID
	_ = s.redis.ZRem(ctx, key, reservationID).Err()
}
