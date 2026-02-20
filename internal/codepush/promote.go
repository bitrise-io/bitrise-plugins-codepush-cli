package codepush

import (
	"context"
	"fmt"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Promote executes the promote workflow: validate, resolve both deployments,
// optionally resolve label to package ID, call API, export summary.
func Promote(ctx context.Context, client Client, opts *PromoteOptions, out *output.Writer) (*PromoteResult, error) {
	if err := validatePromoteOptions(opts); err != nil {
		return nil, err
	}

	sourceDeploymentID, err := ResolveDeployment(ctx, client, opts.AppID, opts.SourceDeploymentID, out)
	if err != nil {
		return nil, fmt.Errorf("resolving source deployment: %w", err)
	}

	destDeploymentID, err := ResolveDeployment(ctx, client, opts.AppID, opts.DestDeploymentID, out)
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
		packageID, err := resolvePackageLabel(ctx, client, opts.AppID, sourceDeploymentID, opts.Label, out)
		if err != nil {
			return nil, err
		}
		req.PackageID = packageID
	}

	out.Step("Promoting from %s to %s", opts.SourceDeploymentID, opts.DestDeploymentID)
	pkg, err := client.Promote(ctx, opts.AppID, sourceDeploymentID, req)
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
		exportSummary("codepush-promote-summary.json", result, out)
	}

	return result, nil
}

func validatePromoteOptions(opts *PromoteOptions) error {
	if err := validateBaseOptions(opts.AppID, opts.Token); err != nil {
		return err
	}
	if opts.SourceDeploymentID == "" {
		return fmt.Errorf("source deployment is required: set --source-deployment")
	}
	if opts.DestDeploymentID == "" {
		return fmt.Errorf("destination deployment is required: set --destination-deployment")
	}
	if opts.SourceDeploymentID == opts.DestDeploymentID {
		return fmt.Errorf("source and destination deployments must be different")
	}
	return nil
}
