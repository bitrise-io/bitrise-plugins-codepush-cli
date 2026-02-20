package codepush

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
	ziputil "github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/zip"
)

// Push executes the full push workflow: zip, upload, and poll for completion.
func Push(client Client, opts *PushOptions, out *output.Writer) (*PushResult, error) {
	return PushWithConfig(client, opts, DefaultPollConfig, out)
}

// PushWithConfig executes the push workflow with a configurable poll config.
func PushWithConfig(client Client, opts *PushOptions, pollCfg PollConfig, out *output.Writer) (*PushResult, error) {
	if err := validatePushOptions(opts); err != nil {
		return nil, err
	}

	deploymentID, err := ResolveDeployment(client, opts.AppID, opts.DeploymentID, out)
	if err != nil {
		return nil, err
	}

	out.Step("Packaging bundle: %s", opts.BundlePath)
	zipPath, err := ziputil.Directory(opts.BundlePath)
	if err != nil {
		return nil, fmt.Errorf("packaging bundle: %w", err)
	}
	defer os.Remove(zipPath)

	zipInfo, err := os.Stat(zipPath)
	if err != nil {
		return nil, fmt.Errorf("reading zip file info: %w", err)
	}
	out.Info("Package size: %d bytes", zipInfo.Size())

	packageID := uuid.New().String()

	out.Step("Requesting upload URL")
	uploadResp, err := client.GetUploadURL(opts.AppID, deploymentID, packageID, UploadURLRequest{
		AppVersion:    opts.AppVersion,
		FileName:      filepath.Base(zipPath),
		FileSizeBytes: zipInfo.Size(),
		Description:   opts.Description,
		Mandatory:     opts.Mandatory,
		Disabled:      opts.Disabled,
		Rollout:       opts.Rollout,
	})
	if err != nil {
		return nil, fmt.Errorf("requesting upload URL: %w", err)
	}

	err = out.Spinner("Uploading package", func() error {
		zipFile, openErr := os.Open(zipPath)
		if openErr != nil {
			return fmt.Errorf("opening zip for upload: %w", openErr)
		}
		defer zipFile.Close()

		return client.UploadFile(UploadFileRequest{
			URL:           uploadResp.URL,
			Method:        uploadResp.Method,
			Headers:       uploadResp.Headers,
			Body:          zipFile,
			ContentLength: zipInfo.Size(),
		})
	})
	if err != nil {
		return nil, fmt.Errorf("uploading package: %w", err)
	}

	var status *PackageStatus
	err = out.Spinner("Processing package", func() error {
		var pollErr error
		status, pollErr = pollStatus(client, PackageRef{AppID: opts.AppID, DeploymentID: deploymentID, PackageID: packageID}, pollCfg)
		return pollErr
	})
	if err != nil {
		return nil, err
	}

	result := &PushResult{
		PackageID:     packageID,
		AppID:         opts.AppID,
		DeploymentID:  deploymentID,
		AppVersion:    opts.AppVersion,
		Status:        status.Status,
		FileSizeBytes: zipInfo.Size(),
	}

	return result, nil
}

func validatePushOptions(opts *PushOptions) error {
	if err := validateBaseOptions(opts.AppID, opts.Token); err != nil {
		return err
	}
	if opts.DeploymentID == "" {
		return fmt.Errorf("deployment is required: set --deployment or CODEPUSH_DEPLOYMENT")
	}
	if opts.AppVersion == "" {
		return fmt.Errorf("app version is required: set --app-version")
	}
	if opts.BundlePath == "" {
		return fmt.Errorf("bundle path is required: provide as argument or use --bundle")
	}
	if opts.Rollout < 1 || opts.Rollout > 100 {
		return fmt.Errorf("rollout must be between 1 and 100, got %d", opts.Rollout)
	}

	info, err := os.Stat(opts.BundlePath)
	if err != nil {
		return fmt.Errorf("bundle path does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("bundle path is not a directory: %s", opts.BundlePath)
	}

	return nil
}

// ResolveDeployment resolves a deployment name or UUID to a deployment ID.
// If the input is already a valid UUID, it is returned as-is.
// Otherwise, it lists all deployments and finds the one matching by name.
func ResolveDeployment(client Client, appID, deploymentNameOrID string, out *output.Writer) (string, error) {
	if _, err := uuid.Parse(deploymentNameOrID); err == nil {
		return deploymentNameOrID, nil
	}

	out.Step("Resolving deployment %q", deploymentNameOrID)
	deployments, err := client.ListDeployments(appID)
	if err != nil {
		return "", fmt.Errorf("listing deployments: %w", err)
	}

	for _, d := range deployments {
		if d.Name == deploymentNameOrID {
			out.Info("Resolved to %s", d.ID)
			return d.ID, nil
		}
	}

	return "", fmt.Errorf("deployment %q not found: check the deployment name or use a deployment UUID", deploymentNameOrID)
}

func pollStatus(client Client, ref PackageRef, cfg PollConfig) (*PackageStatus, error) {
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		status, err := client.GetPackageStatus(ref.AppID, ref.DeploymentID, ref.PackageID)
		if err != nil {
			return nil, fmt.Errorf("checking package status: %w", err)
		}

		switch status.Status {
		case StatusDone:
			return status, nil
		case StatusFailed:
			return nil, fmt.Errorf("package processing failed: %s", status.StatusReason)
		}

		if attempt < cfg.MaxAttempts-1 {
			time.Sleep(cfg.Interval)
		}
	}

	totalWait := time.Duration(cfg.MaxAttempts) * cfg.Interval
	return nil, fmt.Errorf("package processing timed out after %s", totalWait)
}
