package codepush

import (
	"fmt"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/bitrise"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Rollback executes the rollback workflow: validate, resolve deployment,
// optionally resolve target label to package ID, call API, export summary.
func Rollback(client Client, opts *RollbackOptions, out *output.Writer) (*RollbackResult, error) {
	if err := validateRollbackOptions(opts); err != nil {
		return nil, err
	}

	deploymentID, err := ResolveDeployment(client, opts.AppID, opts.DeploymentID, out)
	if err != nil {
		return nil, err
	}

	req := RollbackRequest{}

	if opts.TargetLabel != "" {
		packageID, err := resolvePackageLabel(client, opts.AppID, deploymentID, opts.TargetLabel, out)
		if err != nil {
			return nil, err
		}
		req.PackageID = packageID
	}

	out.Step("Rolling back deployment")
	pkg, err := client.Rollback(opts.AppID, deploymentID, req)
	if err != nil {
		return nil, fmt.Errorf("rollback failed: %w", err)
	}

	result := &RollbackResult{
		PackageID:    pkg.ID,
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
		return fmt.Errorf("deployment is required: set --deployment or CODEPUSH_DEPLOYMENT")
	}
	return nil
}

// resolvePackageLabel finds a package by its label (e.g. "v3") within a deployment.
func resolvePackageLabel(client Client, appID, deploymentID, label string, out *output.Writer) (string, error) {
	out.Step("Resolving release label %q", label)
	packages, err := client.ListPackages(appID, deploymentID)
	if err != nil {
		return "", fmt.Errorf("listing packages: %w", err)
	}

	for _, p := range packages {
		if p.Label == label {
			out.Info("Resolved to %s", p.ID)
			return p.ID, nil
		}
	}

	return "", fmt.Errorf("release label %q not found in deployment: check the label or omit --target-release to rollback to the previous release", label)
}

