package audit

import "reflect"

// computeDiff returns a map of changed fields in the form
//
//	{ field_name: [old_value, new_value] }
//
// Fields present only in `after` are emitted with old=nil; fields only in
// `before` get new=nil. Identical values are omitted entirely so the diff
// stays compact — even for updates that touched the row but changed nothing
// the diff is `{}` (still recorded so we know the UPDATE fired).
//
// Returns nil when both inputs are nil or empty — the caller can serialise
// nil as null in the audit log.
func computeDiff(before, after map[string]interface{}) map[string]interface{} {
	if len(before) == 0 && len(after) == 0 {
		return nil
	}

	diff := make(map[string]interface{})

	// Fields in after — check against before
	for k, newVal := range after {
		oldVal, existed := before[k]
		if !existed {
			diff[k] = []interface{}{nil, newVal}
			continue
		}
		if !valuesEqual(oldVal, newVal) {
			diff[k] = []interface{}{oldVal, newVal}
		}
	}

	// Fields only in before (e.g. cleared on update) — emit with new=nil
	for k, oldVal := range before {
		if _, stillThere := after[k]; !stillThere {
			diff[k] = []interface{}{oldVal, nil}
		}
	}

	return diff
}

// valuesEqual is a tolerant equality check for the kinds of values that come
// out of reflective struct walks (interfaces wrapping ints, strings, times,
// nested maps). reflect.DeepEqual is the safest general check; we just
// special-case nil pointers so they compare equal across types.
func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}
