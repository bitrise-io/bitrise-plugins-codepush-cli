// Package codepush contains the core logic for CodePush OTA update management.
package codepush

import (
	"encoding/json"
	"fmt"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// validateBaseOptions checks the common required fields shared by all operations.
func validateBaseOptions(appID, token string) error {
	if appID == "" {
		return fmt.Errorf("app ID is required: set --app-id or CODEPUSH_APP_ID")
	}
	if token == "" {
		return fmt.Errorf("API token is required: set --token, BITRISE_API_TOKEN, or run 'codepush auth login'")
	}
	return nil
}

// exportSummary marshals v as JSON and writes it to the Bitrise deploy directory.
func exportSummary(filename string, v interface{}, out *output.Writer) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		out.Warning("failed to marshal %s: %v", filename, err)
		return
	}

	path, err := bitrise.WriteToDeployDir(filename, data)
	if err != nil {
		out.Warning("failed to export %s: %v", filename, err)
		return
	}

	out.Info("Summary exported to: %s", path)
}
