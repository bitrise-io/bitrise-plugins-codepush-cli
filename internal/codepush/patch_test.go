package codepush

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatch(t *testing.T) {
	t.Run("successful patch with label", func(t *testing.T) {
		var capturedReq PatchRequest
		var capturedUpdateID string
		client := &mockClient{
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
				}, nil
			},
			patchUpdateFunc: func(updateID string, req PatchRequest) (*Update, error) {
				capturedReq = req
				capturedUpdateID = updateID
				return &Update{
					ID:        updateID,
					Label:     "v2",
					Mandatory: true,
					Rollout:   50,
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

		assert.Equal(t, "pkg-2", capturedUpdateID)
		assert.Equal(t, "v2", result.Label)
		assert.InDelta(t, 50.0, *capturedReq.Rollout, 0.001)
		assert.Equal(t, "true", capturedReq.Mandatory)
	})

	t.Run("successful patch defaults to latest", func(t *testing.T) {
		var capturedUpdateID string
		client := &mockClient{
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
					{ID: "pkg-3", Label: "v3"},
				}, nil
			},
			patchUpdateFunc: func(updateID string, req PatchRequest) (*Update, error) {
				capturedUpdateID = updateID
				return &Update{
					ID:      updateID,
					Label:   "v3",
					Rollout: 100,
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

		assert.Equal(t, "pkg-3", capturedUpdateID)
		assert.Equal(t, "v3", result.Label)
	})

	t.Run("patch with rollout and mandatory", func(t *testing.T) {
		var capturedReq PatchRequest
		client := &mockClient{
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{{ID: "pkg-1", Label: "v1"}}, nil
			},
			patchUpdateFunc: func(updateID string, req PatchRequest) (*Update, error) {
				capturedReq = req
				return &Update{
					ID:        updateID,
					Label:     "v1",
					Mandatory: true,
					Disabled:  false,
					Rollout:   75,
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
		}

		result, err := Patch(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.InDelta(t, 75.0, *capturedReq.Rollout, 0.001)
		assert.Equal(t, "true", capturedReq.Mandatory)
		assert.Equal(t, "false", capturedReq.Disabled)
		assert.InDelta(t, 75.0, result.Rollout, 0.001)
	})

	t.Run("no releases in deployment", func(t *testing.T) {
		client := &mockClient{
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{}, nil
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
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{{ID: "pkg-1", Label: "v1"}}, nil
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
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				resolvedDeploymentID = deploymentID
				return []Update{{ID: "pkg-1", Label: "v1"}}, nil
			},
			patchUpdateFunc: func(updateID string, req PatchRequest) (*Update, error) {
				return &Update{ID: updateID, Label: "v1", Rollout: 50}, nil
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
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{{ID: "pkg-1", Label: "v1"}}, nil
			},
			patchUpdateFunc: func(updateID string, req PatchRequest) (*Update, error) {
				return nil, errors.New("API returned HTTP 500: internal error")
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
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{{ID: "pkg-1", Label: "v1", AppVersion: "1.0.0"}}, nil
			},
			patchUpdateFunc: func(updateID string, req PatchRequest) (*Update, error) {
				return &Update{
					ID:         updateID,
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
		}

		req, err := buildPatchRequest(opts)
		require.NoError(t, err)

		require.NotNil(t, req.Rollout)
		assert.InDelta(t, 75.0, *req.Rollout, 0.001)
		assert.Equal(t, "true", req.Mandatory)
		assert.Equal(t, "false", req.Disabled)
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
		assert.InDelta(t, 50.0, *req.Rollout, 0.001)
		assert.Empty(t, req.Mandatory)
		assert.Empty(t, req.Disabled)
	})

	t.Run("rollout zero is valid", func(t *testing.T) {
		opts := &PatchOptions{Rollout: "0"}
		req, err := buildPatchRequest(opts)
		require.NoError(t, err)
		require.NotNil(t, req.Rollout)
		assert.InDelta(t, 0.0, *req.Rollout, 0.001)
	})

	t.Run("invalid rollout too low", func(t *testing.T) {
		opts := &PatchOptions{Rollout: "-1"}
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

	t.Run("normalizes non-canonical bool aliases to true/false", func(t *testing.T) {
		for _, alias := range []string{"1", "T", "TRUE"} {
			opts := &PatchOptions{Mandatory: alias, Disabled: "0"}
			req, err := buildPatchRequest(opts)
			require.NoError(t, err, "alias=%s", alias)
			assert.Equal(t, "true", req.Mandatory, "alias=%s", alias)
			assert.Equal(t, "false", req.Disabled, "alias=%s", alias)
		}
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

func TestResolveUpdateForPatch(t *testing.T) {
	t.Run("resolves by label", func(t *testing.T) {
		client := &mockClient{
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
				}, nil
			},
		}

		id, label, err := ResolveUpdateForPatch(context.Background(), client, "dep-456", "v2", testOut)
		require.NoError(t, err)
		assert.Equal(t, "pkg-2", id)
		assert.Equal(t, "v2", label)
	})

	t.Run("resolves latest when no label", func(t *testing.T) {
		client := &mockClient{
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
					{ID: "pkg-3", Label: "v3"},
				}, nil
			},
		}

		id, label, err := ResolveUpdateForPatch(context.Background(), client, "dep-456", "", testOut)
		require.NoError(t, err)
		assert.Equal(t, "pkg-3", id)
		assert.Equal(t, "v3", label)
	})

	t.Run("empty deployment", func(t *testing.T) {
		client := &mockClient{
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return []Update{}, nil
			},
		}

		_, _, err := ResolveUpdateForPatch(context.Background(), client, "dep-456", "", testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "no releases found")
	})

	t.Run("list updates error", func(t *testing.T) {
		client := &mockClient{
			listUpdatesFunc: func(deploymentID string) ([]Update, error) {
				return nil, errors.New("network error")
			},
		}

		_, _, err := ResolveUpdateForPatch(context.Background(), client, "dep-456", "", testOut)
		require.Error(t, err)
	})
}
