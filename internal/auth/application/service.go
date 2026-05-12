// Package application contains the AUTH bounded-context use-cases (login, register, confirm, refresh, me, delete).
package application

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/brevo"
	sharedErrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/token"
)

// PermissionReader returns the set of permission codes granted to a user (via RBAC roles + direct grants).
type PermissionReader interface {
	PermissionCodesForUser(userID uint) (map[string]struct{}, error)
}

// MediaFileValidator checks if a file ID references a valid, READY non-video raster image.
type MediaFileValidator interface {
	ValidateProfileImageFile(fileID string) error
}

// OrphanCleanupEnqueuer schedules deferred cloud storage cleanup for a media file.
type OrphanCleanupEnqueuer interface {
	EnqueueOrphanCleanup(fileID string)
}

// AuthService implements the auth domain use-cases with injected dependencies.
type AuthService struct {
	userRepo       domain.UserRepository
	sessionRepo    sessionRepo // extended infra methods
	permReader     PermissionReader
	mediaValidator MediaFileValidator
	orphanEnqueuer OrphanCleanupEnqueuer
	redis          *redis.Client
}

// sessionRepo embeds the extended repository methods needed by application layer
// (AddSession, SaveSession beyond the domain interface).
type sessionRepo interface {
	domain.RefreshSessionRepository
	AddSession(ctx context.Context, userID uint, sessionStr string, entry domain.RefreshSessionEntry) error
	SaveSession(ctx context.Context, userID uint, sessionStr string, entry domain.RefreshSessionEntry) error
}

// NewAuthService constructs the service.  redis may be nil (cache is skipped when nil).
func NewAuthService(
	userRepo domain.UserRepository,
	sess sessionRepo,
	perm PermissionReader,
	media MediaFileValidator,
	orphan OrphanCleanupEnqueuer,
	rdb *redis.Client,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		sessionRepo:    sess,
		permReader:     perm,
		mediaValidator: media,
		orphanEnqueuer: orphan,
		redis:          rdb,
	}
}

// --- Login ---

// Login validates credentials and returns a token pair.
func (s *AuthService) Login(ctx context.Context, email, password string, rememberMe bool) (domain.TokenPairResult, error) {
	normEmail := normalizeEmail(email)
	if s.loginInvalidCached(ctx, normEmail) {
		return domain.TokenPairResult{}, domain.ErrInvalidCredentials
	}

	user, err := s.loadUserForLogin(ctx, email, normEmail)
	if err != nil {
		if isNotFound(err) {
			s.setLoginInvalidCache(ctx, normEmail)
			return domain.TokenPairResult{}, domain.ErrInvalidCredentials
		}
		return domain.TokenPairResult{}, err
	}

	if user.IsDisable {
		return domain.TokenPairResult{}, domain.ErrUserDisabled
	}
	if !user.EmailConfirmed {
		return domain.TokenPairResult{}, domain.ErrEmailNotConfirmed
	}
	if !checkPasswordHash(password, user.HashPassword) {
		s.setLoginInvalidCache(ctx, normEmail)
		return domain.TokenPairResult{}, domain.ErrInvalidCredentials
	}

	refreshTTL := domain.RefreshTokenTTL
	if rememberMe {
		refreshTTL = domain.RememberMeRefreshTTL
	}
	result, err := s.issueTokenPair(ctx, user, rememberMe, refreshTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	s.delLoginInvalidCache(ctx, normEmail)
	s.warmMeCache(ctx, user)
	return result, nil
}

// --- Register ---

// Register creates or updates a pending user and sends a confirmation email.
func (s *AuthService) Register(ctx context.Context, email, password, displayName string) error {
	if !isStrongPassword(password) {
		return domain.ErrWeakPassword
	}
	norm := normalizeEmail(email)

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		if existing.EmailConfirmed {
			return domain.ErrEmailAlreadyExists
		}
		return s.registerResendPending(ctx, norm, email, password, displayName, existing)
	}
	if !isNotFound(err) {
		return err
	}
	return s.registerNewPending(ctx, norm, email, password, displayName)
}

func (s *AuthService) registerNewPending(ctx context.Context, norm, email, password, displayName string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	uc, err := uuid.NewV7()
	if err != nil {
		return err
	}
	tok := uuid.New().String()
	now := time.Now()
	user := &domain.User{
		UserCode:          uc.String(),
		Email:             email,
		HashPassword:      hash,
		DisplayName:       displayName,
		ConfirmationToken: &tok,
		ConfirmationSentAt: &now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}
	return s.sendRegistrationEmail(ctx, norm, email, displayName, user)
}

func (s *AuthService) registerResendPending(ctx context.Context, norm, email, password, displayName string, existing *domain.User) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	tok := uuid.New().String()
	now := time.Now()
	existing.HashPassword = hash
	existing.DisplayName = displayName
	existing.ConfirmationToken = &tok
	existing.ConfirmationSentAt = &now
	if err := s.userRepo.Save(ctx, existing); err != nil {
		return err
	}
	return s.sendRegistrationEmail(ctx, norm, email, displayName, existing)
}

func (s *AuthService) sendRegistrationEmail(ctx context.Context, norm, email, displayName string, user *domain.User) error {
	// Lifetime cap
	if user.RegistrationEmailSendTotal >= MaxRegisterConfirmationEmailsLifetime {
		if err := s.userRepo.SoftDelete(ctx, user.ID); err != nil {
			return err
		}
		s.delRegisterWindowKey(ctx, user.ID)
		s.delLoginUserByEmail(ctx, norm)
		return domain.ErrRegistrationAbandoned
	}

	// Sliding window (Redis)
	allowed, retryAfter, reservationID, err := s.tryReserveEmailSend(ctx, user.ID)
	if err != nil {
		return err
	}
	if !allowed {
		sec := int64(retryAfter / time.Second)
		if sec < 1 {
			sec = 1
		}
		return &domain.RegistrationEmailRateLimitedError{RetryAfterSeconds: sec}
	}

	if user.ConfirmationToken == nil {
		s.releaseEmailSendReservation(ctx, user.ID, reservationID)
		return domain.ErrConfirmationEmailSendFailed
	}
	confirmURL := setting.AppSetting.AppBaseURL + "/api/v1/auth/confirm?token=" + *user.ConfirmationToken
	if err := brevo.SendConfirmationEmail(email, displayName, confirmURL); err != nil {
		s.releaseEmailSendReservation(ctx, user.ID, reservationID)
		return domain.ErrConfirmationEmailSendFailed
	}
	_, _ = s.sessionRepo.IncrementEmailSendCount(ctx, user.ID)
	return nil
}

// --- ConfirmEmail ---

// ConfirmEmail marks the user's email as confirmed and returns a token pair.
func (s *AuthService) ConfirmEmail(ctx context.Context, confirmToken string) (domain.TokenPairResult, error) {
	user, err := s.userRepo.FindByConfirmationToken(ctx, confirmToken)
	if err != nil {
		if isNotFound(err) {
			return domain.TokenPairResult{}, domain.ErrInvalidConfirmToken
		}
		return domain.TokenPairResult{}, err
	}
	// Update user fields
	user.EmailConfirmed = true
	user.ConfirmationToken = nil
	user.RegistrationEmailSendTotal = 0
	if err := s.userRepo.Save(ctx, user); err != nil {
		return domain.TokenPairResult{}, err
	}
	// Clear caches
	norm := normalizeEmail(user.Email)
	s.delRegisterWindowKey(ctx, user.ID)
	s.delLoginUserByEmail(ctx, norm)

	return s.issueTokenPair(ctx, user, false, domain.RefreshTokenTTL)
}

// --- GetMe ---

// GetMe returns the cached or freshly loaded MeProfile for a user.
func (s *AuthService) GetMe(ctx context.Context, userID uint) (*domain.MeProfile, error) {
	if me, ok := s.getCachedMe(ctx, userID); ok {
		return me, nil
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	perms, err := s.permissionSlice(user.ID)
	if err != nil {
		return nil, err
	}
	me := buildMeProfile(user, perms)
	s.setCachedMe(ctx, me)
	return me, nil
}

// --- UpdateMe ---

// UpdateMe applies PATCH /me fields (avatar_file_id).
func (s *AuthService) UpdateMe(ctx context.Context, userID uint, avatarFileID *string) (*domain.MeProfile, error) {
	if avatarFileID == nil {
		return s.GetMe(ctx, userID)
	}
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	var prev string
	if user.AvatarFileID != nil {
		prev = strings.TrimSpace(*user.AvatarFileID)
	}
	next := strings.TrimSpace(*avatarFileID)
	var nextPtr *string
	if next != "" {
		if s.mediaValidator != nil {
			if err := s.mediaValidator.ValidateProfileImageFile(next); err != nil {
				return nil, err
			}
		}
		nextPtr = &next
	}
	if err := s.userRepo.UpdateAvatar(ctx, userID, nextPtr); err != nil {
		return nil, err
	}
	if prev != "" && prev != next && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanup(prev)
	}
	s.delCachedMe(ctx, userID)
	return s.GetMe(ctx, userID)
}

// --- RefreshSession ---

// RefreshSession rotates the token pair for an existing session.
func (s *AuthService) RefreshSession(ctx context.Context, sessionStr, refreshTokenStr string) (domain.TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret
	refreshClaims, err := token.ParseRefreshIgnoreExpiry(secret, refreshTokenStr)
	if err != nil {
		return domain.TokenPairResult{}, domain.ErrInvalidSession
	}
	user, err := s.userRepo.FindByID(ctx, refreshClaims.UserID)
	if err != nil {
		if isNotFound(err) {
			return domain.TokenPairResult{}, domain.ErrUserNotFound
		}
		return domain.TokenPairResult{}, err
	}
	if user.IsDisable {
		return domain.TokenPairResult{}, domain.ErrUserDisabled
	}
	entry, ok := user.RefreshTokenSession[sessionStr]
	if !ok || entry.RefreshTokenUUID != refreshClaims.UUID {
		return domain.TokenPairResult{}, domain.ErrInvalidSession
	}
	if time.Now().After(entry.RefreshTokenExpired) {
		return domain.TokenPairResult{}, domain.ErrRefreshTokenExpired
	}
	var newTTL time.Duration
	if entry.RememberMe {
		newTTL = domain.RememberMeRefreshTTL
	} else {
		newTTL = time.Until(entry.RefreshTokenExpired)
		if newTTL <= 0 {
			return domain.TokenPairResult{}, domain.ErrRefreshTokenExpired
		}
	}
	return s.rotateSession(ctx, user, sessionStr, entry, newTTL)
}

func (s *AuthService) rotateSession(ctx context.Context, user *domain.User, sessionStr string, entry domain.RefreshSessionEntry, newTTL time.Duration) (domain.TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret
	newUUID := uuid.New().String()
	perms, err := s.permissionSlice(user.ID)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	at, err := token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, domain.AccessTokenTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	rt, err := token.GenerateRefresh(secret, user.ID, newUUID, newTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	updatedEntry := domain.RefreshSessionEntry{
		RefreshTokenUUID:    newUUID,
		RememberMe:          entry.RememberMe,
		RefreshTokenExpired: time.Now().Add(newTTL),
	}
	if err := s.sessionRepo.SaveSession(ctx, user.ID, sessionStr, updatedEntry); err != nil {
		return domain.TokenPairResult{}, err
	}
	return domain.TokenPairResult{
		AccessToken:  at,
		RefreshToken: rt,
		SessionStr:   sessionStr,
		RefreshTTL:   newTTL,
	}, nil
}

// --- SoftDeleteUser ---

// SoftDeleteUser soft-deletes the user and schedules avatar cleanup.
func (s *AuthService) SoftDeleteUser(ctx context.Context, userID uint) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return domain.ErrUserNotFound
		}
		return err
	}
	var avatarFID string
	if user.AvatarFileID != nil {
		avatarFID = strings.TrimSpace(*user.AvatarFileID)
	}
	if err := s.userRepo.SoftDelete(ctx, userID); err != nil {
		return err
	}
	s.delCachedMe(ctx, userID)
	if avatarFID != "" && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanup(avatarFID)
	}
	return nil
}

// --- internal helpers ---

func (s *AuthService) issueTokenPair(ctx context.Context, user *domain.User, rememberMe bool, refreshTTL time.Duration) (domain.TokenPairResult, error) {
	secret := setting.AppSetting.JWTSecret
	sessionStr, err := token.GenerateSessionString(secret)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	sessionUUID := uuid.New().String()
	perms, err := s.permissionSlice(user.ID)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	at, err := token.GenerateAccess(secret, user.ID, user.UserCode, user.Email, user.DisplayName, user.CreatedAt, perms, domain.AccessTokenTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	rt, err := token.GenerateRefresh(secret, user.ID, sessionUUID, refreshTTL)
	if err != nil {
		return domain.TokenPairResult{}, err
	}
	entry := domain.RefreshSessionEntry{
		RefreshTokenUUID:    sessionUUID,
		RememberMe:          rememberMe,
		RefreshTokenExpired: time.Now().Add(refreshTTL),
	}
	if err := s.sessionRepo.AddSession(ctx, user.ID, sessionStr, entry); err != nil {
		return domain.TokenPairResult{}, err
	}
	return domain.TokenPairResult{AccessToken: at, RefreshToken: rt, SessionStr: sessionStr, RefreshTTL: refreshTTL}, nil
}

func (s *AuthService) permissionSlice(userID uint) ([]string, error) {
	if s.permReader == nil {
		return nil, nil
	}
	set, err := s.permReader.PermissionCodesForUser(userID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(set))
	for action := range set {
		out = append(out, action)
	}
	sort.Strings(out)
	return out, nil
}

func (s *AuthService) loadUserForLogin(ctx context.Context, email, normEmail string) (*domain.User, error) {
	if uid, ok := s.getCachedLoginUserID(ctx, normEmail); ok {
		u, err := s.userRepo.FindByID(ctx, uid)
		if err == nil && normalizeEmail(u.Email) == normEmail {
			return u, nil
		}
	}
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	s.setCachedLoginUserID(ctx, normEmail, user.ID)
	return user, nil
}

func (s *AuthService) warmMeCache(ctx context.Context, user *domain.User) {
	if s.redis == nil {
		return
	}
	perms, err := s.permissionSlice(user.ID)
	if err != nil {
		return
	}
	me := buildMeProfile(user, perms)
	s.setCachedMe(ctx, me)
}

// buildMeProfile maps a domain.User + permissions to a MeProfile.
func buildMeProfile(user *domain.User, perms []string) *domain.MeProfile {
	return &domain.MeProfile{
		UserID:         user.ID,
		UserCode:       user.UserCode,
		Email:          user.Email,
		DisplayName:    user.DisplayName,
		AvatarFileID:   user.AvatarFileID,
		EmailConfirmed: user.EmailConfirmed,
		IsDisabled:     user.IsDisable,
		CreatedAt:      user.CreatedAt.Unix(),
		Permissions:    perms,
	}
}

// --- Redis cache helpers ---

func (s *AuthService) redisKey(prefix string, id interface{}) string {
	switch v := id.(type) {
	case string:
		return prefix + v
	case uint:
		return prefix + strconv.FormatUint(uint64(v), 10)
	}
	return prefix
}

func (s *AuthService) getCachedMe(ctx context.Context, userID uint) (*domain.MeProfile, bool) {
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

func (s *AuthService) delCachedMe(ctx context.Context, userID uint) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, s.redisKey(RedisKeyUserMePrefix, userID)).Err()
	}
}

func (s *AuthService) getCachedLoginUserID(ctx context.Context, normEmail string) (uint, bool) {
	if s.redis == nil || normEmail == "" {
		return 0, false
	}
	v, err := s.redis.Get(ctx, s.redisKey(RedisKeyLoginUserByEmailPrefix, normEmail)).Result()
	if err != nil {
		return 0, false
	}
	id, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}

func (s *AuthService) setCachedLoginUserID(ctx context.Context, normEmail string, userID uint) {
	if s.redis == nil || normEmail == "" {
		return
	}
	_ = s.redis.Set(ctx, s.redisKey(RedisKeyLoginUserByEmailPrefix, normEmail),
		strconv.FormatUint(uint64(userID), 10), LoginEmailUserIDTTL).Err()
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

func (s *AuthService) delRegisterWindowKey(ctx context.Context, userID uint) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, s.redisKey(RedisKeyRegisterConfirmEmailWindowPrefix, userID)).Err()
	}
}

func (s *AuthService) tryReserveEmailSend(ctx context.Context, userID uint) (bool, time.Duration, string, error) {
	if s.redis == nil {
		return true, 0, "", nil
	}
	member := uuid.New().String()
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
	key := RedisKeyRegisterConfirmEmailWindowPrefix + strconv.FormatUint(uint64(userID), 10)
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

func (s *AuthService) releaseEmailSendReservation(ctx context.Context, userID uint, reservationID string) {
	if s.redis == nil || reservationID == "" {
		return
	}
	key := RedisKeyRegisterConfirmEmailWindowPrefix + strconv.FormatUint(uint64(userID), 10)
	_ = s.redis.ZRem(ctx, key, reservationID).Err()
}

// --- util funcs ---

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isNotFound(err error) bool {
	return err == sharedErrors.ErrNotFound
}

func isStrongPassword(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	var hasUpper, hasLower, hasSpecial bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasSpecial
}

func hashPassword(password string) (string, error) {
	const cost = 12
	b, err := bcryptGenerateFromPassword([]byte(password), cost)
	return string(b), err
}

func checkPasswordHash(password, hash string) bool {
	return bcryptCompare([]byte(hash), []byte(password)) == nil
}

func toInt64(v interface{}) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	}
	return 0, false
}
