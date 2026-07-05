// Package application contains the AUTH bounded-context use-cases (login, register, confirm, refresh, me, delete).
package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"

	"mycourse-io-be/internal/auth/domain"
	"mycourse-io-be/internal/shared/brevo"
	sharedErrors "mycourse-io-be/internal/shared/errors"
	"mycourse-io-be/internal/shared/setting"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/uuidx"
)

// PermissionReader returns the set of permission codes granted to a user (via RBAC roles + direct grants).
type PermissionReader interface {
	PermissionCodesForUser(userID string) (map[string]struct{}, error)
}

// LearnerRoleEnsurer assigns the baseline learner role when absent.
type LearnerRoleEnsurer interface {
	EnsureLearnerRole(userID string) error
}

// EmailConfirmer atomically persists email-confirm fields and assigns the learner role.
type EmailConfirmer interface {
	ConfirmEmailWithLearnerRole(ctx context.Context, user *domain.User) error
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
	userRepo           domain.UserRepository
	sessionRepo        sessionRepo // extended infra methods
	permReader         PermissionReader
	learnerRoleEnsurer LearnerRoleEnsurer
	emailConfirmer     EmailConfirmer
	mediaValidator     MediaFileValidator
	orphanEnqueuer     OrphanCleanupEnqueuer
	redis              *redis.Client
}

// sessionRepo embeds the extended repository methods needed by application layer
// (AddSession, SaveSession beyond the domain interface).
type sessionRepo interface {
	domain.RefreshSessionRepository
	AddSession(ctx context.Context, userID string, sessionStr string, entry domain.RefreshSessionEntry) error
	SaveSession(ctx context.Context, userID string, sessionStr string, entry domain.RefreshSessionEntry) error
}

// NewAuthService constructs the service.  redis may be nil (cache is skipped when nil).
func NewAuthService(
	userRepo domain.UserRepository,
	sess sessionRepo,
	perm PermissionReader,
	learner LearnerRoleEnsurer,
	emailConfirmer EmailConfirmer,
	media MediaFileValidator,
	orphan OrphanCleanupEnqueuer,
	rdb *redis.Client,
) *AuthService {
	return &AuthService{
		userRepo:           userRepo,
		sessionRepo:        sess,
		permReader:         perm,
		learnerRoleEnsurer: learner,
		emailConfirmer:     emailConfirmer,
		mediaValidator:     media,
		orphanEnqueuer:     orphan,
		redis:              rdb,
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

	if err := checkUserAccessible(user, timex.NowUnix()); err != nil {
		if errors.Is(err, domain.ErrUserDisabled) || errors.Is(err, domain.ErrUserBanned) {
			return domain.TokenPairResult{}, err
		}
		return domain.TokenPairResult{}, domain.ErrInvalidCredentials
	}
	if !user.EmailConfirmed {
		return domain.TokenPairResult{}, domain.ErrEmailNotConfirmed
	}
	if !checkPasswordHash(password, user.HashPassword) {
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
func (s *AuthService) Register(ctx context.Context, email, password, displayName, locale string) error {
	if !isStrongPassword(password) {
		return domain.ErrWeakPassword
	}
	norm := normalizeEmail(email)

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		if existing.EmailConfirmed {
			return domain.ErrEmailAlreadyExists
		}
		return s.registerResendPending(ctx, norm, email, password, displayName, locale, existing)
	}
	if !isNotFound(err) {
		return err
	}
	return s.registerNewPending(ctx, norm, email, password, displayName, locale)
}

func (s *AuthService) registerNewPending(ctx context.Context, norm, email, password, displayName, locale string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	uid, err := uuidx.NewV7()
	if err != nil {
		return err
	}
	uc := uuidx.NewULID()
	tok := uuidx.NewV4()
	now := time.Now()
	user := &domain.User{
		ID:                 uid,
		UserCode:           uc,
		Email:              email,
		HashPassword:       hash,
		DisplayName:        displayName,
		ConfirmationToken:  &tok,
		ConfirmationSentAt: &now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}
	return s.sendRegistrationEmail(ctx, norm, email, displayName, locale, user)
}

func (s *AuthService) registerResendPending(ctx context.Context, norm, email, password, displayName, locale string, existing *domain.User) error {
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	tok := uuidx.NewV4()
	now := time.Now()
	existing.HashPassword = hash
	existing.DisplayName = displayName
	existing.ConfirmationToken = &tok
	existing.ConfirmationSentAt = &now
	if err := s.userRepo.Save(ctx, existing); err != nil {
		return err
	}
	return s.sendRegistrationEmail(ctx, norm, email, displayName, locale, existing)
}

func (s *AuthService) sendRegistrationEmail(ctx context.Context, norm, email, displayName, locale string, user *domain.User) error {
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
	baseURL := strings.TrimRight(setting.AppSetting.AppClientBaseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(setting.AppSetting.AppBaseURL, "/")
	}
	confirmURL := fmt.Sprintf("%s/%s/confirm-email?token=%s", baseURL, locale, *user.ConfirmationToken)
	if err := brevo.SendConfirmationEmail(email, displayName, confirmURL, locale); err != nil {
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
	if s.emailConfirmer == nil {
		return domain.TokenPairResult{}, errors.New("email confirmer not configured")
	}
	if err := s.emailConfirmer.ConfirmEmailWithLearnerRole(ctx, user); err != nil {
		return domain.TokenPairResult{}, err
	}
	norm := normalizeEmail(user.Email)
	s.delRegisterWindowKey(ctx, user.ID)
	s.delLoginUserByEmail(ctx, norm)
	s.delCachedMe(ctx, user.ID)

	return s.issueTokenPair(ctx, user, false, domain.RefreshTokenTTL)
}

// --- GetMe ---

// GetMe returns the cached or freshly loaded MeProfile for a user.
func (s *AuthService) GetMe(ctx context.Context, userID string) (*domain.MeProfile, error) {
	if me, ok := s.getCachedMe(ctx, userID); ok {
		if !needsLearnerRoleHeal(me) {
			return me, nil
		}
		s.delCachedMe(ctx, userID)
	}
	user, err := s.loadAccessibleUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	perms, err := s.permissionSlice(user.ID)
	if err != nil {
		return nil, err
	}
	perms, err = s.healConfirmedEmptyPermissions(user, perms)
	if err != nil {
		return nil, err
	}
	me := buildMeProfile(user, perms)
	s.setCachedMe(ctx, me)
	return me, nil
}

// --- UpdateMe ---

// UpdateMe applies PATCH /me fields (avatar_file_id).
func (s *AuthService) UpdateMe(ctx context.Context, userID string, avatarFileID *string) (*domain.MeProfile, error) {
	if avatarFileID == nil {
		return s.GetMe(ctx, userID)
	}
	user, err := s.loadAccessibleUser(ctx, userID)
	if err != nil {
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

// --- SoftDeleteUser ---

// SoftDeleteUser soft-deletes the user and schedules avatar cleanup.
func (s *AuthService) SoftDeleteUser(ctx context.Context, userID string) error {
	return s.deleteUserAccount(ctx, userID, s.userRepo.SoftDelete)
}

// HardDeleteUser permanently removes the user row and schedules avatar cleanup.
func (s *AuthService) HardDeleteUser(ctx context.Context, userID string) error {
	return s.deleteUserAccount(ctx, userID, s.userRepo.HardDelete)
}

func (s *AuthService) deleteUserAccount(ctx context.Context, userID string, deleteFn func(context.Context, string) error) error {
	user, err := s.loadAccessibleUser(ctx, userID)
	if err != nil {
		return err
	}
	var avatarFID string
	if user.AvatarFileID != nil {
		avatarFID = strings.TrimSpace(*user.AvatarFileID)
	}
	if err := deleteFn(ctx, userID); err != nil {
		return err
	}
	s.delCachedMe(ctx, userID)
	if avatarFID != "" && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanup(avatarFID)
	}
	return nil
}

func (s *AuthService) loadAccessibleUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	if err := checkUserAccessible(user, timex.NowUnix()); err != nil {
		return nil, err
	}
	return user, nil
}

// --- internal helpers ---

func (s *AuthService) permissionSlice(userID string) ([]string, error) {
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

func needsLearnerRoleHeal(me *domain.MeProfile) bool {
	return me != nil && me.EmailConfirmed && len(me.Permissions) == 0
}

func (s *AuthService) ensureLearnerRole(userID string) error {
	if s.learnerRoleEnsurer == nil {
		return nil
	}
	return s.learnerRoleEnsurer.EnsureLearnerRole(userID)
}

func (s *AuthService) healConfirmedEmptyPermissions(user *domain.User, perms []string) ([]string, error) {
	if user == nil || !user.EmailConfirmed || len(perms) > 0 {
		return perms, nil
	}
	if err := s.ensureLearnerRole(user.ID); err != nil {
		return nil, err
	}
	return s.permissionSlice(user.ID)
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
		CreatedAt:      user.CreatedAt,
		Permissions:    perms,
	}
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
