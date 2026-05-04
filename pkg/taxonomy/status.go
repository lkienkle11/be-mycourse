package taxonomy

import (
	"strings"

	"mycourse-io-be/constants"
)

// NormalizeTaxonomyStatus maps free-form request strings to taxonomy status enum values.
func NormalizeTaxonomyStatus(raw string) constants.TaxonomyStatus {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(constants.TaxonomyStatusInactive):
		return constants.TaxonomyStatusInactive
	default:
		return constants.TaxonomyStatusActive
	}
}
