package middleware

import (
	"math"

	"mycourse-io-be/internal/shared/resilience"
)

func effectiveRateAttempts(attempts int) int {
	if attempts < 1 {
		return attempts
	}
	factor := resilience.Global.AttemptsFactor()
	if factor >= 1 {
		return attempts
	}
	scaled := int(math.Floor(float64(attempts) * factor))
	if scaled < 1 {
		return 1
	}
	return scaled
}
