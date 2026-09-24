package codepush

import (
	"slices"
	"time"
)

// NewestUpdates returns up to limit updates ordered newest first by CreatedAt.
// A limit of zero or less returns all updates.
//
// The list API's ordering is not something callers should rely on (it currently
// returns newest first), so anything that picks "the latest" or "the last N"
// releases must go through this. Updates with a missing or unparsable CreatedAt
// sort after dated ones; ties keep their original relative order.
func NewestUpdates(updates []Update, limit int) []Update {
	sorted := slices.Clone(updates)
	slices.SortStableFunc(sorted, func(a, b Update) int {
		ta, okA := parseCreatedAt(a.CreatedAt)
		tb, okB := parseCreatedAt(b.CreatedAt)
		switch {
		case okA && okB:
			return tb.Compare(ta)
		case okA:
			return -1
		case okB:
			return 1
		default:
			return 0
		}
	})

	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func parseCreatedAt(value string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, value)
	return t, err == nil
}
