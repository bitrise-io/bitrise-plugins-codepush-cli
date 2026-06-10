package codepush

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Promote executes the promote workflow: validate, resolve both deployments,
// optionally resolve label to update ID, call API, export summary.
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
	}
	if opts.Rollout != "" {
		v, err := strconv.ParseFloat(opts.Rollout, 64)
		if err != nil || v < 0 || v > 100 {
			return nil, fmt.Errorf("rollout must be between 0 and 100, got %q", opts.Rollout)
		}
		req.Rollout = opts.Rollout
	}

	if opts.Label != "" {
		updateID, err := resolveUpdateLabel(ctx, client, sourceDeploymentID, opts.Label, out)
		if err != nil {
			return nil, err
		}
		req.PackageID = updateID
	}

	step := out.StartStep("Promoting from %s to %s", opts.SourceDeploymentID, opts.DestDeploymentID)
	pkg, err := client.Promote(ctx, sourceDeploymentID, req)
	if err != nil {
		step.Cancel()
		return nil, fmt.Errorf("promote failed: %w", err)
	}
	step.Done()

	result := &PromoteResult{
		UpdateID:         pkg.ID,
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
		return errors.New("source deployment is required: set --source-deployment")
	}
	if opts.DestDeploymentID == "" {
		return errors.New("destination deployment is required: set --destination-deployment")
	}
	if opts.SourceDeploymentID == opts.DestDeploymentID {
		return errors.New("source and destination deployments must be different")
	}
	return nil
}
