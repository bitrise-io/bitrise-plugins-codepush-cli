package codepush

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
)

// Promote executes the promote workflow: validate, resolve both deployments,
// optionally resolve label to package ID, call API, export summary.
func Promote(client Client, opts *PromoteOptions) (*PromoteResult, error) {
	if err := validatePromoteOptions(opts); err != nil {
		return nil, err
	}

	sourceDeploymentID, err := resolveDeployment(client, opts.AppID, opts.SourceDeploymentID)
	if err != nil {
		return nil, fmt.Errorf("resolving source deployment: %w", err)
	}

	destDeploymentID, err := resolveDeployment(client, opts.AppID, opts.DestDeploymentID)
	if err != nil {
		return nil, fmt.Errorf("resolving destination deployment: %w", err)
	}

	req := PromoteRequest{
		TargetDeploymentID: destDeploymentID,
		AppVersion:         opts.AppVersion,
		Description:        opts.Description,
		Mandatory:          opts.Mandatory,
		Disabled:           opts.Disabled,
		Rollout:            opts.Rollout,
	}

	if opts.Label != "" {
		packageID, err := resolvePackageLabel(client, opts.AppID, sourceDeploymentID, opts.Label)
		if err != nil {
			return nil, err
		}
		req.PackageID = packageID
	}

	fmt.Fprintf(os.Stderr, "Promoting from %s to %s...\n", opts.SourceDeploymentID, opts.DestDeploymentID)
	pkg, err := client.Promote(opts.AppID, sourceDeploymentID, req)
	if err != nil {
		return nil, fmt.Errorf("promote failed: %w", err)
	}

	result := &PromoteResult{
		PackageID:        pkg.ID,
		AppID:            opts.AppID,
		SourceDeployment: sourceDeploymentID,
		DestDeployment:   destDeploymentID,
		Label:            pkg.Label,
		AppVersion:       pkg.AppVersion,
		Description:      pkg.Description,
	}

	if bitrise.IsBitriseEnvironment() {
		exportPromoteSummary(result)
	}

	return result, nil
}

func validatePromoteOptions(opts *PromoteOptions) error {
	if opts.AppID == "" {
		return fmt.Errorf("app ID is required: set --app-id or CODEPUSH_APP_ID")
	}
	if opts.SourceDeploymentID == "" {
		return fmt.Errorf("source deployment is required: set --source-deployment")
	}
	if opts.DestDeploymentID == "" {
		return fmt.Errorf("destination deployment is required: set --destination-deployment")
	}
	if opts.Token == "" {
		return fmt.Errorf("API token is required: set --token, BITRISE_API_TOKEN, or run 'codepush auth login'")
	}
	if opts.SourceDeploymentID == opts.DestDeploymentID {
		return fmt.Errorf("source and destination deployments must be different")
	}
	return nil
}

type promoteSummary struct {
	PackageID        string `json:"package_id"`
	AppID            string `json:"app_id"`
	SourceDeployment string `json:"source_deployment_id"`
	DestDeployment   string `json:"dest_deployment_id"`
	Label            string `json:"label"`
	AppVersion       string `json:"app_version"`
	Description      string `json:"description"`
}

func exportPromoteSummary(result *PromoteResult) {
	summary := promoteSummary{
		PackageID:        result.PackageID,
		AppID:            result.AppID,
		SourceDeployment: result.SourceDeployment,
		DestDeployment:   result.DestDeployment,
		Label:            result.Label,
		AppVersion:       result.AppVersion,
		Description:      result.Description,
	}

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal promote summary: %v\n", err)
		return
	}

	path, err := bitrise.WriteToDeployDir("codepush-promote-summary.json", data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to export promote summary: %v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Promote summary exported to: %s\n", path)
}
