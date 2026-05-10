package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"mycourse-io-be/constants"
	"mycourse-io-be/pkg/cache_clients"
)

// Sliding window for successful registration confirmation emails (scores = unix milliseconds).
var registerConfirmWindowLua = redis.NewScript(`
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

func registerConfirmWindowKey(userID uint) string {
	return constants.RedisKeyRegisterConfirmEmailWindowPrefix + strconv.FormatUint(uint64(userID), 10)
}

func luaInt64(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case uint64:
		return int64(x), true
	default:
		return 0, false
	}
}

// TryReserveRegisterConfirmationSend reserves one slot in the Redis sliding window before sending email.
// When Redis is unavailable it returns allowed=true and empty reservationID (window not enforced).
// On success reservationID must be passed to ReleaseRegisterConfirmationSend if the email send fails.
func TryReserveRegisterConfirmationSend(ctx context.Context, userID uint) (allowed bool, retryAfter time.Duration, reservationID string, err error) {
	if !cache_clients.RedisAvailable() {
		return true, 0, "", nil
	}
	member := uuid.New().String()
	now := time.Now().UnixMilli()
	win := constants.RegisterConfirmationEmailWindow.Milliseconds()
	max := int64(constants.MaxRegisterConfirmationEmailsPerWindow)
	r, err := registerConfirmWindowLua.Run(ctx, cache_clients.Redis,
		[]string{registerConfirmWindowKey(userID)},
		now, win, max, member,
	).Slice()
	if err != nil {
		return false, 0, "", err
	}
	if len(r) < 2 {
		return false, 0, "", nil
	}
	flag, _ := luaInt64(r[0])
	retryMs, _ := luaInt64(r[1])
	if flag == 0 {
		return false, time.Duration(retryMs) * time.Millisecond, "", nil
	}
	return true, 0, member, nil
}

// ReleaseRegisterConfirmationSend removes a reserved window slot after a failed email send.
func ReleaseRegisterConfirmationSend(ctx context.Context, userID uint, reservationID string) {
	if reservationID == "" || !cache_clients.RedisAvailable() {
		return
	}
	_ = cache_clients.Redis.ZRem(ctx, registerConfirmWindowKey(userID), reservationID).Err()
}

// DeleteRegisterConfirmationEmailWindow removes the entire window key (e.g. after successful email confirm).
func DeleteRegisterConfirmationEmailWindow(ctx context.Context, userID uint) {
	if !cache_clients.RedisAvailable() {
		return
	}
	_ = cache_clients.Redis.Del(ctx, registerConfirmWindowKey(userID)).Err()
}
