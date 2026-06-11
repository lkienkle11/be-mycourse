package validate

import (
	"reflect"
	"strconv"

	"github.com/go-playground/validator/v10"

	"mycourse-io-be/internal/shared/utils"
)

func registerTextRules(v *validator.Validate) {
	_ = v.RegisterValidation("nonwhitespace_min", validateNonWhitespaceMin)
	_ = v.RegisterValidation("delta_nonwhitespace_min", validateDeltaNonWhitespaceMin)
}

func validateNonWhitespaceMin(fl validator.FieldLevel) bool {
	min, err := strconv.Atoi(fl.Param())
	if err != nil {
		return false
	}
	return utils.CountNonWhitespace(readStringField(fl)) >= min
}

func validateDeltaNonWhitespaceMin(fl validator.FieldLevel) bool {
	min, err := strconv.Atoi(fl.Param())
	if err != nil {
		return false
	}
	return utils.CountDeltaNonWhitespace(readStringField(fl)) >= min
}

func readStringField(fl validator.FieldLevel) string {
	field := fl.Field()
	if field.Kind() == reflect.String {
		return field.String()
	}
	return ""
}
