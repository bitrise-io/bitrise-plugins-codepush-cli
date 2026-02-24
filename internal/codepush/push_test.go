package codepush

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPush(t *testing.T) {
	t.Run("successful end to end", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)
		var capturedReq UploadURLRequest
		var capturedUploadBody []byte

		client := &mockClient{
			getUploadURLFunc: func(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error) {
				capturedReq = req
				assert.Equal(t, "app-123", appID)
				return &UploadURLResponse{
					URL:     "https://storage.example.com/upload",
					Method:  "PUT",
					Headers: map[string]string{"Content-Type": "application/zip"},
				}, nil
			},
			uploadFileFunc: func(req UploadFileRequest) error {
				assert.Equal(t, "https://storage.example.com/upload", req.URL)
				assert.Equal(t, "PUT", req.Method)
				capturedUploadBody, _ = io.ReadAll(req.Body)
				return nil
			},
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{PackageID: packageID, Status: StatusProcessedValid}, nil
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			AppVersion:   "1.0.0",
			Description:  "test update",
			Mandatory:    true,
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		result, err := PushWithConfig(context.Background(), client, opts, fastPollConfig, testOut)
		require.NoError(t, err)

		assert.Equal(t, "1.0.0", result.AppVersion)
		assert.Equal(t, StatusProcessedValid, result.Status)
		assert.NotEmpty(t, result.PackageID)
		assert.NotZero(t, result.FileSizeBytes)

		assert.Equal(t, "1.0.0", capturedReq.AppVersion)
		assert.True(t, capturedReq.Mandatory)
		assert.NotEmpty(t, capturedUploadBody)
	})

	t.Run("deployment name resolution", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)
		var resolvedDeploymentID string

		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return []Deployment{
					{ID: "dep-aaa", Name: "Staging"},
					{ID: "dep-bbb", Name: "Production"},
				}, nil
			},
			getUploadURLFunc: func(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error) {
				resolvedDeploymentID = deploymentID
				return &UploadURLResponse{URL: "https://example.com/upload", Method: "PUT"}, nil
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "Production",
			Token:        "test-token",
			AppVersion:   "1.0.0",
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		_, err := PushWithConfig(context.Background(), client, opts, fastPollConfig, testOut)
		require.NoError(t, err)

		assert.Equal(t, "dep-bbb", resolvedDeploymentID)
	})

	t.Run("deployment UUID passthrough", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)
		listCalled := false

		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				listCalled = true
				return nil, nil
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			AppVersion:   "1.0.0",
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		_, err := PushWithConfig(context.Background(), client, opts, fastPollConfig, testOut)
		require.NoError(t, err)

		assert.False(t, listCalled, "ListDeployments should not be called when UUID is provided")
	})

	t.Run("deployment name not found", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)

		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return []Deployment{
					{ID: "dep-aaa", Name: "Staging"},
				}, nil
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "NonExistent",
			Token:        "test-token",
			AppVersion:   "1.0.0",
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		_, err := PushWithConfig(context.Background(), client, opts, fastPollConfig, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "NonExistent")
	})

	t.Run("upload URL failure", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)

		client := &mockClient{
			getUploadURLFunc: func(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error) {
				return nil, fmt.Errorf("API returned HTTP 500: internal error")
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			AppVersion:   "1.0.0",
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		_, err := PushWithConfig(context.Background(), client, opts, fastPollConfig, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "upload URL")
	})

	t.Run("upload failure", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)

		client := &mockClient{
			uploadFileFunc: func(req UploadFileRequest) error {
				return fmt.Errorf("upload failed with HTTP 403: URL expired")
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			AppVersion:   "1.0.0",
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		_, err := PushWithConfig(context.Background(), client, opts, fastPollConfig, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "uploading package")
	})

	t.Run("poll returns failed", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)

		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{
					PackageID:    packageID,
					Status:       StatusProcessedError,
					StatusReason: "invalid bundle format",
				}, nil
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			AppVersion:   "1.0.0",
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		_, err := PushWithConfig(context.Background(), client, opts, fastPollConfig, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid bundle format")
	})

	t.Run("poll timeout", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)

		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{PackageID: packageID, Status: StatusUploaded}, nil
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			AppVersion:   "1.0.0",
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		_, err := PushWithConfig(context.Background(), client, opts, PollConfig{MaxAttempts: 2, Interval: 1 * time.Millisecond}, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "timed out")
	})

	t.Run("does not export bitrise summary", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)
		deployDir := t.TempDir()
		t.Setenv("BITRISE_DEPLOY_DIR", deployDir)
		t.Setenv("BITRISE_BUILD_NUMBER", "42")

		client := &mockClient{}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			AppVersion:   "2.0.0",
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		_, err := PushWithConfig(context.Background(), client, opts, fastPollConfig, testOut)
		require.NoError(t, err)

		// Push no longer exports to Bitrise deploy dir; the CLI layer handles that
		summaryPath := filepath.Join(deployDir, "codepush-push-summary.json")
		_, err = os.Stat(summaryPath)
		assert.Error(t, err, "push should not export summary; that responsibility moved to CLI layer")
	})
}

func TestValidatePushOptions(t *testing.T) {
	bundleDir := createTestBundleDir(t)

	tests := []struct {
		name    string
		opts    PushOptions
		wantErr string
	}{
		{
			name:    "missing app ID",
			opts:    PushOptions{DeploymentID: "dep", Token: "tok", AppVersion: "1.0", Rollout: 100, BundlePath: bundleDir},
			wantErr: "app ID is required",
		},
		{
			name:    "missing deployment",
			opts:    PushOptions{AppID: "app", Token: "tok", AppVersion: "1.0", Rollout: 100, BundlePath: bundleDir},
			wantErr: "deployment is required",
		},
		{
			name:    "missing token",
			opts:    PushOptions{AppID: "app", DeploymentID: "dep", AppVersion: "1.0", Rollout: 100, BundlePath: bundleDir},
			wantErr: "API token is required",
		},
		{
			name:    "missing app version",
			opts:    PushOptions{AppID: "app", DeploymentID: "dep", Token: "tok", Rollout: 100, BundlePath: bundleDir},
			wantErr: "app version is required",
		},
		{
			name:    "missing bundle path",
			opts:    PushOptions{AppID: "app", DeploymentID: "dep", Token: "tok", AppVersion: "1.0", Rollout: 100},
			wantErr: "bundle path is required",
		},
		{
			name:    "rollout too low",
			opts:    PushOptions{AppID: "app", DeploymentID: "dep", Token: "tok", AppVersion: "1.0", Rollout: 0, BundlePath: bundleDir},
			wantErr: "rollout must be between 1 and 100",
		},
		{
			name:    "rollout too high",
			opts:    PushOptions{AppID: "app", DeploymentID: "dep", Token: "tok", AppVersion: "1.0", Rollout: 101, BundlePath: bundleDir},
			wantErr: "rollout must be between 1 and 100",
		},
		{
			name:    "bundle path does not exist",
			opts:    PushOptions{AppID: "app", DeploymentID: "dep", Token: "tok", AppVersion: "1.0", Rollout: 100, BundlePath: "/nonexistent"},
			wantErr: "bundle path does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePushOptions(&tt.opts)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestResolveDeployment(t *testing.T) {
	t.Run("UUID passthrough", func(t *testing.T) {
		client := &mockClient{}
		id, err := ResolveDeployment(context.Background(), client, "app-123", "00000000-0000-0000-0000-000000000001", testOut)
		require.NoError(t, err)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", id)
	})

	t.Run("name resolution", func(t *testing.T) {
		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return []Deployment{
					{ID: "dep-aaa", Name: "Staging"},
					{ID: "dep-bbb", Name: "Production"},
				}, nil
			},
		}

		id, err := ResolveDeployment(context.Background(), client, "app-123", "Production", testOut)
		require.NoError(t, err)
		assert.Equal(t, "dep-bbb", id)
	})

	t.Run("name not found", func(t *testing.T) {
		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return []Deployment{{ID: "dep-aaa", Name: "Staging"}}, nil
			},
		}

		_, err := ResolveDeployment(context.Background(), client, "app-123", "Production", testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "Production")
	})

	t.Run("list deployments error", func(t *testing.T) {
		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		_, err := ResolveDeployment(context.Background(), client, "app-123", "Production", testOut)
		require.Error(t, err)
	})
}

func TestPollStatus(t *testing.T) {
	t.Run("returns on done", func(t *testing.T) {
		callCount := 0
		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				callCount++
				if callCount < 3 {
					return &PackageStatus{PackageID: packageID, Status: StatusUploaded}, nil
				}
				return &PackageStatus{PackageID: packageID, Status: StatusProcessedValid}, nil
			},
		}

		ref := PackageRef{AppID: "app", DeploymentID: "dep", PackageID: "pkg"}
		status, err := pollStatus(context.Background(), client, ref, PollConfig{MaxAttempts: 5, Interval: 1 * time.Millisecond})
		require.NoError(t, err)
		assert.Equal(t, StatusProcessedValid, status.Status)
		assert.Equal(t, 3, callCount)
	})

	t.Run("returns error on failed", func(t *testing.T) {
		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{PackageID: packageID, Status: StatusProcessedError, StatusReason: "bad format"}, nil
			},
		}

		ref := PackageRef{AppID: "app", DeploymentID: "dep", PackageID: "pkg"}
		_, err := pollStatus(context.Background(), client, ref, fastPollConfig)
		require.Error(t, err)
		assert.ErrorContains(t, err, "bad format")
	})

	t.Run("times out", func(t *testing.T) {
		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{PackageID: packageID, Status: StatusUploaded}, nil
			},
		}

		ref := PackageRef{AppID: "app", DeploymentID: "dep", PackageID: "pkg"}
		_, err := pollStatus(context.Background(), client, ref, PollConfig{MaxAttempts: 2, Interval: 1 * time.Millisecond})
		require.Error(t, err)
		assert.ErrorContains(t, err, "timed out")
	})
}

func createTestBundleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundle")
	require.NoError(t, os.Mkdir(bundleDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bundleDir, "main.jsbundle"), []byte("bundle"), 0o644))
	return bundleDir
}
