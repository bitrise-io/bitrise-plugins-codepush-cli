package codepush

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollback(t *testing.T) {
	t.Run("successful rollback without target release", func(t *testing.T) {
		var capturedReq RollbackRequest
		client := &mockClient{
			rollbackFunc: func(appID, deploymentID string, req RollbackRequest) (*Package, error) {
				capturedReq = req
				assert.Equal(t, "app-123", appID)
				assert.Equal(t, "00000000-0000-0000-0000-000000000001", deploymentID)
				return &Package{
					ID:         "pkg-rolled-back",
					Label:      "v5",
					AppVersion: "1.0.0",
				}, nil
			},
		}

		opts := &RollbackOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
		}

		result, err := Rollback(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "pkg-rolled-back", result.PackageID)
		assert.Equal(t, "v5", result.Label)
		assert.Empty(t, capturedReq.PackageID)
	})

	t.Run("rollback with target release label", func(t *testing.T) {
		var capturedReq RollbackRequest
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
					{ID: "pkg-3", Label: "v3"},
				}, nil
			},
			rollbackFunc: func(appID, deploymentID string, req RollbackRequest) (*Package, error) {
				capturedReq = req
				return &Package{ID: "pkg-new", Label: "v4", AppVersion: "1.0.0"}, nil
			},
		}

		opts := &RollbackOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			TargetLabel:  "v2",
		}

		_, err := Rollback(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "pkg-2", capturedReq.PackageID)
	})

	t.Run("target release label not found", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{
					{ID: "pkg-1", Label: "v1"},
				}, nil
			},
		}

		opts := &RollbackOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			TargetLabel:  "v99",
		}

		_, err := Rollback(context.Background(), client, opts, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "v99")
	})

	t.Run("deployment name resolution", func(t *testing.T) {
		var resolvedID string
		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return []Deployment{
					{ID: "dep-aaa", Name: "Staging"},
					{ID: "dep-bbb", Name: "Production"},
				}, nil
			},
			rollbackFunc: func(appID, deploymentID string, req RollbackRequest) (*Package, error) {
				resolvedID = deploymentID
				return &Package{ID: "pkg-new", Label: "v2"}, nil
			},
		}

		opts := &RollbackOptions{
			AppID:        "app-123",
			DeploymentID: "Production",
			Token:        "test-token",
		}

		_, err := Rollback(context.Background(), client, opts, testOut)
		require.NoError(t, err)
		assert.Equal(t, "dep-bbb", resolvedID)
	})

	t.Run("API error", func(t *testing.T) {
		client := &mockClient{
			rollbackFunc: func(appID, deploymentID string, req RollbackRequest) (*Package, error) {
				return nil, fmt.Errorf("API returned HTTP 404: deployment not found")
			},
		}

		opts := &RollbackOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
		}

		_, err := Rollback(context.Background(), client, opts, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "rollback failed")
	})

	t.Run("bitrise environment exports summary", func(t *testing.T) {
		deployDir := t.TempDir()
		t.Setenv("BITRISE_DEPLOY_DIR", deployDir)
		t.Setenv("BITRISE_BUILD_NUMBER", "42")

		client := &mockClient{
			rollbackFunc: func(appID, deploymentID string, req RollbackRequest) (*Package, error) {
				return &Package{ID: "pkg-rb", Label: "v5", AppVersion: "1.0.0"}, nil
			},
		}

		opts := &RollbackOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
		}

		_, err := Rollback(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		summaryPath := filepath.Join(deployDir, "codepush-rollback-summary.json")
		data, err := os.ReadFile(summaryPath)
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, `"label": "v5"`)
	})
}

func TestValidateRollbackOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    RollbackOptions
		wantErr string
	}{
		{
			name:    "missing app ID",
			opts:    RollbackOptions{DeploymentID: "dep", Token: "tok"},
			wantErr: "app ID is required",
		},
		{
			name:    "missing deployment",
			opts:    RollbackOptions{AppID: "app", Token: "tok"},
			wantErr: "deployment is required",
		},
		{
			name:    "missing token",
			opts:    RollbackOptions{AppID: "app", DeploymentID: "dep"},
			wantErr: "API token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRollbackOptions(&tt.opts)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestResolvePackageLabel(t *testing.T) {
	t.Run("finds matching label", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
				}, nil
			},
		}

		id, err := resolvePackageLabel(context.Background(), client, "app-123", "dep-456", "v2", testOut)
		require.NoError(t, err)
		assert.Equal(t, "pkg-2", id)
	})

	t.Run("label not found", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{{ID: "pkg-1", Label: "v1"}}, nil
			},
		}

		_, err := resolvePackageLabel(context.Background(), client, "app-123", "dep-456", "v99", testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "v99")
	})

	t.Run("list packages error", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		_, err := resolvePackageLabel(context.Background(), client, "app-123", "dep-456", "v1", testOut)
		require.Error(t, err)
	})
}
