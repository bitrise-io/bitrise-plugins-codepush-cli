package cmdutil

import (
	"encoding/json"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// ExportDeploySummary writes a JSON summary to the Bitrise deploy directory.
func ExportDeploySummary(filename string, v interface{}, out *output.Writer) {
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

// ExportEnvVars exports key-value pairs as Bitrise environment variables via envman.
func ExportEnvVars(vars map[string]string, out *output.Writer) {
	for key, value := range vars {
		if err := bitrise.ExportEnvVar(key, value); err != nil {
			out.Warning("failed to export %s: %v", key, err)
		}
	}
}
