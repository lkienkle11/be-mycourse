package taxonomy

import (
	"strings"

	"mycourse-io-be/pkg/entities"
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	repo "mycourse-io-be/repository/taxonomy"
)

func ListCourseLevels(filter dto.CourseLevelFilter) ([]models.CourseLevel, int64, error) {
	return repo.NewCourseLevelRepository(models.DB).ListCourseLevels(filter)
}

func CreateCourseLevel(actorID uint, req dto.CreateCourseLevelRequest) (*models.CourseLevel, error) {
	row := &models.CourseLevel{
		CourseLevel: entities.CourseLevel{
			Name:   strings.TrimSpace(req.Name),
			Slug:   strings.TrimSpace(req.Slug),
			Status: normalizeTaxonomyStatus(req.Status),
		},
	}
	if actorID > 0 {
		row.CreatedBy = &actorID
	}
	if err := repo.NewCourseLevelRepository(models.DB).CreateCourseLevel(row); err != nil {
		return nil, err
	}
	return row, nil
}

func UpdateCourseLevel(id uint, req dto.UpdateCourseLevelRequest) (*models.CourseLevel, error) {
	r := repo.NewCourseLevelRepository(models.DB)
	row, err := r.GetCourseLevelByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil && strings.TrimSpace(*req.Slug) != "" {
		row.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		row.Status = normalizeTaxonomyStatus(*req.Status)
	}

	if err := r.UpdateCourseLevel(row); err != nil {
		return nil, err
	}
	return row, nil
}

func DeleteCourseLevel(id uint) error {
	return repo.NewCourseLevelRepository(models.DB).DeleteCourseLevel(id)
}
