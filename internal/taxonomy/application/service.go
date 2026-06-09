// Package application contains the TAXONOMY bounded-context use-case layer.
package application

import (
	"context"
	"errors"
	"strings"

	taxpkg "mycourse-io-be/internal/shared/taxonomy"
	"mycourse-io-be/internal/shared/utils"
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

func (s *TaxonomyService) CreateTopic(ctx context.Context, in domain.CreateCourseTopicInput) (*domain.CourseTopic, error) {
	if err := validateChildTopics(in.ChildTopics); err != nil {
		return nil, err
	}
	fileID := strings.TrimSpace(in.ImageFileID)
	if fileID != "" && s.mediaValidator != nil {
		if _, err := s.mediaValidator.LoadValidatedProfileImageFile(ctx, fileID); err != nil {
			return nil, err
		}
	}
	childTopics := taxpkg.NormalizeTreeSlugs(in.ChildTopics)
	n, sl, st := trimmedTaxonomyFields(in.Name, in.Status)
	t := &domain.CourseTopic{
		Name: n, Slug: sl, Status: st, ChildTopics: childTopics,
		CreatedBy: stringPtrIfNotBlank(in.ActorID),
	}
	if fileID != "" {
		t.ImageFileID = &fileID
	}
	if err := s.topicRepo.Create(ctx, t); err != nil {
		return nil, err
	}
	return s.topicRepo.GetByID(ctx, t.ID)
}

func (s *TaxonomyService) UpdateTopic(ctx context.Context, id string, in domain.UpdateCourseTopicInput) (*domain.CourseTopic, error) {
	row, err := s.topicRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyFields(&row.Name, &row.Slug, &row.Status, in.Name, in.Status)
	if in.ChildTopics != nil {
		normalized := taxpkg.NormalizeTreeSlugs(*in.ChildTopics)
		if err := validateChildTopics(normalized); err != nil {
			return nil, err
		}
		row.ChildTopics = normalized
	}
	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.mutateImageFileID(ctx, &row.ImageFileID, in.ImageFileID); err != nil {
		return nil, err
	}
	if err := s.topicRepo.Save(ctx, row); err != nil {
		return nil, err
	}
	nextFileID := imageFileIDStr(row.ImageFileID)
	if in.ImageFileID != nil && prevFileID != "" && prevFileID != nextFileID && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanupForFileID(ctx, prevFileID)
	}
	return s.topicRepo.GetByID(ctx, id)
}

func (s *TaxonomyService) DeleteTopic(ctx context.Context, id string) error {
	return s.topicRepo.SoftDelete(ctx, id)
}

func (s *TaxonomyService) HardDeleteTopic(ctx context.Context, id string) error {
	return s.deleteWithOrphanImage(ctx, id, imageIDLoaderTopic(s), s.topicRepo.HardDelete)
}

func (s *TaxonomyService) GetTopic(ctx context.Context, id string) (*domain.CourseTopic, error) {
	return s.topicRepo.GetByID(ctx, id)
}

// --- CourseOutcome -----------------------------------------------------------

func (s *TaxonomyService) ListCourseOutcomes(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseOutcome, int64, error) {
	return s.outcomeRepo.List(ctx, filter)
}

func (s *TaxonomyService) ListCourseOutcomesFull(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseOutcome, int64, error) {
	filter.IncludeDeleted = true
	return s.outcomeRepo.List(ctx, filter)
}

func (s *TaxonomyService) CreateCourseOutcome(ctx context.Context, in domain.CreateCourseOutcomeInput) (*domain.CourseOutcome, error) {
	if err := validateOutcomePayload(in.ShortDescription, in.Description); err != nil {
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
		ShortDescription: strings.TrimSpace(in.ShortDescription),
		Description:      in.Description,
		Status:           st,
		CreatedBy:        stringPtrIfNotBlank(in.ActorID),
	}
	if fileID != "" {
		o.ImageFileID = &fileID
	}
	if err := s.outcomeRepo.Create(ctx, o); err != nil {
		return nil, err
	}
	return s.outcomeRepo.GetByID(ctx, o.ID)
}

func (s *TaxonomyService) UpdateCourseOutcome(ctx context.Context, id string, in domain.UpdateCourseOutcomeInput) (*domain.CourseOutcome, error) {
	row, err := s.outcomeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.ShortDescription != nil {
		row.ShortDescription = strings.TrimSpace(*in.ShortDescription)
	}
	if in.Description != nil {
		row.Description = *in.Description
	}
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "" {
			row.Status = v
		}
	}
	if err := validateOutcomePayload(row.ShortDescription, row.Description); err != nil {
		return nil, err
	}
	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.mutateImageFileID(ctx, &row.ImageFileID, in.ImageFileID); err != nil {
		return nil, err
	}
	if err := s.outcomeRepo.Save(ctx, row); err != nil {
		return nil, err
	}
	nextFileID := imageFileIDStr(row.ImageFileID)
	if in.ImageFileID != nil && prevFileID != "" && prevFileID != nextFileID && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanupForFileID(ctx, prevFileID)
	}
	return s.outcomeRepo.GetByID(ctx, id)
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

func (s *TaxonomyService) CreateCourseSkill(ctx context.Context, in domain.CreateCourseSkillInput) (*domain.CourseSkill, error) {
	if err := validateChildren(in.Children); err != nil {
		return nil, err
	}
	children := taxpkg.NormalizeTreeSlugs(in.Children)
	n, sl, st := trimmedTaxonomyFields(in.Name, in.Status)
	sk := &domain.CourseSkill{
		Name: n, Slug: sl, Status: st, Children: children,
		CreatedBy: stringPtrIfNotBlank(in.ActorID),
	}
	if err := s.skillRepo.Create(ctx, sk); err != nil {
		return nil, err
	}
	return s.skillRepo.GetByID(ctx, sk.ID)
}

func (s *TaxonomyService) UpdateCourseSkill(ctx context.Context, id string, in domain.UpdateCourseSkillInput) (*domain.CourseSkill, error) {
	row, err := s.skillRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyFields(&row.Name, &row.Slug, &row.Status, in.Name, in.Status)
	if in.Children != nil {
		normalized := taxpkg.NormalizeTreeSlugs(*in.Children)
		if err := validateChildren(normalized); err != nil {
			return nil, err
		}
		row.Children = normalized
	}
	if err := s.skillRepo.Save(ctx, row); err != nil {
		return nil, err
	}
	return s.skillRepo.GetByID(ctx, id)
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

func (s *TaxonomyService) CreateTag(ctx context.Context, in domain.CreateTagInput) (*domain.Tag, error) {
	return createSlugStatusFromInput(ctx, in, s.tagCreator())
}

func (s *TaxonomyService) UpdateTag(ctx context.Context, id string, in domain.UpdateTagInput) (*domain.Tag, error) {
	return updateSlugStatusRepo(ctx, id, in, s.tagRepo.GetByID, s.tagRepo.Save)
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

func (s *TaxonomyService) CreateCourseLevel(ctx context.Context, in domain.CreateCourseLevelInput) (*domain.CourseLevel, error) {
	return createSlugStatusFromInput(ctx, in, s.courseLevelCreator())
}

func (s *TaxonomyService) UpdateCourseLevel(ctx context.Context, id string, in domain.UpdateCourseLevelInput) (*domain.CourseLevel, error) {
	return updateSlugStatusRepo(ctx, id, in, s.courseLevelRepo.GetByID, s.courseLevelRepo.Save)
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

func validateChildTopics(nodes []taxpkg.TreeNode) error {
	if nodes == nil {
		return nil
	}
	nodes = taxpkg.NormalizeTreeSlugs(nodes)
	if err := taxpkg.ValidateTree(nodes, taxpkg.ValidateTreeOpts{}); err != nil {
		return errors.Join(ErrTaxonomyValidation, err)
	}
	return nil
}

func validateChildren(nodes []taxpkg.TreeNode) error {
	return validateChildTopics(nodes)
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

func trimmedTaxonomyFields(name, status string) (string, string, string) {
	n := strings.TrimSpace(name)
	sl := slugFromName(n)
	st := strings.ToUpper(strings.TrimSpace(status))
	if st == "" {
		st = "ACTIVE"
	}
	return n, sl, st
}

func slugFromName(name string) string {
	return strings.TrimSpace(utils.SlugifyName(name))
}

func applyOptionalTaxonomyFields(dstName, dstSlug, dstStatus *string, name, status *string) {
	if name != nil {
		*dstName = strings.TrimSpace(*name)
		*dstSlug = slugFromName(*dstName)
	}
	if status != nil {
		v := strings.ToUpper(strings.TrimSpace(*status))
		if v != "" {
			*dstStatus = v
		}
	}
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
