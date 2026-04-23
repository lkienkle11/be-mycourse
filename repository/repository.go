package repository

import (
	"gorm.io/gorm"

	taxonomyrepo "mycourse-io-be/repository/taxonomy"
)

type TaxonomyRepository struct {
	CourseLevels *taxonomyrepo.CourseLevelRepository
	Categories   *taxonomyrepo.CategoryRepository
	Tags         *taxonomyrepo.TagRepository
}

type Repository struct {
	Taxonomy TaxonomyRepository
}

func New(db *gorm.DB) *Repository {
	return &Repository{
		Taxonomy: TaxonomyRepository{
			CourseLevels: taxonomyrepo.NewCourseLevelRepository(db),
			Categories:   taxonomyrepo.NewCategoryRepository(db),
			Tags:         taxonomyrepo.NewTagRepository(db),
		},
	}
}
