package slug

// AvailableFunc reports whether candidate is free for the caller's table.
// Callers fold any extra rule into it (excluding the row's own id on update,
// reserved words, reserved history) so the algorithm below stays pure.
type AvailableFunc func(candidate string) (bool, error)

// ResolveCreate picks the slug to store on create.
//
//   - manual != "": a free manual slug is used as is; a taken one yields a
//     *ConflictError carrying a free recommendation and a nil error (the
//     caller decides how to surface it, typically HTTP 409).
//   - manual == "": a base is derived from name via GenerateAutoSlugBase; a
//     free base is used, otherwise (or when the base is only the fallback
//     kind literal) random suffixes are tried until one is free.
//
// avail is a UX fast path only; the database unique index is the safety net.
func ResolveCreate(manual, name string, kind Kind, maxLen int, avail AvailableFunc) (string, *ConflictError, error) {
	if manual != "" {
		ok, err := avail(manual)
		if err != nil {
			return "", nil, err
		}
		if ok {
			return manual, nil, nil
		}
		recommended, err := resolveWithSuffix(manual, maxLen, avail)
		if err != nil {
			return "", nil, err
		}
		return "", &ConflictError{RecommendedSlug: recommended}, nil
	}
	base, mustSuffix := GenerateAutoSlugBase(name, maxLen, kind)
	if !mustSuffix {
		ok, err := avail(base)
		if err != nil {
			return "", nil, err
		}
		if ok {
			return base, nil, nil
		}
	}
	final, err := resolveWithSuffix(base, maxLen, avail)
	return final, nil, err
}

// ResolveUpdate picks the slug to store on update: newSlug when free,
// otherwise a randomly suffixed variant. It never reports a conflict.
func ResolveUpdate(newSlug string, maxLen int, avail AvailableFunc) (string, error) {
	ok, err := avail(newSlug)
	if err != nil {
		return "", err
	}
	if ok {
		return newSlug, nil
	}
	return resolveWithSuffix(newSlug, maxLen, avail)
}

func resolveWithSuffix(base string, maxLen int, avail AvailableFunc) (string, error) {
	return RetryWithSuffix(func(suffix string) (string, bool, error) {
		candidate, err := BuildSuffixedCandidate(base, suffix, maxLen)
		if err != nil {
			return "", false, err
		}
		ok, err := avail(candidate)
		return candidate, ok, err
	})
}
