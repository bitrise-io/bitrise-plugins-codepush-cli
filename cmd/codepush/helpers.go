package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/google/uuid"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/auth"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/config"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
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

// resolveAppID returns the app ID using the priority:
// 1. --app-id flag
// 2. CODEPUSH_APP_ID environment variable
// 3. .codepush.json file in current directory
func resolveAppID() string {
	if globalAppID != "" {
		return globalAppID
	}
	if envValue := os.Getenv("CODEPUSH_APP_ID"); envValue != "" {
		return envValue
	}
	cfg, err := config.Load()
	if err != nil {
		if out != nil {
			out.Warning("could not load %s: %v", config.FileName, err)
		}
		return ""
	}
	if cfg != nil {
		return cfg.AppID
	}
	return ""
}

// requireCredentials resolves and validates the app ID and API token.
func requireCredentials() (appID, token string, err error) {
	appID = resolveAppID()
	token = resolveToken()

	if appID == "" {
		return "", "", fmt.Errorf("app ID is required: set --app-id, CODEPUSH_APP_ID, or run 'codepush init'")
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

// resolveAppIDInteractive resolves the app ID using the priority:
// 1. --app-id flag
// 2. CODEPUSH_APP_ID environment variable
// 3. .codepush.json config file
// 4. Interactive terminal input prompt
// 5. Non-interactive error with flag hint
func resolveAppIDInteractive() (string, error) {
	appID := resolveAppID()
	if appID != "" {
		if _, err := uuid.Parse(appID); err != nil {
			return "", fmt.Errorf("invalid app ID %q: must be a valid UUID", appID)
		}
		return appID, nil
	}

	if !out.IsInteractive() {
		return "", fmt.Errorf("app ID is required: set --app-id, CODEPUSH_APP_ID, or run 'codepush init'")
	}

	appID, err := out.Input("Enter your app ID (UUID)", "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	if err != nil {
		return "", err
	}

	if appID == "" {
		return "", fmt.Errorf("app ID is required")
	}

	if _, err := uuid.Parse(appID); err != nil {
		return "", fmt.Errorf("invalid app ID %q: must be a valid UUID", appID)
	}

	return appID, nil
}

// resolveDeploymentInteractive resolves a deployment using the priority:
// 1. Flag value (passed directly)
// 2. Environment variable
// 3. Interactive terminal selector (fetches deployments from API)
// 4. Non-interactive error with flag hint
func resolveDeploymentInteractive(ctx context.Context, client codepush.Client, appID, flagValue, envKey string) (string, error) {
	deployment := resolveFlag(flagValue, envKey)

	if deployment != "" {
		return codepush.ResolveDeployment(ctx, client, appID, deployment, out)
	}

	if !out.IsInteractive() {
		if envKey != "" {
			return "", fmt.Errorf("deployment is required: set --deployment or %s", envKey)
		}
		return "", fmt.Errorf("deployment is required: provide a deployment name or UUID")
	}

	deployments, err := client.ListDeployments(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("listing deployments: %w", err)
	}

	if len(deployments) == 0 {
		return "", fmt.Errorf("no deployments found: create one with 'codepush deployment add'")
	}

	options := make([]output.SelectOption, len(deployments))
	for i, d := range deployments {
		options[i] = output.SelectOption{Label: d.Name, Value: d.ID}
	}

	return out.Select("Select deployment", options)
}

// resolvePlatformInteractive resolves the platform flag interactively.
// If the flag value is set, returns it. Otherwise prompts if interactive
// or returns an error with a flag hint.
func resolvePlatformInteractive(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if !out.IsInteractive() {
		return "", fmt.Errorf("--platform is required: set --platform to ios or android")
	}

	return out.Select("Select platform", []output.SelectOption{
		{Label: "iOS", Value: "ios"},
		{Label: "Android", Value: "android"},
	})
}
