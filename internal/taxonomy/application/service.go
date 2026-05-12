// Package application contains the TAXONOMY bounded-context use-case layer.
package application

import (
	"context"
	"strings"

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
	categoryRepo    domain.CategoryRepository
	tagRepo         domain.TagRepository
	courseLevelRepo domain.CourseLevelRepository
	mediaValidator  MediaFileValidator
	orphanEnqueuer  OrphanImageEnqueuer
}

// NewTaxonomyService constructs a TaxonomyService.
func NewTaxonomyService(
	categoryRepo domain.CategoryRepository,
	tagRepo domain.TagRepository,
	courseLevelRepo domain.CourseLevelRepository,
	mediaValidator MediaFileValidator,
	orphanEnqueuer OrphanImageEnqueuer,
) *TaxonomyService {
	return &TaxonomyService{
		categoryRepo:    categoryRepo,
		tagRepo:         tagRepo,
		courseLevelRepo: courseLevelRepo,
		mediaValidator:  mediaValidator,
		orphanEnqueuer:  orphanEnqueuer,
	}
}

// --- Category ----------------------------------------------------------------

func (s *TaxonomyService) ListCategories(ctx context.Context, filter domain.TaxonomyFilter) ([]domain.Category, int64, error) {
	return s.categoryRepo.List(ctx, filter)
}

func (s *TaxonomyService) CreateCategory(ctx context.Context, in domain.CreateCategoryInput) (*domain.Category, error) {
	fileID := strings.TrimSpace(in.ImageFileID)
	if fileID != "" && s.mediaValidator != nil {
		if _, err := s.mediaValidator.LoadValidatedProfileImageFile(ctx, fileID); err != nil {
			return nil, err
		}
	}
	n, sl, st := trimmedTaxonomyFields(in.Name, in.Slug, in.Status)
	c := &domain.Category{
		Name: n, Slug: sl, Status: st,
		CreatedBy: uintPtrIfPos(in.ActorID),
	}
	if fileID != "" {
		c.ImageFileID = &fileID
	}
	if err := s.categoryRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return s.categoryRepo.GetByID(ctx, c.ID)
}

func (s *TaxonomyService) UpdateCategory(ctx context.Context, id uint, in domain.UpdateCategoryInput) (*domain.Category, error) {
	row, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyFields(&row.Name, &row.Slug, &row.Status, in.Name, in.Slug, in.Status)

	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.mutateCategoryImageFileID(ctx, row, in.ImageFileID); err != nil {
		return nil, err
	}
	if err := s.categoryRepo.Save(ctx, row); err != nil {
		return nil, err
	}
	nextFileID := imageFileIDStr(row.ImageFileID)
	if in.ImageFileID != nil && prevFileID != "" && prevFileID != nextFileID && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanupForFileID(ctx, prevFileID)
	}
	return s.categoryRepo.GetByID(ctx, id)
}

func (s *TaxonomyService) DeleteCategory(ctx context.Context, id uint) error {
	row, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	prevFileID := imageFileIDStr(row.ImageFileID)
	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		return err
	}
	if prevFileID != "" && s.orphanEnqueuer != nil {
		s.orphanEnqueuer.EnqueueOrphanCleanupForFileID(ctx, prevFileID)
	}
	return nil
}

func (s *TaxonomyService) GetCategory(ctx context.Context, id uint) (*domain.Category, error) {
	return s.categoryRepo.GetByID(ctx, id)
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

func (s *TaxonomyService) mutateCategoryImageFileID(ctx context.Context, row *domain.Category, imageFileID *string) error {
	if imageFileID == nil {
		return nil
	}
	next := strings.TrimSpace(*imageFileID)
	if next == "" {
		row.ImageFileID = nil
		return nil
	}
	if s.mediaValidator != nil {
		if _, err := s.mediaValidator.LoadValidatedProfileImageFile(ctx, next); err != nil {
			return err
		}
	}
	row.ImageFileID = &next
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
