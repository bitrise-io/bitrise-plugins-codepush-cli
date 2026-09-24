package codepush

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func labels(updates []Update) []string {
	out := make([]string, len(updates))
	for i, u := range updates {
		out[i] = u.Label
	}
	return out
}

func TestNewestUpdates(t *testing.T) {
	// Newest first, the order the list API returns today.
	apiOrder := []Update{
		{Label: "v3", CreatedAt: "2026-09-24T10:42:34.000Z"},
		{Label: "v2", CreatedAt: "2026-09-24T10:20:18.000Z"},
		{Label: "v1", CreatedAt: "2026-07-30T11:45:38.000Z"},
	}

	t.Run("keeps newest-first input", func(t *testing.T) {
		assert.Equal(t, []string{"v3", "v2", "v1"}, labels(NewestUpdates(apiOrder, 0)))
	})

	t.Run("sorts oldest-first input", func(t *testing.T) {
		in := []Update{apiOrder[2], apiOrder[1], apiOrder[0]}
		assert.Equal(t, []string{"v3", "v2", "v1"}, labels(NewestUpdates(in, 0)))
	})

	t.Run("sorts shuffled input", func(t *testing.T) {
		in := []Update{apiOrder[1], apiOrder[2], apiOrder[0]}
		assert.Equal(t, []string{"v3", "v2", "v1"}, labels(NewestUpdates(in, 0)))
	})

	t.Run("limit keeps the newest", func(t *testing.T) {
		assert.Equal(t, []string{"v3", "v2"}, labels(NewestUpdates(apiOrder, 2)))
	})

	t.Run("limit larger than list returns all", func(t *testing.T) {
		assert.Len(t, NewestUpdates(apiOrder, 10), 3)
	})

	t.Run("does not modify the input", func(t *testing.T) {
		in := []Update{apiOrder[2], apiOrder[1], apiOrder[0]}
		NewestUpdates(in, 0)
		assert.Equal(t, []string{"v1", "v2", "v3"}, labels(in))
	})

	t.Run("undated updates sort last in original order", func(t *testing.T) {
		in := []Update{
			{Label: "undated-a"},
			{Label: "v1", CreatedAt: "2026-07-30T11:45:38.000Z"},
			{Label: "undated-b", CreatedAt: "not a timestamp"},
			{Label: "v2", CreatedAt: "2026-09-24T10:20:18.000Z"},
		}
		assert.Equal(t, []string{"v2", "v1", "undated-a", "undated-b"}, labels(NewestUpdates(in, 0)))
	})

	t.Run("empty input", func(t *testing.T) {
		assert.Empty(t, NewestUpdates(nil, 1))
	})
}
