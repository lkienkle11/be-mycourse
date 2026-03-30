package validate

import (
	"github.com/go-playground/validator/v10"

	"mycourse-io-be/pkg/errcode"
)

// V is a shared validator instance (like a Spring LocalValidatorFactoryBean).
var V *validator.Validate

func init() {
	V = validator.New()
}

// Struct runs validation tags on s (use `validate:"required"` etc.).
// For Gin JSON binding, prefer struct tags `binding:"required"` with ShouldBindJSON;
// use Struct for values not bound through Gin or for extra checks after bind.
func Struct(s interface{}) error {
	return V.Struct(s)
}

// FlattenErrors turns validator.ValidationErrors into field rows for JSON APIs.
// Each row includes error_code (app code) alongside field metadata.
func FlattenErrors(err validator.ValidationErrors) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(err))
	for _, e := range err {
		out = append(out, map[string]interface{}{
			"field":      e.Field(),
			"namespace":  e.Namespace(),
			"tag":        e.Tag(),
			"message":    humanTag(e),
			"error_code": errcode.ValidationField,
		})
	}
	return out
}

func humanTag(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "required"
	case "email":
		return "must be a valid email"
	case "min":
		return "below minimum " + e.Param()
	case "max":
		return "above maximum " + e.Param()
	case "len":
		return "must have length " + e.Param()
	case "oneof":
		return "must be one of: " + e.Param()
	default:
		return "failed on " + e.Tag()
	}
}
