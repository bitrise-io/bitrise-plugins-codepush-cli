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

func TestPromote(t *testing.T) {
	t.Run("successful promote", func(t *testing.T) {
		var capturedReq PromoteRequest
		var capturedSourceDepID string
		client := &mockClient{
			promoteFunc: func(appID, deploymentID string, req PromoteRequest) (*Package, error) {
				capturedReq = req
				capturedSourceDepID = deploymentID
				return &Package{
					ID:         "pkg-promoted",
					Label:      "v1",
					AppVersion: "2.0.0",
				}, nil
			},
		}

		opts := &PromoteOptions{
			AppID:              "app-123",
			SourceDeploymentID: "00000000-0000-0000-0000-000000000001",
			DestDeploymentID:   "00000000-0000-0000-0000-000000000002",
			Token:              "test-token",
		}

		result, err := Promote(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "pkg-promoted", result.PackageID)
		assert.Equal(t, "00000000-0000-0000-0000-000000000001", capturedSourceDepID)
		assert.Equal(t, "00000000-0000-0000-0000-000000000002", capturedReq.TargetDeploymentID)
	})

	t.Run("promote with overrides", func(t *testing.T) {
		var capturedReq PromoteRequest
		client := &mockClient{
			promoteFunc: func(appID, deploymentID string, req PromoteRequest) (*Package, error) {
				capturedReq = req
				return &Package{ID: "pkg-new", Label: "v1", AppVersion: "3.0.0"}, nil
			},
		}

		opts := &PromoteOptions{
			AppID:              "app-123",
			SourceDeploymentID: "00000000-0000-0000-0000-000000000001",
			DestDeploymentID:   "00000000-0000-0000-0000-000000000002",
			Token:              "test-token",
			AppVersion:         "3.0.0",
			Description:        "production release",
			Mandatory:          "true",
			Disabled:           "false",
			Rollout:            "50",
		}

		_, err := Promote(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "3.0.0", capturedReq.AppVersion)
		assert.Equal(t, "production release", capturedReq.Description)
		assert.Equal(t, "true", capturedReq.Mandatory)
		assert.Equal(t, "50", capturedReq.Rollout)
	})

	t.Run("promote with label resolution", func(t *testing.T) {
		var capturedReq PromoteRequest
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{
					{ID: "pkg-1", Label: "v1"},
					{ID: "pkg-2", Label: "v2"},
				}, nil
			},
			promoteFunc: func(appID, deploymentID string, req PromoteRequest) (*Package, error) {
				capturedReq = req
				return &Package{ID: "pkg-new", Label: "v1"}, nil
			},
		}

		opts := &PromoteOptions{
			AppID:              "app-123",
			SourceDeploymentID: "00000000-0000-0000-0000-000000000001",
			DestDeploymentID:   "00000000-0000-0000-0000-000000000002",
			Token:              "test-token",
			Label:              "v2",
		}

		_, err := Promote(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "pkg-2", capturedReq.PackageID)
	})

	t.Run("deployment name resolution for both source and dest", func(t *testing.T) {
		var capturedSourceDepID string
		var capturedDestDepID string
		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return []Deployment{
					{ID: "dep-aaa", Name: "Staging"},
					{ID: "dep-bbb", Name: "Production"},
				}, nil
			},
			promoteFunc: func(appID, deploymentID string, req PromoteRequest) (*Package, error) {
				capturedSourceDepID = deploymentID
				capturedDestDepID = req.TargetDeploymentID
				return &Package{ID: "pkg-new", Label: "v1"}, nil
			},
		}

		opts := &PromoteOptions{
			AppID:              "app-123",
			SourceDeploymentID: "Staging",
			DestDeploymentID:   "Production",
			Token:              "test-token",
		}

		_, err := Promote(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		assert.Equal(t, "dep-aaa", capturedSourceDepID)
		assert.Equal(t, "dep-bbb", capturedDestDepID)
	})

	t.Run("same source and destination error", func(t *testing.T) {
		opts := &PromoteOptions{
			AppID:              "app-123",
			SourceDeploymentID: "Staging",
			DestDeploymentID:   "Staging",
			Token:              "test-token",
		}

		_, err := Promote(context.Background(), &mockClient{}, opts, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "must be different")
	})

	t.Run("API error", func(t *testing.T) {
		client := &mockClient{
			promoteFunc: func(appID, deploymentID string, req PromoteRequest) (*Package, error) {
				return nil, fmt.Errorf("API returned HTTP 409: conflict")
			},
		}

		opts := &PromoteOptions{
			AppID:              "app-123",
			SourceDeploymentID: "00000000-0000-0000-0000-000000000001",
			DestDeploymentID:   "00000000-0000-0000-0000-000000000002",
			Token:              "test-token",
		}

		_, err := Promote(context.Background(), client, opts, testOut)
		require.Error(t, err)
		assert.ErrorContains(t, err, "promote failed")
	})

	t.Run("bitrise environment exports summary", func(t *testing.T) {
		deployDir := t.TempDir()
		t.Setenv("BITRISE_DEPLOY_DIR", deployDir)
		t.Setenv("BITRISE_BUILD_NUMBER", "42")

		client := &mockClient{
			promoteFunc: func(appID, deploymentID string, req PromoteRequest) (*Package, error) {
				return &Package{ID: "pkg-promo", Label: "v1", AppVersion: "2.0.0"}, nil
			},
		}

		opts := &PromoteOptions{
			AppID:              "app-123",
			SourceDeploymentID: "00000000-0000-0000-0000-000000000001",
			DestDeploymentID:   "00000000-0000-0000-0000-000000000002",
			Token:              "test-token",
		}

		_, err := Promote(context.Background(), client, opts, testOut)
		require.NoError(t, err)

		summaryPath := filepath.Join(deployDir, "codepush-promote-summary.json")
		data, err := os.ReadFile(summaryPath)
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, `"label": "v1"`)
		assert.Contains(t, content, `"dest_deployment_id"`)
	})
}

func TestValidatePromoteOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    PromoteOptions
		wantErr string
	}{
		{
			name:    "missing app ID",
			opts:    PromoteOptions{SourceDeploymentID: "src", DestDeploymentID: "dst", Token: "tok"},
			wantErr: "app ID is required",
		},
		{
			name:    "missing source deployment",
			opts:    PromoteOptions{AppID: "app", DestDeploymentID: "dst", Token: "tok"},
			wantErr: "source deployment is required",
		},
		{
			name:    "missing destination deployment",
			opts:    PromoteOptions{AppID: "app", SourceDeploymentID: "src", Token: "tok"},
			wantErr: "destination deployment is required",
		},
		{
			name:    "missing token",
			opts:    PromoteOptions{AppID: "app", SourceDeploymentID: "src", DestDeploymentID: "dst"},
			wantErr: "API token is required",
		},
		{
			name:    "same source and destination",
			opts:    PromoteOptions{AppID: "app", SourceDeploymentID: "same", DestDeploymentID: "same", Token: "tok"},
			wantErr: "must be different",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePromoteOptions(&tt.opts)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
