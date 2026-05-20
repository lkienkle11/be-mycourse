// Package application contains the TAXONOMY bounded-context use-case layer.
package application

import (
	"context"
	"errors"
	"strings"

	"mycourse-io-be/internal/taxonomy/domain"
	taxpkg "mycourse-io-be/pkg/taxonomy"
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
	n, sl, st := trimmedTaxonomyFields(in.Name, in.Slug, in.Status)
	t := &domain.CourseTopic{
		Name: n, Slug: sl, Status: st, ChildTopics: in.ChildTopics,
		CreatedBy: uintPtrIfPos(in.ActorID),
	}
	if fileID != "" {
		t.ImageFileID = &fileID
	}
	if err := s.topicRepo.Create(ctx, t); err != nil {
		return nil, err
	}
	return s.topicRepo.GetByID(ctx, t.ID)
}

func (s *TaxonomyService) UpdateTopic(ctx context.Context, id uint, in domain.UpdateCourseTopicInput) (*domain.CourseTopic, error) {
	row, err := s.topicRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyFields(&row.Name, &row.Slug, &row.Status, in.Name, in.Slug, in.Status)
	if in.ChildTopics != nil {
		if err := validateChildTopics(*in.ChildTopics); err != nil {
			return nil, err
		}
		row.ChildTopics = *in.ChildTopics
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

func (s *TaxonomyService) DeleteTopic(ctx context.Context, id uint) error {
	row, err := s.topicRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.topicRepo.Delete(ctx, id); err != nil {
		return err
	}
	if prevFileID != "" && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanupForFileID(ctx, prevFileID)
	}
	return nil
}

func (s *TaxonomyService) GetTopic(ctx context.Context, id uint) (*domain.CourseTopic, error) {
	return s.topicRepo.GetByID(ctx, id)
}

// --- CourseOutcome -----------------------------------------------------------

func (s *TaxonomyService) ListCourseOutcomes(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseOutcome, int64, error) {
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
		CreatedBy:        uintPtrIfPos(in.ActorID),
	}
	if fileID != "" {
		o.ImageFileID = &fileID
	}
	if err := s.outcomeRepo.Create(ctx, o); err != nil {
		return nil, err
	}
	return s.outcomeRepo.GetByID(ctx, o.ID)
}

func (s *TaxonomyService) UpdateCourseOutcome(ctx context.Context, id uint, in domain.UpdateCourseOutcomeInput) (*domain.CourseOutcome, error) {
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

func (s *TaxonomyService) DeleteCourseOutcome(ctx context.Context, id uint) error {
	row, err := s.outcomeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.outcomeRepo.Delete(ctx, id); err != nil {
		return err
	}
	if prevFileID != "" && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanupForFileID(ctx, prevFileID)
	}
	return nil
}

// --- CourseSkill -------------------------------------------------------------

func (s *TaxonomyService) ListCourseSkills(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseSkill, int64, error) {
	return s.skillRepo.List(ctx, filter)
}

func (s *TaxonomyService) CreateCourseSkill(ctx context.Context, in domain.CreateCourseSkillInput) (*domain.CourseSkill, error) {
	if err := validateChildren(in.Children); err != nil {
		return nil, err
	}
	n, sl, st := trimmedTaxonomyFields(in.Name, in.Slug, in.Status)
	sk := &domain.CourseSkill{
		Name: n, Slug: sl, Status: st, Children: in.Children,
		CreatedBy: uintPtrIfPos(in.ActorID),
	}
	if err := s.skillRepo.Create(ctx, sk); err != nil {
		return nil, err
	}
	return s.skillRepo.GetByID(ctx, sk.ID)
}

func (s *TaxonomyService) UpdateCourseSkill(ctx context.Context, id uint, in domain.UpdateCourseSkillInput) (*domain.CourseSkill, error) {
	row, err := s.skillRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyFields(&row.Name, &row.Slug, &row.Status, in.Name, in.Slug, in.Status)
	if in.Children != nil {
		if err := validateChildren(*in.Children); err != nil {
			return nil, err
		}
		row.Children = *in.Children
	}
	if err := s.skillRepo.Save(ctx, row); err != nil {
		return nil, err
	}
	return s.skillRepo.GetByID(ctx, id)
}

func (s *TaxonomyService) DeleteCourseSkill(ctx context.Context, id uint) error {
	return s.skillRepo.Delete(ctx, id)
}

// --- Tag ---------------------------------------------------------------------

func (s *TaxonomyService) ListTags(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.Tag, int64, error) {
	return s.tagRepo.List(ctx, filter)
}

func (s *TaxonomyService) CreateTag(ctx context.Context, in domain.CreateTagInput) (*domain.Tag, error) {
	n, sl, st := trimmedTaxonomyFields(in.Name, in.Slug, in.Status)
	t := &domain.Tag{Name: n, Slug: sl, Status: st, CreatedBy: uintPtrIfPos(in.ActorID)}
	if err := s.tagRepo.Create(ctx, t); err != nil {
		return nil, err
	}
	return s.tagRepo.GetByID(ctx, t.ID)
}

func (s *TaxonomyService) UpdateTag(ctx context.Context, id uint, in domain.UpdateTagInput) (*domain.Tag, error) {
	t, err := s.tagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyFields(&t.Name, &t.Slug, &t.Status, in.Name, in.Slug, in.Status)
	if err := s.tagRepo.Save(ctx, t); err != nil {
		return nil, err
	}
	return s.tagRepo.GetByID(ctx, id)
}

func (s *TaxonomyService) DeleteTag(ctx context.Context, id uint) error {
	return s.tagRepo.Delete(ctx, id)
}

// --- CourseLevel -------------------------------------------------------------

func (s *TaxonomyService) ListCourseLevels(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.CourseLevel, int64, error) {
	return s.courseLevelRepo.List(ctx, filter)
}

func (s *TaxonomyService) CreateCourseLevel(ctx context.Context, in domain.CreateCourseLevelInput) (*domain.CourseLevel, error) {
	n, sl, st := trimmedTaxonomyFields(in.Name, in.Slug, in.Status)
	cl := &domain.CourseLevel{Name: n, Slug: sl, Status: st, CreatedBy: uintPtrIfPos(in.ActorID)}
	if err := s.courseLevelRepo.Create(ctx, cl); err != nil {
		return nil, err
	}
	return s.courseLevelRepo.GetByID(ctx, cl.ID)
}

func (s *TaxonomyService) UpdateCourseLevel(ctx context.Context, id uint, in domain.UpdateCourseLevelInput) (*domain.CourseLevel, error) {
	cl, err := s.courseLevelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyFields(&cl.Name, &cl.Slug, &cl.Status, in.Name, in.Slug, in.Status)
	if err := s.courseLevelRepo.Save(ctx, cl); err != nil {
		return nil, err
	}
	return s.courseLevelRepo.GetByID(ctx, id)
}

func (s *TaxonomyService) DeleteCourseLevel(ctx context.Context, id uint) error {
	return s.courseLevelRepo.Delete(ctx, id)
}

// --- internal helpers --------------------------------------------------------

// ErrTaxonomyValidation is returned when tree or description payload fails validation.
var ErrTaxonomyValidation = errors.New("taxonomy validation failed")

func validateChildTopics(nodes []taxpkg.TreeNode) error {
	if nodes == nil {
		return nil
	}
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

func trimmedTaxonomyFields(name, slug, status string) (string, string, string) {
	n := strings.TrimSpace(name)
	sl := strings.TrimSpace(slug)
	st := strings.ToUpper(strings.TrimSpace(status))
	if st == "" {
		st = "ACTIVE"
	}
	return n, sl, st
}

func applyOptionalTaxonomyFields(dstName, dstSlug, dstStatus *string, name, slug, status *string) {
	if name != nil {
		*dstName = strings.TrimSpace(*name)
	}
	if slug != nil {
		*dstSlug = strings.TrimSpace(*slug)
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

func uintPtrIfPos(id uint) *uint {
	if id > 0 {
		return &id
	}
	return nil
}
