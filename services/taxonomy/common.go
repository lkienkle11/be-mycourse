package taxonomy

import (
	"strings"

	"mycourse-io-be/constants"
)

func normalizeTaxonomyStatus(raw string) constants.TaxonomyStatus {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(constants.TaxonomyStatusInactive):
		return constants.TaxonomyStatusInactive
	default:
		return constants.TaxonomyStatusActive
	}
}
