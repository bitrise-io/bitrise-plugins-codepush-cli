package codepush

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
	ziputil "github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/zip"
)

// Push executes the full push workflow: zip, upload, and poll for completion.
func Push(ctx context.Context, client Client, opts *PushOptions, out *output.Writer) (*PushResult, error) {
	return PushWithConfig(ctx, client, opts, DefaultPollConfig, out)
}

// PushWithConfig executes the push workflow with a configurable poll config.
func PushWithConfig(ctx context.Context, client Client, opts *PushOptions, pollCfg PollConfig, out *output.Writer) (*PushResult, error) {
	if err := validatePushOptions(opts); err != nil {
		return nil, err
	}

	deploymentID, err := ResolveDeployment(ctx, client, opts.AppID, opts.DeploymentID, out)
	if err != nil {
		return nil, err
	}

	updateID, fileSizeBytes, err := uploadBundle(ctx, client, opts, deploymentID, out)
	if err != nil {
		return nil, err
	}

	var status *UpdateStatus
	err = out.Indeterminate("Processing update", func() error {
		var pollErr error
		status, pollErr = pollStatus(ctx, client, UpdateRef{AppID: opts.AppID, DeploymentID: deploymentID, UpdateID: updateID}, pollCfg)
		return pollErr
	})
	if err != nil {
		return nil, err
	}

	return &PushResult{
		UpdateID:      updateID,
		AppID:         opts.AppID,
		DeploymentID:  deploymentID,
		AppVersion:    opts.AppVersion,
		Status:        status.Status,
		FileSizeBytes: fileSizeBytes,
		Rollout:       opts.Rollout,
	}, nil
}

func uploadBundle(ctx context.Context, client Client, opts *PushOptions, deploymentID string, out *output.Writer) (string, int64, error) {
	step := out.StartStep("Packaging bundle: %s", opts.BundlePath)
	zipPath, err := ziputil.Directory(opts.BundlePath)
	if err != nil {
		step.Cancel()
		return "", 0, fmt.Errorf("packaging bundle: %w", err)
	}
	defer func() { _ = os.Remove(zipPath) }()

	zipInfo, err := os.Stat(zipPath)
	if err != nil {
		return "", 0, fmt.Errorf("reading zip file info: %w", err)
	}
	step.Done()
	out.Info("Update size: %s", output.HumanBytes(zipInfo.Size()))

	updateID := uuid.New().String()

	stepURL := out.StartStep("Requesting upload URL")
	uploadResp, err := client.GetUploadURL(ctx, opts.AppID, deploymentID, updateID, UploadURLRequest{
		AppVersion:    opts.AppVersion,
		FileName:      filepath.Base(zipPath),
		FileSizeBytes: zipInfo.Size(),
		Description:   opts.Description,
		Mandatory:     opts.Mandatory,
		Disabled:      opts.Disabled,
		Rollout:       opts.Rollout,
	})
	if err != nil {
		stepURL.Cancel()
		return "", 0, fmt.Errorf("requesting upload URL: %w", err)
	}
	stepURL.Done()

	zipFile, err := os.Open(zipPath)
	if err != nil {
		return "", 0, fmt.Errorf("opening zip for upload: %w", err)
	}
	defer func() { _ = zipFile.Close() }()

	progress := out.NewProgress("Uploading")
	pr := output.NewProgressReader(zipFile, zipInfo.Size(), progress)
	uploadErr := client.UploadFile(ctx, UploadFileRequest{
		URL:           uploadResp.URL,
		Method:        uploadResp.Method,
		Headers:       uploadResp.Headers,
		Body:          pr,
		ContentLength: zipInfo.Size(),
	})
	if uploadErr != nil {
		progress.Cancel()
		return "", 0, fmt.Errorf("uploading update: %w", uploadErr)
	}
	progress.Done(output.HumanBytes(zipInfo.Size()))

	return updateID, zipInfo.Size(), nil
}

func validatePushOptions(opts *PushOptions) error {
	if err := validateBaseOptions(opts.AppID, opts.Token); err != nil {
		return err
	}
	if opts.DeploymentID == "" {
		return errors.New("deployment is required: set --deployment or CODEPUSH_DEPLOYMENT")
	}
	if opts.AppVersion == "" {
		return errors.New("app version is required: set --app-version")
	}
	if opts.BundlePath == "" {
		return errors.New("bundle path is required: provide as argument or use --bundle")
	}
	if opts.Rollout < 0 || opts.Rollout > 100 {
		return fmt.Errorf("rollout must be between 0 and 100, got %d", opts.Rollout)
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

// deploymentLister is the subset of Client needed by ResolveDeployment.
type deploymentLister interface {
	ListDeployments(ctx context.Context, appID string) ([]Deployment, error)
}

// ResolveDeployment resolves a deployment name or UUID to a deployment ID.
// If the input is already a valid UUID, it is returned as-is.
// Otherwise, it lists all deployments and finds the one matching by name.
func ResolveDeployment(ctx context.Context, client deploymentLister, appID, deploymentNameOrID string, out *output.Writer) (string, error) {
	if _, err := uuid.Parse(deploymentNameOrID); err == nil {
		return deploymentNameOrID, nil
	}

	step := out.StartStep("Resolving deployment %q", deploymentNameOrID)
	deployments, err := client.ListDeployments(ctx, appID)
	if err != nil {
		step.Cancel()
		return "", fmt.Errorf("listing deployments: %w", err)
	}

	for _, d := range deployments {
		if d.Name == deploymentNameOrID {
			step.Done()
			out.Info("Resolved to %s", d.ID)
			return d.ID, nil
		}
	}

	step.Cancel()
	return "", fmt.Errorf("deployment %q not found: check the deployment name or use a deployment UUID", deploymentNameOrID)
}

// statusChecker is the subset of Client needed by pollStatus.
type statusChecker interface {
	GetUpdateStatus(ctx context.Context, appID, deploymentID, updateID string) (*UpdateStatus, error)
}

func pollStatus(ctx context.Context, client statusChecker, ref UpdateRef, cfg PollConfig) (*UpdateStatus, error) {
	for attempt := range cfg.MaxAttempts {
		status, err := client.GetUpdateStatus(ctx, ref.AppID, ref.DeploymentID, ref.UpdateID)
		if err != nil {
			return nil, fmt.Errorf("checking update status: %w", err)
		}

		switch status.Status {
		case StatusProcessedValid:
			return status, nil
		case StatusProcessedError:
			return nil, fmt.Errorf("update processing failed: %s", status.StatusReason)
		}

		if attempt < cfg.MaxAttempts-1 {
			time.Sleep(cfg.Interval)
		}
	}

	totalWait := time.Duration(cfg.MaxAttempts) * cfg.Interval
	return nil, fmt.Errorf("update processing timed out after %s", totalWait)
}
