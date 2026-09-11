package application

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"mycourse-io-be/internal/authorization/domain"
)

func validateConditions(conditions []domain.Condition) error {
	for _, condition := range conditions {
		if strings.TrimSpace(condition.Key) == "" || len(condition.Values) == 0 {
			return domain.ErrInvalidGrant
		}
		switch condition.Operator {
		case domain.ConditionStringEquals, domain.ConditionBoolEquals, domain.ConditionNumericEquals,
			domain.ConditionDateBefore, domain.ConditionDateAfter:
		default:
			return fmt.Errorf("%w: unsupported condition operator %q", domain.ErrInvalidGrant, condition.Operator)
		}
	}
	return nil
}

func conditionsMatch(conditions []domain.Condition, attributes map[string]any) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}
	for _, condition := range conditions {
		if strings.TrimSpace(condition.Key) == "" || len(condition.Values) == 0 {
			return false, nil
		}
		switch condition.Operator {
		case domain.ConditionStringEquals, domain.ConditionBoolEquals, domain.ConditionNumericEquals,
			domain.ConditionDateBefore, domain.ConditionDateAfter:
		default:
			return false, nil
		}
		actual, exists := attributes[condition.Key]
		if !exists {
			return false, nil
		}
		matched := false
		for _, expected := range condition.Values {
			ok, err := conditionValueMatches(condition.Operator, actual, expected)
			if err != nil {
				return false, err
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func conditionValueMatches(operator domain.ConditionOperator, actual, expected any) (bool, error) {
	switch operator {
	case domain.ConditionStringEquals:
		a, aok := actual.(string)
		e, eok := expected.(string)
		return aok && eok && a == e, nil
	case domain.ConditionBoolEquals:
		a, aok := actual.(bool)
		e, eok := expected.(bool)
		return aok && eok && a == e, nil
	case domain.ConditionNumericEquals:
		a, aok := numericValue(actual)
		e, eok := numericValue(expected)
		return aok && eok && math.Abs(a-e) < 1e-9, nil
	case domain.ConditionDateBefore, domain.ConditionDateAfter:
		a, aok := unixValue(actual)
		e, eok := unixValue(expected)
		if !aok || !eok {
			return false, nil
		}
		if operator == domain.ConditionDateBefore {
			return a < e, nil
		}
		return a > e, nil
	default:
		return false, nil
	}
}

func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func unixValue(value any) (int64, bool) {
	if number, ok := numericValue(value); ok {
		return int64(number), true
	}
	switch v := value.(type) {
	case time.Time:
		return v.Unix(), true
	case string:
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			return parsed.Unix(), true
		}
		parsed, err := strconv.ParseInt(v, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
