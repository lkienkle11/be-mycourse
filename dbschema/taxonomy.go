package dbschema

const (
	taxonomyCourseLevels = "course_levels"
	taxonomyCategories   = "categories"
	taxonomyTags         = "tags"
)

// Taxonomy exposes taxonomy table names for models and repositories.
var Taxonomy taxonomyNS

type taxonomyNS struct{}

func (taxonomyNS) CourseLevels() string { return taxonomyCourseLevels }
func (taxonomyNS) Categories() string   { return taxonomyCategories }
func (taxonomyNS) Tags() string         { return taxonomyTags }
