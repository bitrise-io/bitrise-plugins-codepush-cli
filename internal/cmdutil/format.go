package cmdutil

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// OutputJSON marshals v as indented JSON to stdout. Used when --json is set.
func OutputJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON output: %w", err)
	}
	_, _ = fmt.Fprintln(os.Stdout, string(data))
	return nil
}

// Truncate shortens a string to max length, appending "..." if truncated.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// FormatBytes returns a human-readable byte size.
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return strconv.FormatInt(b, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
		if exp >= len("KMGTPE")-1 {
			break
		}
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// DiffStatusPairs renders an update's diff generation state for out.Result:
// the state itself, then one line per predecessor that has a diff, in hash
// order. Servers that predate the field yield nothing.
func DiffStatusPairs(status *codepush.UpdateStatus) []output.KeyValue {
	if status.DiffGenerationStatus == "" {
		return nil
	}

	state := status.DiffGenerationStatus
	if state == codepush.DiffGenerationCompleted && len(status.Diffs) == 0 {
		state += " (no diffs, clients download the full package)"
	}
	pairs := []output.KeyValue{{Key: "Diff generation", Value: state}}

	for _, hash := range slices.Sorted(maps.Keys(status.Diffs)) {
		pairs = append(pairs, output.KeyValue{
			Key:   "Diff from " + Truncate(hash, 15),
			Value: describeDiff(status.Diffs[hash]),
		})
	}
	return pairs
}

func describeDiff(info codepush.DiffInfo) string {
	kinds := make([]string, 0, 2)
	if info.V1 {
		kinds = append(kinds, "v1 (file-level)")
	}
	if info.V2 {
		kinds = append(kinds, "v2 (binary)")
	}
	if len(kinds) == 0 {
		return "none"
	}
	return strings.Join(kinds, ", ")
}
