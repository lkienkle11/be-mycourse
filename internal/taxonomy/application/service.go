// Package application contains the TAXONOMY bounded-context use-case layer.
package application

import (
	"context"
	"errors"
	"strings"

	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/taxonomy/domain"
)

// MediaFileValidator validates and loads a profile-image file by ID.
// Implemented by internal/media/application.MediaService.
type MediaFileValidator interface {
	LoadValidatedProfileImageFile(ctx context.Context, fileID string) (imageURL string, err error)
}

// OrphanImageEnqueuer schedules deferred cloud cleanup for a media file ID.
type OrphanImageEnqueuer interface {
	EnqueueOrphanCleanupForFileID(ctx context.Context, fileID string)
}

// TaxonomyService provides all taxonomy use-cases.
type TaxonomyService struct {
	topicRepo       domain.CourseTopicRepository
	outcomeRepo     domain.CourseOutcomeRepository
	skillRepo       domain.CourseSkillRepository
	tagRepo         domain.TagRepository
	courseLevelRepo domain.CourseLevelRepository
	mediaValidator  MediaFileValidator
	orphanEnqueuer  OrphanImageEnqueuer
}

// NewTaxonomyService constructs a TaxonomyService.
func NewTaxonomyService(
	topicRepo domain.CourseTopicRepository,
	outcomeRepo domain.CourseOutcomeRepository,
	skillRepo domain.CourseSkillRepository,
	tagRepo domain.TagRepository,
	courseLevelRepo domain.CourseLevelRepository,
	mediaValidator MediaFileValidator,
	orphanEnqueuer OrphanImageEnqueuer,
) *TaxonomyService {
	return &TaxonomyService{
		topicRepo:       topicRepo,
		outcomeRepo:     outcomeRepo,
		skillRepo:       skillRepo,
		tagRepo:         tagRepo,
		courseLevelRepo: courseLevelRepo,
		mediaValidator:  mediaValidator,
		orphanEnqueuer:  orphanEnqueuer,
	}
}

// --- CourseTopic -------------------------------------------------------------

func (s *TaxonomyService) ListTopics(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseTopic, int64, error) {
	return s.topicRepo.List(ctx, filter)
}

func (s *TaxonomyService) ListTopicsFull(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseTopic, int64, error) {
	filter.IncludeDeleted = true
	return s.topicRepo.List(ctx, filter)
}

func (s *TaxonomyService) GetTopic(ctx context.Context, id, locale string, viewEdit bool) (*domain.CourseTopic, error) {
	return s.topicRepo.GetDetail(ctx, id, locale, viewEdit)
}

func (s *TaxonomyService) CreateTopic(ctx context.Context, in domain.CreateCourseTopicInput) (*domain.CourseTopic, error) {
	name, tr, err := syncNameCanonicalAndTranslations(in.Name, in.Translations)
	if err != nil {
		return nil, err
	}
	manualSlug, err := validateManualTaxonomySlug(in.Slug)
	if err != nil {
		return nil, err
	}
	childTopics, err := resolveTreeForCreate(in.ChildTopics)
	if err != nil {
		return nil, err
	}
	fileID := strings.TrimSpace(in.ImageFileID)
	if fileID != "" && s.mediaValidator != nil {
		if _, err := s.mediaValidator.LoadValidatedProfileImageFile(ctx, fileID); err != nil {
			return nil, err
		}
	}
	_, st := trimmedNameAndStatus(name, in.Status)
	t := &domain.CourseTopic{
		Name: name, Slug: manualSlug, Status: st, ChildTopics: childTopics,
		Translations: tr, RowVersion: 1,
		CreatedBy: stringPtrIfNotBlank(in.ActorID),
	}
	if fileID != "" {
		t.ImageFileID = &fileID
	}
	if err := s.topicRepo.Create(ctx, t); err != nil {
		return nil, err // may be *domain.SlugConflictError — propagated as-is, mapped at delivery
	}
	return s.topicRepo.GetDetail(ctx, t.ID, "", true)
}

func (s *TaxonomyService) UpdateTopic(ctx context.Context, id string, in domain.UpdateCourseTopicInput) (*domain.CourseTopic, error) {
	row, err := s.topicRepo.GetDetail(ctx, id, "", true)
	if err != nil {
		return nil, err
	}
	if err := applyCourseTopicUpdate(row, in); err != nil {
		return nil, err
	}
	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.mutateImageFileID(ctx, &row.ImageFileID, in.ImageFileID); err != nil {
		return nil, err
	}
	if err := s.topicRepo.Save(ctx, row, in.ExpectedRowVersion); err != nil {
		return nil, err
	}
	enqueueReplacedImageCleanup(ctx, s.orphanEnqueuer, in.ImageFileID, prevFileID, imageFileIDStr(row.ImageFileID))
	return s.topicRepo.GetDetail(ctx, id, "", true)
}

func applyCourseTopicUpdate(row *domain.CourseTopic, in domain.UpdateCourseTopicInput) error {
	canonical := row.Name
	if in.Name != nil {
		canonical = *in.Name
	}
	merged := row.Translations
	if in.Translations != nil {
		patch, err := canonicalizeNameTranslations(in.Translations)
		if err != nil {
			return err
		}
		merged = replaceNameTranslations(patch)
	}
	name, tr, err := syncNameCanonicalAndTranslations(canonical, merged)
	if err != nil {
		return err
	}
	slugInput, err := resolveTaxonomySlugInput(in.Slug)
	if err != nil {
		return err
	}
	row.Name = name
	// Slug is independent of name as of the taxonomy slug unification: only
	// touched when slugInput is non-nil (an explicit new value); omitted
	// leaves row.Slug exactly as GetDetail loaded it.
	if slugInput != nil {
		row.Slug = *slugInput
	}
	row.Translations = tr
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "" {
			row.Status = v
		}
	}
	return applyCourseTopicChildren(row, in.ChildTopics)
}

func applyCourseTopicChildren(row *domain.CourseTopic, childTopics *[]taxpkg.TreeNode) error {
	resolved, err := resolveTreeFieldForUpdate(row.ChildTopics, childTopics)
	if err != nil {
		return err
	}
	row.ChildTopics = resolved
	return nil
}

// resolveTreeFieldForUpdate resolves a tree field's node slugs for update,
// shared by applyCourseTopicChildren (CourseTopic.ChildTopics) and
// UpdateCourseSkill (CourseSkill.Children) — both entities have an
// independent tree field with identical "omitted → preserve from current,
// provided → use incoming" semantics, differing only in which struct field
// holds the tree, which the caller already resolves before calling in.
func resolveTreeFieldForUpdate(current []taxpkg.TreeNode, incoming *[]taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	oldByID := taxpkg.BuildNodeIndexByID(current)
	source := current
	if incoming != nil {
		source = *incoming
	}
	return resolveTreeForUpdate(source, oldByID)
}

func (s *TaxonomyService) DeleteTopic(ctx context.Context, id string) error {
	return s.topicRepo.SoftDelete(ctx, id)
}

func (s *TaxonomyService) HardDeleteTopic(ctx context.Context, id string) error {
	return s.deleteWithOrphanImage(ctx, id, imageIDLoaderTopic(s), s.topicRepo.HardDelete)
}

// --- CourseOutcome -----------------------------------------------------------

func (s *TaxonomyService) ListCourseOutcomes(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseOutcome, int64, error) {
	return s.outcomeRepo.List(ctx, filter)
}

func (s *TaxonomyService) ListCourseOutcomesFull(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseOutcome, int64, error) {
	filter.IncludeDeleted = true
	return s.outcomeRepo.List(ctx, filter)
}

func (s *TaxonomyService) GetCourseOutcome(ctx context.Context, id, locale string, viewEdit bool) (*domain.CourseOutcome, error) {
	return s.outcomeRepo.GetDetail(ctx, id, locale, viewEdit)
}

func (s *TaxonomyService) CreateCourseOutcome(ctx context.Context, in domain.CreateCourseOutcomeInput) (*domain.CourseOutcome, error) {
	short, desc, tr, err := syncOutcomeCanonicalAndTranslations(in.ShortDescription, in.Description, in.Translations)
	if err != nil {
		return nil, err
	}
	if err := validateOutcomePayload(short, desc); err != nil {
		return nil, err
	}
	fileID := strings.TrimSpace(in.ImageFileID)
	if fileID != "" && s.mediaValidator != nil {
		if _, err := s.mediaValidator.LoadValidatedProfileImageFile(ctx, fileID); err != nil {
			return nil, err
		}
	}
	st := strings.ToUpper(strings.TrimSpace(in.Status))
	if st == "" {
		st = "ACTIVE"
	}
	o := &domain.CourseOutcome{
		ShortDescription: short, Description: desc, Status: st,
		Translations: tr, RowVersion: 1,
		CreatedBy: stringPtrIfNotBlank(in.ActorID),
	}
	if fileID != "" {
		o.ImageFileID = &fileID
	}
	if err := s.outcomeRepo.Create(ctx, o); err != nil {
		return nil, err
	}
	return s.outcomeRepo.GetDetail(ctx, o.ID, "", true)
}

func (s *TaxonomyService) UpdateCourseOutcome(ctx context.Context, id string, in domain.UpdateCourseOutcomeInput) (*domain.CourseOutcome, error) {
	row, err := s.outcomeRepo.GetDetail(ctx, id, "", true)
	if err != nil {
		return nil, err
	}
	if err := applyCourseOutcomeUpdate(row, in); err != nil {
		return nil, err
	}
	return s.persistOutcomeUpdate(ctx, id, row, in)
}

func (s *TaxonomyService) persistOutcomeUpdate(
	ctx context.Context,
	id string,
	row *domain.CourseOutcome,
	in domain.UpdateCourseOutcomeInput,
) (*domain.CourseOutcome, error) {
	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.mutateImageFileID(ctx, &row.ImageFileID, in.ImageFileID); err != nil {
		return nil, err
	}
	if err := s.outcomeRepo.Save(ctx, row, in.ExpectedRowVersion); err != nil {
		return nil, err
	}
	enqueueReplacedImageCleanup(ctx, s.orphanEnqueuer, in.ImageFileID, prevFileID, imageFileIDStr(row.ImageFileID))
	return s.outcomeRepo.GetDetail(ctx, id, "", true)
}

func applyCourseOutcomeUpdate(row *domain.CourseOutcome, in domain.UpdateCourseOutcomeInput) error {
	short := row.ShortDescription
	if in.ShortDescription != nil {
		short = *in.ShortDescription
	}
	desc := row.Description
	if in.Description != nil {
		desc = *in.Description
	}
	merged := row.Translations
	if in.Translations != nil {
		patch, err := canonicalizeOutcomeTranslations(in.Translations)
		if err != nil {
			return err
		}
		merged = replaceOutcomeTranslations(patch)
	}
	short, desc, tr, err := syncOutcomeCanonicalAndTranslations(short, desc, merged)
	if err != nil {
		return err
	}
	if err := validateOutcomePayload(short, desc); err != nil {
		return err
	}
	row.ShortDescription = short
	row.Description = desc
	row.Translations = tr
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "" {
			row.Status = v
		}
	}
	return nil
}

func enqueueReplacedImageCleanup(
	ctx context.Context,
	enqueuer OrphanImageEnqueuer,
	imageFileID *string,
	prevFileID, nextFileID string,
) {
	if imageFileID == nil || prevFileID == "" || prevFileID == nextFileID || enqueuer == nil {
		return
	}
	enqueuer.EnqueueOrphanCleanupForFileID(ctx, prevFileID)
}

func (s *TaxonomyService) DeleteCourseOutcome(ctx context.Context, id string) error {
	return s.outcomeRepo.SoftDelete(ctx, id)
}

func (s *TaxonomyService) HardDeleteCourseOutcome(ctx context.Context, id string) error {
	return s.deleteWithOrphanImage(ctx, id, imageIDLoaderOutcome(s), s.outcomeRepo.HardDelete)
}

// --- CourseSkill -------------------------------------------------------------

func (s *TaxonomyService) ListCourseSkills(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseSkill, int64, error) {
	return s.skillRepo.List(ctx, filter)
}

func (s *TaxonomyService) ListCourseSkillsFull(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseSkill, int64, error) {
	filter.IncludeDeleted = true
	return s.skillRepo.List(ctx, filter)
}

func (s *TaxonomyService) GetCourseSkill(ctx context.Context, id, locale string, viewEdit bool) (*domain.CourseSkill, error) {
	return s.skillRepo.GetDetail(ctx, id, locale, viewEdit)
}

func (s *TaxonomyService) CreateCourseSkill(ctx context.Context, in domain.CreateCourseSkillInput) (*domain.CourseSkill, error) {
	name, tr, err := syncNameCanonicalAndTranslations(in.Name, in.Translations)
	if err != nil {
		return nil, err
	}
	manualSlug, err := validateManualTaxonomySlug(in.Slug)
	if err != nil {
		return nil, err
	}
	children, err := resolveTreeForCreate(in.Children)
	if err != nil {
		return nil, err
	}
	_, st := trimmedNameAndStatus(name, in.Status)
	sk := &domain.CourseSkill{
		Name: name, Slug: manualSlug, Status: st, Children: children,
		Translations: tr, RowVersion: 1,
		CreatedBy: stringPtrIfNotBlank(in.ActorID),
	}
	if err := s.skillRepo.Create(ctx, sk); err != nil {
		return nil, err
	}
	return s.skillRepo.GetDetail(ctx, sk.ID, "", true)
}

func (s *TaxonomyService) UpdateCourseSkill(ctx context.Context, id string, in domain.UpdateCourseSkillInput) (*domain.CourseSkill, error) {
	row, err := s.skillRepo.GetDetail(ctx, id, "", true)
	if err != nil {
		return nil, err
	}
	canonical := row.Name
	if in.Name != nil {
		canonical = *in.Name
	}
	merged := row.Translations
	if in.Translations != nil {
		patch, err := canonicalizeNameTranslations(in.Translations)
		if err != nil {
			return nil, err
		}
		merged = replaceNameTranslations(patch)
	}
	name, tr, err := syncNameCanonicalAndTranslations(canonical, merged)
	if err != nil {
		return nil, err
	}
	slugInput, err := resolveTaxonomySlugInput(in.Slug)
	if err != nil {
		return nil, err
	}
	row.Name = name
	if slugInput != nil {
		row.Slug = *slugInput
	}
	row.Translations = tr
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "" {
			row.Status = v
		}
	}
	resolved, err := resolveTreeFieldForUpdate(row.Children, in.Children)
	if err != nil {
		return nil, err
	}
	row.Children = resolved
	if err := s.skillRepo.Save(ctx, row, in.ExpectedRowVersion); err != nil {
		return nil, err
	}
	return s.skillRepo.GetDetail(ctx, id, "", true)
}

func (s *TaxonomyService) DeleteCourseSkill(ctx context.Context, id string) error {
	return s.skillRepo.SoftDelete(ctx, id)
}

func (s *TaxonomyService) HardDeleteCourseSkill(ctx context.Context, id string) error {
	return s.skillRepo.HardDelete(ctx, id)
}

// --- Tag ---------------------------------------------------------------------

func (s *TaxonomyService) ListTags(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.Tag, int64, error) {
	return s.tagRepo.List(ctx, filter)
}

func (s *TaxonomyService) ListTagsFull(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.Tag, int64, error) {
	filter.IncludeDeleted = true
	return s.tagRepo.List(ctx, filter)
}

func (s *TaxonomyService) GetTag(ctx context.Context, id, locale string, viewEdit bool) (*domain.Tag, error) {
	return s.tagRepo.GetDetail(ctx, id, locale, viewEdit)
}

func (s *TaxonomyService) CreateTag(ctx context.Context, in domain.CreateTagInput) (*domain.Tag, error) {
	return createSlugStatusFromInput(ctx, in, s.tagCreator())
}

func (s *TaxonomyService) UpdateTag(ctx context.Context, id string, in domain.UpdateTagInput) (*domain.Tag, error) {
	return updateSlugStatusRepo(ctx, id, in, s.tagRepo.GetDetail, s.tagRepo.Save, applyTagUpdate)
}

func (s *TaxonomyService) DeleteTag(ctx context.Context, id string) error {
	return s.tagRepo.SoftDelete(ctx, id)
}

func (s *TaxonomyService) HardDeleteTag(ctx context.Context, id string) error {
	return s.tagRepo.HardDelete(ctx, id)
}

// --- CourseLevel -------------------------------------------------------------

func (s *TaxonomyService) ListCourseLevels(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseLevel, int64, error) {
	return s.courseLevelRepo.List(ctx, filter)
}

func (s *TaxonomyService) ListCourseLevelsFull(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseLevel, int64, error) {
	filter.IncludeDeleted = true
	return s.courseLevelRepo.List(ctx, filter)
}

func (s *TaxonomyService) GetCourseLevel(ctx context.Context, id, locale string, viewEdit bool) (*domain.CourseLevel, error) {
	return s.courseLevelRepo.GetDetail(ctx, id, locale, viewEdit)
}

func (s *TaxonomyService) CreateCourseLevel(ctx context.Context, in domain.CreateCourseLevelInput) (*domain.CourseLevel, error) {
	return createSlugStatusFromInput(ctx, in, s.courseLevelCreator())
}

func (s *TaxonomyService) UpdateCourseLevel(ctx context.Context, id string, in domain.UpdateCourseLevelInput) (*domain.CourseLevel, error) {
	return updateSlugStatusRepo(ctx, id, in, s.courseLevelRepo.GetDetail, s.courseLevelRepo.Save, applyCourseLevelUpdate)
}

func (s *TaxonomyService) DeleteCourseLevel(ctx context.Context, id string) error {
	return s.courseLevelRepo.SoftDelete(ctx, id)
}

func (s *TaxonomyService) HardDeleteCourseLevel(ctx context.Context, id string) error {
	return s.courseLevelRepo.HardDelete(ctx, id)
}

// --- internal helpers --------------------------------------------------------

// ErrTaxonomyValidation is returned when tree or description payload fails validation.
var ErrTaxonomyValidation = errors.New("taxonomy validation failed")

// validateResolvedTree replaces the old validateChildTopics/validateChildren
// (identical wrappers that re-ran taxpkg.NormalizeTreeSlugs internally on
// already-normalized input from prepareTreeWrite before validating — a
// harmless but wasted double-normalize). Now that resolveTreeForCreate/
// resolveTreeForUpdate produce final, already-resolved slugs directly, no
// second normalize pass is needed here.
func validateResolvedTree(nodes []taxpkg.TreeNode) error {
	if nodes == nil {
		return nil
	}
	if err := taxpkg.ValidateTree(nodes, taxpkg.ValidateTreeOpts{}); err != nil {
		return errors.Join(ErrTaxonomyValidation, err)
	}
	return nil
}

// resolveTreeForCreate resolves slugs for a brand-new tree (create path —
// no prior tree exists, so every node is "new" for preserve-by-id purposes).
func resolveTreeForCreate(nodes []taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	return resolveTreeForWrite(nodes, nil)
}

// resolveTreeForUpdate resolves slugs for an update, given the entity's
// previously-stored tree (oldByID) for preserve-by-id lookups.
func resolveTreeForUpdate(nodes []taxpkg.TreeNode, oldByID map[string]taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	return resolveTreeForWrite(nodes, oldByID)
}

func resolveTreeForWrite(nodes []taxpkg.TreeNode, oldByID map[string]taxpkg.TreeNode) ([]taxpkg.TreeNode, error) {
	canonized, err := canonicalizeTreeLocales(nodes)
	if err != nil {
		return nil, err
	}
	resolved, conflict, err := taxpkg.PrepareTreeForWriteWithSlugPolicy(canonized, oldByID, "node")
	if err != nil {
		return nil, err
	}
	if conflict {
		return nil, domain.ErrTaxonomyCanonicalConflict
	}
	if err := validateResolvedTree(resolved); err != nil {
		return nil, err
	}
	return resolved, nil
}

func validateOutcomePayload(short string, desc []string) error {
	short = strings.TrimSpace(short)
	if short == "" || len(short) > 100 {
		return errors.Join(ErrTaxonomyValidation, errors.New("short_description must be 1-100 characters"))
	}
	if desc == nil {
		desc = []string{}
	}
	if err := taxpkg.ValidateDescriptionParagraphs(desc, taxpkg.DefaultMaxDescriptionItems, taxpkg.DefaultMaxDescriptionLen); err != nil {
		return errors.Join(ErrTaxonomyValidation, err)
	}
	return nil
}

func (s *TaxonomyService) mutateImageFileID(ctx context.Context, dst **string, imageFileID *string) error {
	if imageFileID == nil {
		return nil
	}
	next := strings.TrimSpace(*imageFileID)
	if next == "" {
		*dst = nil
		return nil
	}
	if s.mediaValidator != nil {
		if _, err := s.mediaValidator.LoadValidatedProfileImageFile(ctx, next); err != nil {
			return err
		}
	}
	*dst = &next
	return nil
}

// trimmedNameAndStatus replaces the old trimmedTaxonomyFields, which also
// returned a synchronously-computed slug — slug resolution now requires DB
// access and lives in the infra layer (internal/taxonomy/infra/slug.go).
func trimmedNameAndStatus(name, status string) (string, string) {
	n := strings.TrimSpace(name)
	st := strings.ToUpper(strings.TrimSpace(status))
	if st == "" {
		st = "ACTIVE"
	}
	return n, st
}

func imageFileIDStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func stringPtrIfNotBlank(id string) *string {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	return &id
}
