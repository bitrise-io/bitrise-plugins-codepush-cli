package codepush

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Patch executes the patch workflow: validate, resolve deployment,
// resolve label (or find latest), build request, call API, export summary.
func Patch(client Client, opts *PatchOptions, out *output.Writer) (*PatchResult, error) {
	if err := validatePatchOptions(opts); err != nil {
		return nil, err
	}

	deploymentID, err := ResolveDeployment(client, opts.AppID, opts.DeploymentID, out)
	if err != nil {
		return nil, err
	}

	packageID, packageLabel, err := ResolvePackageForPatch(client, opts.AppID, deploymentID, opts.Label, out)
	if err != nil {
		return nil, err
	}

	req, err := buildPatchRequest(opts)
	if err != nil {
		return nil, err
	}

	out.Step("Patching release %s", packageLabel)
	pkg, err := client.PatchPackage(opts.AppID, deploymentID, packageID, req)
	if err != nil {
		return nil, fmt.Errorf("patch failed: %w", err)
	}

	result := &PatchResult{
		PackageID:    pkg.ID,
		AppID:        opts.AppID,
		DeploymentID: deploymentID,
		Label:        pkg.Label,
		AppVersion:   pkg.AppVersion,
		Mandatory:    pkg.Mandatory,
		Disabled:     pkg.Disabled,
		Rollout:      pkg.Rollout,
		Description:  pkg.Description,
	}

	if bitrise.IsBitriseEnvironment() {
		exportPatchSummary(result, out)
	}

	return result, nil
}

func validatePatchOptions(opts *PatchOptions) error {
	if opts.AppID == "" {
		return fmt.Errorf("app ID is required: set --app-id or CODEPUSH_APP_ID")
	}
	if opts.DeploymentID == "" {
		return fmt.Errorf("deployment is required: set --deployment or CODEPUSH_DEPLOYMENT")
	}
	if opts.Token == "" {
		return fmt.Errorf("API token is required: set --token, BITRISE_API_TOKEN, or run 'codepush auth login'")
	}
	if opts.Rollout == "" && opts.Mandatory == "" && opts.Disabled == "" && opts.Description == "" && opts.AppVersion == "" {
		return fmt.Errorf("at least one change is required: set --rollout, --mandatory, --disabled, --description, or --app-version")
	}
	return nil
}

// ResolvePackageForPatch resolves a package by label or finds the latest package.
// Returns the package ID and label.
func ResolvePackageForPatch(client Client, appID, deploymentID, label string, out *output.Writer) (string, string, error) {
	if label != "" {
		id, err := resolvePackageLabel(client, appID, deploymentID, label, out)
		if err != nil {
			return "", "", err
		}
		return id, label, nil
	}

	out.Step("Resolving latest release")
	packages, err := client.ListPackages(appID, deploymentID)
	if err != nil {
		return "", "", fmt.Errorf("listing packages: %w", err)
	}

	if len(packages) == 0 {
		return "", "", fmt.Errorf("no releases found in deployment: push a release first")
	}

	latest := packages[len(packages)-1]
	out.Info("Resolved latest release: %s (%s)", latest.Label, latest.ID)
	return latest.ID, latest.Label, nil
}

func buildPatchRequest(opts *PatchOptions) (PatchRequest, error) {
	var req PatchRequest

	if opts.Rollout != "" {
		v, err := strconv.Atoi(opts.Rollout)
		if err != nil || v < 1 || v > 100 {
			return req, fmt.Errorf("rollout must be between 1 and 100, got %q", opts.Rollout)
		}
		req.Rollout = &v
	}

	if opts.Mandatory != "" {
		v, err := strconv.ParseBool(opts.Mandatory)
		if err != nil {
			return req, fmt.Errorf("mandatory must be true or false, got %q", opts.Mandatory)
		}
		req.Mandatory = &v
	}

	if opts.Disabled != "" {
		v, err := strconv.ParseBool(opts.Disabled)
		if err != nil {
			return req, fmt.Errorf("disabled must be true or false, got %q", opts.Disabled)
		}
		req.Disabled = &v
	}

	if opts.Description != "" {
		req.Description = &opts.Description
	}

	if opts.AppVersion != "" {
		req.AppVersion = &opts.AppVersion
	}

	return req, nil
}

type patchSummary struct {
	PackageID    string `json:"package_id"`
	AppID        string `json:"app_id"`
	DeploymentID string `json:"deployment_id"`
	Label        string `json:"label"`
	AppVersion   string `json:"app_version"`
	Mandatory    bool   `json:"mandatory"`
	Disabled     bool   `json:"disabled"`
	Rollout      int    `json:"rollout"`
	Description  string `json:"description"`
}

func exportPatchSummary(result *PatchResult, out *output.Writer) {
	summary := patchSummary{
		PackageID:    result.PackageID,
		AppID:        result.AppID,
		DeploymentID: result.DeploymentID,
		Label:        result.Label,
		AppVersion:   result.AppVersion,
		Mandatory:    result.Mandatory,
		Disabled:     result.Disabled,
		Rollout:      result.Rollout,
		Description:  result.Description,
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		out.Warning("failed to marshal patch summary: %v", err)
		return
	}

	path, err := bitrise.WriteToDeployDir("codepush-patch-summary.json", data)
	if err != nil {
		out.Warning("failed to export patch summary: %v", err)
		return
	}

	out.Info("Patch summary exported to: %s", path)
}
