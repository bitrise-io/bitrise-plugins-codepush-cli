package codepush

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	ziputil "github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/zip"
)

// Push executes the full push workflow: zip, upload, and poll for completion.
func Push(client Client, opts *PushOptions) (*PushResult, error) {
	return PushWithConfig(client, opts, DefaultPollConfig)
}

// PushWithConfig executes the push workflow with a configurable poll config.
func PushWithConfig(client Client, opts *PushOptions, pollCfg PollConfig) (*PushResult, error) {
	if err := validatePushOptions(opts); err != nil {
		return nil, err
	}

	deploymentID, err := ResolveDeployment(client, opts.AppID, opts.DeploymentID)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stderr, "Packaging bundle: %s\n", opts.BundlePath)
	zipPath, err := ziputil.Directory(opts.BundlePath)
	if err != nil {
		return nil, fmt.Errorf("packaging bundle: %w", err)
	}
	defer os.Remove(zipPath)

	zipInfo, err := os.Stat(zipPath)
	if err != nil {
		return nil, fmt.Errorf("reading zip file info: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Package size: %d bytes\n", zipInfo.Size())

	packageID := uuid.New().String()

	fmt.Fprintf(os.Stderr, "Requesting upload URL...\n")
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

	fmt.Fprintf(os.Stderr, "Uploading package...\n")
	zipFile, err := os.Open(zipPath)
	if err != nil {
		return nil, fmt.Errorf("opening zip for upload: %w", err)
	}
	defer zipFile.Close()

	if err := client.UploadFile(uploadResp.URL, uploadResp.Method, uploadResp.Headers, zipFile, zipInfo.Size()); err != nil {
		return nil, fmt.Errorf("uploading package: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Processing package...\n")
	status, err := pollStatus(client, opts.AppID, deploymentID, packageID, pollCfg)
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
	if opts.AppID == "" {
		return fmt.Errorf("app ID is required: set --app-id or CODEPUSH_APP_ID")
	}
	if opts.DeploymentID == "" {
		return fmt.Errorf("deployment is required: set --deployment or CODEPUSH_DEPLOYMENT")
	}
	if opts.Token == "" {
		return fmt.Errorf("API token is required: set --token, BITRISE_API_TOKEN, or run 'codepush auth login'")
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
		return fmt.Errorf("bundle path does not exist: %s", opts.BundlePath)
	}
	if !info.IsDir() {
		return fmt.Errorf("bundle path is not a directory: %s", opts.BundlePath)
	}

	return nil
}

// ResolveDeployment resolves a deployment name or UUID to a deployment ID.
// If the input is already a valid UUID, it is returned as-is.
// Otherwise, it lists all deployments and finds the one matching by name.
func ResolveDeployment(client Client, appID, deploymentNameOrID string) (string, error) {
	if _, err := uuid.Parse(deploymentNameOrID); err == nil {
		return deploymentNameOrID, nil
	}

	fmt.Fprintf(os.Stderr, "Resolving deployment %q...\n", deploymentNameOrID)
	deployments, err := client.ListDeployments(appID)
	if err != nil {
		return "", fmt.Errorf("listing deployments: %w", err)
	}

	for _, d := range deployments {
		if d.Name == deploymentNameOrID {
			fmt.Fprintf(os.Stderr, "Resolved deployment %q to ID %s\n", deploymentNameOrID, d.ID)
			return d.ID, nil
		}
	}

	return "", fmt.Errorf("deployment %q not found: check the deployment name or use a deployment UUID", deploymentNameOrID)
}

func pollStatus(client Client, appID, deploymentID, packageID string, cfg PollConfig) (*PackageStatus, error) {
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		status, err := client.GetPackageStatus(appID, deploymentID, packageID)
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

