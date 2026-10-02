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

// DeltaStatusPairs renders an update's delta generation state for out.Result:
// the state itself, then one line per predecessor that has a delta, in hash
// order. Servers that predate the field yield nothing.
func DeltaStatusPairs(status *codepush.UpdateStatus) []output.KeyValue {
	if status.DeltaGenerationStatus == "" {
		return nil
	}

	state := status.DeltaGenerationStatus
	if state == codepush.DeltaGenerationCompleted && len(status.Deltas) == 0 {
		state += " (no deltas, clients download the full package)"
	}
	pairs := []output.KeyValue{{Key: "Delta generation", Value: state}}

	for _, hash := range slices.Sorted(maps.Keys(status.Deltas)) {
		pairs = append(pairs, output.KeyValue{
			Key:   "Delta from " + Truncate(hash, 15),
			Value: describeDelta(status.Deltas[hash]),
		})
	}
	return pairs
}

func describeDelta(info codepush.DeltaInfo) string {
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
