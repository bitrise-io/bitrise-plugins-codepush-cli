package codepush

import (
	"context"
	"errors"
	"fmt"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Rollback executes the rollback workflow: validate, resolve deployment,
// optionally resolve target label to update ID, call API, export summary.
func Rollback(ctx context.Context, client Client, opts *RollbackOptions, out *output.Writer) (*RollbackResult, error) {
	if err := validateRollbackOptions(opts); err != nil {
		return nil, err
	}

	deploymentID, err := ResolveDeployment(ctx, client, opts.AppID, opts.DeploymentID, out)
	if err != nil {
		return nil, err
	}

	req := RollbackRequest{}

	if opts.TargetLabel != "" {
		updateID, err := resolveUpdateLabel(ctx, client, opts.AppID, deploymentID, opts.TargetLabel, out)
		if err != nil {
			return nil, err
		}
		req.UpdateID = updateID
	}

	out.Step("Rolling back deployment")
	pkg, err := client.Rollback(ctx, opts.AppID, deploymentID, req)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	result := &RollbackResult{
		UpdateID:     pkg.ID,
		AppID:        opts.AppID,
		DeploymentID: deploymentID,
		Label:        pkg.Label,
		AppVersion:   pkg.AppVersion,
	}

	if bitrise.IsBitriseEnvironment() {
		exportSummary("codepush-rollback-summary.json", result, out)
	}

	return result, nil
}

func validateRollbackOptions(opts *RollbackOptions) error {
	if err := validateBaseOptions(opts.AppID, opts.Token); err != nil {
		return err
	}
	if opts.DeploymentID == "" {
		return errors.New("deployment is required: set --deployment or CODEPUSH_DEPLOYMENT")
	}
	return nil
}

// updateLister is the subset of Client needed by resolveUpdateLabel.
type updateLister interface {
	ListUpdates(ctx context.Context, appID, deploymentID string) ([]Update, error)
}

// resolveUpdateLabel finds an update by its label (e.g. "v3") within a deployment.
func resolveUpdateLabel(ctx context.Context, client updateLister, appID, deploymentID, label string, out *output.Writer) (string, error) {
	out.Step("Resolving release label %q", label)
	updates, err := client.ListUpdates(ctx, appID, deploymentID)
	if err != nil {
		return "", fmt.Errorf("listing updates: %w", err)
	}

	for _, u := range updates {
		if u.Label == label {
			out.Info("Resolved to %s", u.ID)
			return u.ID, nil
		}
	}

	return "", fmt.Errorf("release label %q not found in deployment: check the label or omit --target-release to rollback to the previous release", label)
}
