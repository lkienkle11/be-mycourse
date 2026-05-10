package taxonomy

import (
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	repo "mycourse-io-be/repository/taxonomy"
)

func ListTags(filter dto.TagFilter) ([]models.Tag, int64, error) {
	rows, total, err := repo.NewTagRepository(models.DB).ListTags(filter)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func CreateTag(actorID uint, req dto.CreateTagRequest) (*models.Tag, error) {
	n, s, st := trimmedTaxonomyFields(req.Name, req.Slug, req.Status)
	row := &models.Tag{
		Name:   n,
		Slug:   s,
		Status: st,
	}
	if actorID > 0 {
		row.CreatedBy = &actorID
	}
	if err := repo.NewTagRepository(models.DB).CreateTag(row); err != nil {
		return nil, err
	}
	return row, nil
}

func UpdateTag(id uint, req dto.UpdateTagRequest) (*models.Tag, error) {
	r := repo.NewTagRepository(models.DB)
	row, err := r.GetTagByID(id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyNameSlugStatus(&row.Name, &row.Slug, &row.Status, req.Name, req.Slug, req.Status)
	if err := r.UpdateTag(row); err != nil {
		return nil, err
	}
	return row, nil
}

func DeleteTag(id uint) error {
	return repo.NewTagRepository(models.DB).DeleteTag(id)
}

func ListCourseLevels(filter dto.CourseLevelFilter) ([]models.CourseLevel, int64, error) {
	rows, total, err := repo.NewCourseLevelRepository(models.DB).ListCourseLevels(filter)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func CreateCourseLevel(actorID uint, req dto.CreateCourseLevelRequest) (*models.CourseLevel, error) {
	n, s, st := trimmedTaxonomyFields(req.Name, req.Slug, req.Status)
	row := &models.CourseLevel{
		Name:   n,
		Slug:   s,
		Status: st,
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
	applyOptionalTaxonomyNameSlugStatus(&row.Name, &row.Slug, &row.Status, req.Name, req.Slug, req.Status)
	if err := r.UpdateCourseLevel(row); err != nil {
		return nil, err
	}
	return row, nil
}

func DeleteCourseLevel(id uint) error {
	return repo.NewCourseLevelRepository(models.DB).DeleteCourseLevel(id)
}
