package taxonomy

import (
	"mycourse-io-be/dto"
	"mycourse-io-be/models"
	"mycourse-io-be/pkg/logic/mapping"
	repo "mycourse-io-be/repository/taxonomy"
)

func ListTags(filter dto.TagFilter) ([]dto.TagResponse, int64, error) {
	rows, total, err := repo.NewTagRepository(models.DB).ListTags(filter)
	if err != nil {
		return nil, 0, err
	}
	return mapping.ToTagResponsesFromModels(rows), total, nil
}

func CreateTag(actorID uint, req dto.CreateTagRequest) (*dto.TagResponse, error) {
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
	dtoRow := mapping.ToTagResponseModel(*row)
	return &dtoRow, nil
}

func UpdateTag(id uint, req dto.UpdateTagRequest) (*dto.TagResponse, error) {
	r := repo.NewTagRepository(models.DB)
	row, err := r.GetTagByID(id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyNameSlugStatus(&row.Name, &row.Slug, &row.Status, req.Name, req.Slug, req.Status)
	if err := r.UpdateTag(row); err != nil {
		return nil, err
	}
	dtoRow := mapping.ToTagResponseModel(*row)
	return &dtoRow, nil
}

func DeleteTag(id uint) error {
	return repo.NewTagRepository(models.DB).DeleteTag(id)
}

func ListCourseLevels(filter dto.CourseLevelFilter) ([]dto.CourseLevelResponse, int64, error) {
	rows, total, err := repo.NewCourseLevelRepository(models.DB).ListCourseLevels(filter)
	if err != nil {
		return nil, 0, err
	}
	return mapping.ToCourseLevelResponsesFromModels(rows), total, nil
}

func CreateCourseLevel(actorID uint, req dto.CreateCourseLevelRequest) (*dto.CourseLevelResponse, error) {
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
	dtoRow := mapping.ToCourseLevelResponseModel(*row)
	return &dtoRow, nil
}

func UpdateCourseLevel(id uint, req dto.UpdateCourseLevelRequest) (*dto.CourseLevelResponse, error) {
	r := repo.NewCourseLevelRepository(models.DB)
	row, err := r.GetCourseLevelByID(id)
	if err != nil {
		return nil, err
	}
	applyOptionalTaxonomyNameSlugStatus(&row.Name, &row.Slug, &row.Status, req.Name, req.Slug, req.Status)
	if err := r.UpdateCourseLevel(row); err != nil {
		return nil, err
	}
	dtoRow := mapping.ToCourseLevelResponseModel(*row)
	return &dtoRow, nil
}

func DeleteCourseLevel(id uint) error {
	return repo.NewCourseLevelRepository(models.DB).DeleteCourseLevel(id)
}
