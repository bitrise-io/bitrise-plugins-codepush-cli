package codepush

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRollback(t *testing.T) {
	t.Run("successful rollback without target release", func(t *testing.T) {
		var capturedReq RollbackRequest
		client := &mockClient{
			rollbackFunc: func(appID, deploymentID string, req RollbackRequest) (*Package, error) {
				capturedReq = req
				if appID != "app-123" {
					t.Errorf("appID: got %q", appID)
				}
				if deploymentID != "00000000-0000-0000-0000-000000000001" {
					t.Errorf("deploymentID: got %q", deploymentID)
				}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.PackageID != "pkg-rolled-back" {
			t.Errorf("package_id: got %q", result.PackageID)
		}
		if result.Label != "v5" {
			t.Errorf("label: got %q", result.Label)
		}
		if capturedReq.PackageID != "" {
			t.Errorf("request package_id should be empty: got %q", capturedReq.PackageID)
		}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedReq.PackageID != "pkg-2" {
			t.Errorf("request package_id: got %q, want %q", capturedReq.PackageID, "pkg-2")
		}
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
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "v99") {
			t.Errorf("error should mention label: %v", err)
		}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resolvedID != "dep-bbb" {
			t.Errorf("resolved deployment ID: got %q, want %q", resolvedID, "dep-bbb")
		}
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
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "rollback failed") {
			t.Errorf("error should mention rollback failed: %v", err)
		}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		summaryPath := filepath.Join(deployDir, "codepush-rollback-summary.json")
		data, err := os.ReadFile(summaryPath)
		if err != nil {
			t.Fatalf("reading summary: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, `"label": "v5"`) {
			t.Errorf("summary should contain label: %s", content)
		}
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
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "pkg-2" {
			t.Errorf("id: got %q, want %q", id, "pkg-2")
		}
	})

	t.Run("label not found", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{{ID: "pkg-1", Label: "v1"}}, nil
			},
		}

		_, err := resolvePackageLabel(context.Background(), client, "app-123", "dep-456", "v99", testOut)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "v99") {
			t.Errorf("error should mention label: %v", err)
		}
	})

	t.Run("list packages error", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		_, err := resolvePackageLabel(context.Background(), client, "app-123", "dep-456", "v1", testOut)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
