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
	sharedslug "mycourse-io-be/internal/shared/slug"
	"mycourse-io-be/internal/shared/timex"
	"mycourse-io-be/internal/shared/uuidx"
)

// PermissionReader returns the set of permission codes granted to a user (via RBAC roles + direct grants).
type PermissionReader interface {
	PermissionCodesForUser(userID string) (map[string]struct{}, error)
}

// RoleNameReader returns display-only RBAC role names for a user.
type RoleNameReader interface {
	RoleNamesForUser(ctx context.Context, userID string) ([]string, error)
}

// MeProjectionDeps groups the RBAC collaborators used to construct the cached /me projection.
type MeProjectionDeps struct {
	Permissions PermissionReader
	Roles       RoleNameReader
	Learner     LearnerRoleEnsurer
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
	roleNameReader     RoleNameReader
	learnerRoleEnsurer LearnerRoleEnsurer
	emailConfirmer     EmailConfirmer
	mediaValidator     MediaFileValidator
	orphanEnqueuer     OrphanCleanupEnqueuer
	redis              *redis.Client
	oauthIdentityRepo  domain.OAuthIdentityRepository
	oauthAccountWriter OAuthAccountWriter
	googleOAuth        GoogleOAuthClient
	xOAuth             XOAuthClient
	discordOAuth       DiscordOAuthClient
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
	meProjection MeProjectionDeps,
	emailConfirmer EmailConfirmer,
	media MediaFileValidator,
	orphan OrphanCleanupEnqueuer,
	rdb *redis.Client,
) *AuthService {
	return &AuthService{
		userRepo:           userRepo,
		sessionRepo:        sess,
		permReader:         meProjection.Permissions,
		roleNameReader:     meProjection.Roles,
		learnerRoleEnsurer: meProjection.Learner,
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
	if !user.HasLocalPassword() {
		return domain.TokenPairResult{}, domain.ErrInvalidCredentials
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
	now := time.Now()
	user.PasswordSetAt = &now
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
	me, err := s.loadMeProfile(ctx, user)
	if err != nil {
		return nil, err
	}
	s.setCachedMe(ctx, me)
	return me, nil
}

// --- UpdateMe ---

// UpdateMe applies PATCH /me fields: avatar_file_id and slug. A field that is
// not set (nil) is left untouched; when neither is set the current profile is
// returned. Everything that can be rejected without writing (reserved slug,
// avatar media validation) is checked before the first write; the two writes
// themselves are separate statements, so the cached /me is dropped as soon as
// any write has happened, even when a later write fails.
func (s *AuthService) UpdateMe(ctx context.Context, userID string, params domain.UpdateMeParams) (*domain.MeProfile, error) {
	if params.AvatarFileID == nil && params.Slug == nil {
		return s.GetMe(ctx, userID)
	}
	if params.Slug != nil && sharedslug.IsReserved(*params.Slug) {
		return nil, domain.ErrUserSlugReserved
	}
	user, err := s.loadAccessibleUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var avatar *avatarChange
	if params.AvatarFileID != nil {
		if avatar, err = s.prepareAvatarChange(user, *params.AvatarFileID); err != nil {
			return nil, err
		}
	}
	if params.Slug != nil && *params.Slug != user.Slug {
		_, err := s.userRepo.ChangeSlug(ctx, userID, *params.Slug)
		s.delCachedMe(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	if avatar != nil {
		err := s.commitAvatarChange(ctx, userID, avatar)
		s.delCachedMe(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	return s.GetMe(ctx, userID)
}

// avatarChange is a validated avatar update ready to be written.
type avatarChange struct {
	prev string
	next string
	ptr  *string // nil removes the avatar
}

// prepareAvatarChange validates a new avatar file id ("" removes the avatar)
// without writing anything.
func (s *AuthService) prepareAvatarChange(user *domain.User, avatarFileID string) (*avatarChange, error) {
	change := &avatarChange{next: strings.TrimSpace(avatarFileID)}
	if user.AvatarFileID != nil {
		change.prev = strings.TrimSpace(*user.AvatarFileID)
	}
	if change.next != "" {
		if s.mediaValidator != nil {
			if err := s.mediaValidator.ValidateProfileImageFile(change.next); err != nil {
				return nil, err
			}
		}
		change.ptr = &change.next
	}
	return change, nil
}

// commitAvatarChange stores the validated avatar and schedules cleanup of the
// previous file.
func (s *AuthService) commitAvatarChange(ctx context.Context, userID string, change *avatarChange) error {
	if err := s.userRepo.UpdateAvatar(ctx, userID, change.ptr); err != nil {
		return err
	}
	if change.prev != "" && change.prev != change.next && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanup(change.prev)
	}
	return nil
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

func (s *AuthService) roleNameSlice(ctx context.Context, userID string) ([]string, error) {
	if s.roleNameReader == nil {
		return []string{}, nil
	}
	names, err := s.roleNameReader.RoleNamesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return sortRolesByRank(names), nil
}

func (s *AuthService) loadMeProfile(ctx context.Context, user *domain.User) (*domain.MeProfile, error) {
	perms, err := s.permissionSlice(user.ID)
	if err != nil {
		return nil, err
	}
	perms, err = s.healConfirmedEmptyPermissions(user, perms)
	if err != nil {
		return nil, err
	}
	roles, err := s.roleNameSlice(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return buildMeProfile(user, perms, roles), nil
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
	me, err := s.loadMeProfile(ctx, user)
	if err != nil {
		return
	}
	s.setCachedMe(ctx, me)
}

// buildMeProfile maps a domain.User plus display projection fields to a MeProfile.
func buildMeProfile(user *domain.User, perms, roles []string) *domain.MeProfile {
	return &domain.MeProfile{
		UserID:         user.ID,
		UserCode:       user.UserCode,
		Email:          user.Email,
		DisplayName:    user.DisplayName,
		Slug:           user.Slug,
		AvatarFileID:   user.AvatarFileID,
		EmailConfirmed: user.EmailConfirmed,
		IsDisabled:     user.IsDisable,
		CreatedAt:      user.CreatedAt,
		Permissions:    perms,
		Roles:          nonNilStrings(roles),
	}
}

var roleRankOrder = []string{"sysadmin", "admin", "instructor", "learner"}

func sortRolesByRank(names []string) []string {
	out := nonNilStrings(names)
	sort.SliceStable(out, func(i, j int) bool {
		return roleRank(out[i]) < roleRank(out[j])
	})
	return out
}

func roleRank(name string) int {
	for rank, rankedName := range roleRankOrder {
		if name == rankedName {
			return rank
		}
	}
	return len(roleRankOrder)
}

func nonNilStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
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
