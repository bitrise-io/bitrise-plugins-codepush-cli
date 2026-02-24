package cmdutil

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/auth"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/config"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// ResolveFlag returns flagValue if non-empty, otherwise falls back to the environment variable.
func ResolveFlag(flagValue, envKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envKey)
}

// ResolveToken returns the API token using the priority:
// 1. BITRISE_API_TOKEN environment variable
// 2. Stored config file token (from 'codepush auth login')
func ResolveToken(out *output.Writer) string {
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

// ResolveAppID returns the app ID using the priority:
// 1. globalAppID flag value
// 2. CODEPUSH_APP_ID environment variable
// 3. .codepush.json file in current directory
func ResolveAppID(globalAppID string, out *output.Writer) string {
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

// RequireCredentials resolves and validates the app ID and API token.
func RequireCredentials(globalAppID string, out *output.Writer) (appID, token string, err error) {
	appID = ResolveAppID(globalAppID, out)
	token = ResolveToken(out)

	if appID == "" {
		return "", "", fmt.Errorf("app ID is required: set --app-id, CODEPUSH_APP_ID, or run 'codepush init'")
	}
	if token == "" {
		return "", "", fmt.Errorf("API token is required: set BITRISE_API_TOKEN or run 'codepush auth login'")
	}
	return appID, token, nil
}

// ResolveInputInteractive returns the value if non-empty, otherwise prompts
// interactively. In non-interactive mode it returns an error with a hint.
func ResolveInputInteractive(value, title, placeholder string, out *output.Writer) (string, error) {
	if value != "" {
		return value, nil
	}

	if !out.IsInteractive() {
		return "", fmt.Errorf("%s: required in non-interactive mode", title)
	}

	result, err := out.Input(title, placeholder)
	if err != nil {
		return "", err
	}

	if result == "" {
		return "", fmt.Errorf("value is required")
	}

	return result, nil
}

// ResolveAppIDInteractive resolves the app ID using the priority:
// 1. --app-id flag
// 2. CODEPUSH_APP_ID environment variable
// 3. .codepush.json config file
// 4. Interactive terminal input prompt
// 5. Non-interactive error with flag hint
func ResolveAppIDInteractive(globalAppID string, out *output.Writer) (string, error) {
	appID := ResolveAppID(globalAppID, out)
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

// ResolveDeploymentInteractive resolves a deployment using the priority:
// 1. Flag value (passed directly)
// 2. Environment variable
// 3. Interactive terminal selector (fetches deployments from API)
// 4. Non-interactive error with flag hint
func ResolveDeploymentInteractive(ctx context.Context, client codepush.Client, appID, flagValue, envKey string, out *output.Writer) (string, error) {
	deployment := ResolveFlag(flagValue, envKey)

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

// ResolvePlatformInteractive resolves the platform flag interactively.
// If the flag value is set, returns it. Otherwise prompts if interactive
// or returns an error with a flag hint.
func ResolvePlatformInteractive(flagValue string, out *output.Writer) (string, error) {
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
