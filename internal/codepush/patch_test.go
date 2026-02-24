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

func TestPatch(t *testing.T) {
	t.Run("successful patch with label", func(t *testing.T) {
		var capturedReq PatchRequest
		var capturedPackageID string
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
				}, nil
			},
			patchPackageFunc: func(appID, deploymentID, packageID string, req PatchRequest) (*Package, error) {
				capturedReq = req
				capturedPackageID = packageID
				return &Package{
					ID:         packageID,
					Label:      "v2",
					AppVersion: "1.0.0",
					Mandatory:  true,
					Rollout:    50,
				}, nil
			},
		}

		opts := &PatchOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			Label:        "v2",
			Rollout:      "50",
			Mandatory:    "true",
		}

		result, err := Patch(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "pkg-2", capturedPackageID)
		assert.Equal(t, "v2", result.Label)
		assert.Equal(t, 50, *capturedReq.Rollout)
		assert.Equal(t, true, *capturedReq.Mandatory)
	})

	t.Run("successful patch defaults to latest", func(t *testing.T) {
		var capturedPackageID string
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
					{ID: "pkg-3", Label: "v3"},
				}, nil
			},
			patchPackageFunc: func(appID, deploymentID, packageID string, req PatchRequest) (*Package, error) {
				capturedPackageID = packageID
				return &Package{
					ID:         packageID,
					Label:      "v3",
					AppVersion: "2.0.0",
					Rollout:    100,
				}, nil
			},
		}

		opts := &PatchOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			Rollout:      "100",
		}

		result, err := Patch(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "pkg-3", capturedPackageID)
		assert.Equal(t, "v3", result.Label)
	})

	t.Run("patch with all fields", func(t *testing.T) {
		var capturedReq PatchRequest
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{{ID: "pkg-1", Label: "v1"}}, nil
			},
			patchPackageFunc: func(appID, deploymentID, packageID string, req PatchRequest) (*Package, error) {
				capturedReq = req
				return &Package{
					ID:          packageID,
					Label:       "v1",
					AppVersion:  "3.0.0",
					Mandatory:   true,
					Disabled:    false,
					Rollout:     75,
					Description: "hotfix",
				}, nil
			},
		}

		opts := &PatchOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			Rollout:      "75",
			Mandatory:    "true",
			Disabled:     "false",
			Description:  "hotfix",
			AppVersion:   "3.0.0",
		}

		result, err := Patch(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, 75, *capturedReq.Rollout)
		assert.Equal(t, true, *capturedReq.Mandatory)
		assert.Equal(t, false, *capturedReq.Disabled)
		assert.Equal(t, "hotfix", *capturedReq.Description)
		assert.Equal(t, "3.0.0", *capturedReq.AppVersion)
		assert.Equal(t, "hotfix", result.Description)
	})

	t.Run("no releases in deployment", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{}, nil
			},
		}

		opts := &PatchOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			Rollout:      "50",
		}

		_, err := Patch(context.Background(), client, opts, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "no releases found")
	})

	t.Run("label not found", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{{ID: "pkg-1", Label: "v1"}}, nil
			},
		}

		opts := &PatchOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			Label:        "v99",
			Rollout:      "50",
		}

		_, err := Patch(context.Background(), client, opts, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "v99")
	})

	t.Run("deployment name resolution", func(t *testing.T) {
		var resolvedDeploymentID string
		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return []Deployment{
					{ID: "dep-aaa", Name: "Staging"},
					{ID: "dep-bbb", Name: "Production"},
				}, nil
			},
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				resolvedDeploymentID = deploymentID
				return []Package{{ID: "pkg-1", Label: "v1"}}, nil
			},
			patchPackageFunc: func(appID, deploymentID, packageID string, req PatchRequest) (*Package, error) {
				return &Package{ID: packageID, Label: "v1", Rollout: 50}, nil
			},
		}

		opts := &PatchOptions{
			AppID:        "app-123",
			DeploymentID: "Production",
			Token:        "test-token",
			Rollout:      "50",
		}

		_, err := Patch(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "dep-bbb", resolvedDeploymentID)
	})

	t.Run("API error", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{{ID: "pkg-1", Label: "v1"}}, nil
			},
			patchPackageFunc: func(appID, deploymentID, packageID string, req PatchRequest) (*Package, error) {
				return nil, fmt.Errorf("API returned HTTP 500: internal error")
			},
		}

		opts := &PatchOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			Rollout:      "50",
		}

		_, err := Patch(context.Background(), client, opts, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "patch failed")
	})

	t.Run("bitrise environment exports summary", func(t *testing.T) {
		deployDir := t.TempDir()
		t.Setenv("BITRISE_DEPLOY_DIR", deployDir)
		t.Setenv("BITRISE_BUILD_NUMBER", "42")

		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{{ID: "pkg-1", Label: "v1", AppVersion: "1.0.0"}}, nil
			},
			patchPackageFunc: func(appID, deploymentID, packageID string, req PatchRequest) (*Package, error) {
				return &Package{
					ID:         packageID,
					Label:      "v1",
					AppVersion: "1.0.0",
					Rollout:    50,
				}, nil
			},
		}

		opts := &PatchOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			Rollout:      "50",
		}

		_, err := Patch(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		summaryPath := filepath.Join(deployDir, "codepush-patch-summary.json")
		data, err := os.ReadFile(summaryPath)
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, `"label": "v1"`)
		assert.Contains(t, content, `"rollout": 50`)
	})
}

func TestValidatePatchOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    PatchOptions
		wantErr string
	}{
		{
			name:    "missing app ID",
			opts:    PatchOptions{DeploymentID: "dep", Token: "tok", Rollout: "50"},
			wantErr: "app ID is required",
		},
		{
			name:    "missing deployment",
			opts:    PatchOptions{AppID: "app", Token: "tok", Rollout: "50"},
			wantErr: "deployment is required",
		},
		{
			name:    "missing token",
			opts:    PatchOptions{AppID: "app", DeploymentID: "dep", Rollout: "50"},
			wantErr: "API token is required",
		},
		{
			name:    "no changes provided",
			opts:    PatchOptions{AppID: "app", DeploymentID: "dep", Token: "tok"},
			wantErr: "at least one change is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePatchOptions(&tt.opts)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestBuildPatchRequest(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		opts := &PatchOptions{
			AppID:        "app",
			DeploymentID: "dep",
			Token:        "tok",
			Rollout:      "75",
			Mandatory:    "true",
			Disabled:     "false",
			Description:  "updated",
			AppVersion:   "2.0.0",
		}

		req, err := buildPatchRequest(opts)
		require.NoError(t, err)

		require.NotNil(t, req.Rollout)
		assert.Equal(t, 75, *req.Rollout)
		require.NotNil(t, req.Mandatory)
		assert.Equal(t, true, *req.Mandatory)
		require.NotNil(t, req.Disabled)
		assert.Equal(t, false, *req.Disabled)
		require.NotNil(t, req.Description)
		assert.Equal(t, "updated", *req.Description)
		require.NotNil(t, req.AppVersion)
		assert.Equal(t, "2.0.0", *req.AppVersion)
	})

	t.Run("only rollout", func(t *testing.T) {
		opts := &PatchOptions{
			AppID:        "app",
			DeploymentID: "dep",
			Token:        "tok",
			Rollout:      "50",
		}

		req, err := buildPatchRequest(opts)
		require.NoError(t, err)

		require.NotNil(t, req.Rollout)
		assert.Equal(t, 50, *req.Rollout)
		assert.Nil(t, req.Mandatory)
		assert.Nil(t, req.Disabled)
		assert.Nil(t, req.Description)
		assert.Nil(t, req.AppVersion)
	})

	t.Run("invalid rollout too low", func(t *testing.T) {
		opts := &PatchOptions{Rollout: "0"}
		_, err := buildPatchRequest(opts)
		require.Error(t, err)
		assert.ErrorContains(t, err, "rollout must be between")
	})

	t.Run("invalid rollout too high", func(t *testing.T) {
		opts := &PatchOptions{Rollout: "101"}
		_, err := buildPatchRequest(opts)
		require.Error(t, err)
		assert.ErrorContains(t, err, "rollout must be between")
	})

	t.Run("invalid rollout not a number", func(t *testing.T) {
		opts := &PatchOptions{Rollout: "abc"}
		_, err := buildPatchRequest(opts)
		require.Error(t, err)
		assert.ErrorContains(t, err, "rollout must be between")
	})

	t.Run("invalid mandatory", func(t *testing.T) {
		opts := &PatchOptions{Mandatory: "maybe"}
		_, err := buildPatchRequest(opts)
		require.Error(t, err)
		assert.ErrorContains(t, err, "mandatory must be true or false")
	})

	t.Run("invalid disabled", func(t *testing.T) {
		opts := &PatchOptions{Disabled: "maybe"}
		_, err := buildPatchRequest(opts)
		require.Error(t, err)
		assert.ErrorContains(t, err, "disabled must be true or false")
	})
}

func TestResolvePackageForPatch(t *testing.T) {
	t.Run("resolves by label", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
				}, nil
			},
		}

		id, label, err := ResolvePackageForPatch(context.Background(), client, "app-123", "dep-456", "v2", testOut)
		require.NoError(t, err)
		assert.Equal(t, "pkg-2", id)
		assert.Equal(t, "v2", label)
	})

	t.Run("resolves latest when no label", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
					{ID: "pkg-3", Label: "v3"},
				}, nil
			},
		}

		id, label, err := ResolvePackageForPatch(context.Background(), client, "app-123", "dep-456", "", testOut)
		require.NoError(t, err)
		assert.Equal(t, "pkg-3", id)
		assert.Equal(t, "v3", label)
	})

	t.Run("empty deployment", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{}, nil
			},
		}

		_, _, err := ResolvePackageForPatch(context.Background(), client, "app-123", "dep-456", "", testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "no releases found")
	})

	t.Run("list packages error", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		_, _, err := ResolvePackageForPatch(context.Background(), client, "app-123", "dep-456", "", testOut)
		require.Error(t, err)
	})
}
