package repository

import (
	"gorm.io/gorm"

	mediarepo "mycourse-io-be/repository/media"
	taxonomyrepo "mycourse-io-be/repository/taxonomy"
)

type TaxonomyRepository struct {
	CourseLevels *taxonomyrepo.CourseLevelRepository
	Categories   *taxonomyrepo.CategoryRepository
	Tags         *taxonomyrepo.TagRepository
}

type Repository struct {
	Taxonomy TaxonomyRepository
	Media    *mediarepo.FileRepository
}

func New(db *gorm.DB) *Repository {
	return &Repository{
		Taxonomy: TaxonomyRepository{
			CourseLevels: taxonomyrepo.NewCourseLevelRepository(db),
			Categories:   taxonomyrepo.NewCategoryRepository(db),
			Tags:         taxonomyrepo.NewTagRepository(db),
		},
		Media: mediarepo.NewFileRepository(db),
	}
}
