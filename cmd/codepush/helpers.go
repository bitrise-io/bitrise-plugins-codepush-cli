package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/auth"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
)

// outputJSON marshals v as JSON to stdout. Used when --json is set.
func outputJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON output: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

// resolveFlag returns the flag value if non-empty, otherwise falls back to the environment variable.
func resolveFlag(flagValue, envKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envKey)
}

// resolveToken returns the API token using the priority:
// 1. BITRISE_API_TOKEN environment variable
// 2. Stored config file token (from 'codepush auth login')
func resolveToken() string {
	if envValue := os.Getenv("BITRISE_API_TOKEN"); envValue != "" {
		return envValue
	}
	storedToken, err := auth.LoadToken()
	if err != nil {
		if out != nil {
			out.Warning("could not load stored token: %v", err)
		}
	}
	return storedToken
}

// requireCredentials resolves and validates the app ID and API token.
func requireCredentials() (appID, token string, err error) {
	appID = resolveFlag(globalAppID, "CODEPUSH_APP_ID")
	token = resolveToken()

	if appID == "" {
		return "", "", fmt.Errorf("app ID is required: set --app-id or CODEPUSH_APP_ID")
	}
	if token == "" {
		return "", "", fmt.Errorf("API token is required: set BITRISE_API_TOKEN or run 'codepush auth login'")
	}
	return appID, token, nil
}

// exportDeploySummary writes a JSON summary to the Bitrise deploy directory.
func exportDeploySummary(filename string, v interface{}) {
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

// exportEnvVars exports key-value pairs as Bitrise environment variables via envman.
func exportEnvVars(vars map[string]string) {
	for key, value := range vars {
		if err := bitrise.ExportEnvVar(key, value); err != nil {
			out.Warning("failed to export %s: %v", key, err)
		}
	}
}

// truncate shortens a string to max length, appending "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// formatBytes returns a human-readable byte size.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return strconv.FormatInt(b, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
