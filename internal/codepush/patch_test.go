package codepush

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

		result, err := Patch(client, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedPackageID != "pkg-2" {
			t.Errorf("package_id: got %q, want %q", capturedPackageID, "pkg-2")
		}
		if result.Label != "v2" {
			t.Errorf("label: got %q", result.Label)
		}
		if *capturedReq.Rollout != 50 {
			t.Errorf("rollout: got %d", *capturedReq.Rollout)
		}
		if *capturedReq.Mandatory != true {
			t.Error("mandatory should be true")
		}
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

		result, err := Patch(client, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedPackageID != "pkg-3" {
			t.Errorf("should patch latest package: got %q, want %q", capturedPackageID, "pkg-3")
		}
		if result.Label != "v3" {
			t.Errorf("label: got %q", result.Label)
		}
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

		result, err := Patch(client, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if *capturedReq.Rollout != 75 {
			t.Errorf("rollout: got %d", *capturedReq.Rollout)
		}
		if *capturedReq.Mandatory != true {
			t.Error("mandatory should be true")
		}
		if *capturedReq.Disabled != false {
			t.Error("disabled should be false")
		}
		if *capturedReq.Description != "hotfix" {
			t.Errorf("description: got %q", *capturedReq.Description)
		}
		if *capturedReq.AppVersion != "3.0.0" {
			t.Errorf("app_version: got %q", *capturedReq.AppVersion)
		}
		if result.Description != "hotfix" {
			t.Errorf("result description: got %q", result.Description)
		}
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

		_, err := Patch(client, opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no releases found") {
			t.Errorf("error should mention no releases: %v", err)
		}
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

		_, err := Patch(client, opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "v99") {
			t.Errorf("error should mention label: %v", err)
		}
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

		_, err := Patch(client, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resolvedDeploymentID != "dep-bbb" {
			t.Errorf("resolved deployment: got %q, want %q", resolvedDeploymentID, "dep-bbb")
		}
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

		_, err := Patch(client, opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "patch failed") {
			t.Errorf("error should mention patch failed: %v", err)
		}
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

		_, err := Patch(client, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		summaryPath := filepath.Join(deployDir, "codepush-patch-summary.json")
		data, err := os.ReadFile(summaryPath)
		if err != nil {
			t.Fatalf("reading summary: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, `"label": "v1"`) {
			t.Errorf("summary should contain label: %s", content)
		}
		if !strings.Contains(content, `"rollout": 50`) {
			t.Errorf("summary should contain rollout: %s", content)
		}
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
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if req.Rollout == nil || *req.Rollout != 75 {
			t.Errorf("rollout: got %v", req.Rollout)
		}
		if req.Mandatory == nil || *req.Mandatory != true {
			t.Errorf("mandatory: got %v", req.Mandatory)
		}
		if req.Disabled == nil || *req.Disabled != false {
			t.Errorf("disabled: got %v", req.Disabled)
		}
		if req.Description == nil || *req.Description != "updated" {
			t.Errorf("description: got %v", req.Description)
		}
		if req.AppVersion == nil || *req.AppVersion != "2.0.0" {
			t.Errorf("app_version: got %v", req.AppVersion)
		}
	})

	t.Run("only rollout", func(t *testing.T) {
		opts := &PatchOptions{
			AppID:        "app",
			DeploymentID: "dep",
			Token:        "tok",
			Rollout:      "50",
		}

		req, err := buildPatchRequest(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if req.Rollout == nil || *req.Rollout != 50 {
			t.Errorf("rollout: got %v", req.Rollout)
		}
		if req.Mandatory != nil {
			t.Errorf("mandatory should be nil: got %v", req.Mandatory)
		}
		if req.Disabled != nil {
			t.Errorf("disabled should be nil: got %v", req.Disabled)
		}
		if req.Description != nil {
			t.Errorf("description should be nil: got %v", req.Description)
		}
		if req.AppVersion != nil {
			t.Errorf("app_version should be nil: got %v", req.AppVersion)
		}
	})

	t.Run("invalid rollout too low", func(t *testing.T) {
		opts := &PatchOptions{Rollout: "0"}
		_, err := buildPatchRequest(opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "rollout must be between") {
			t.Errorf("error: %v", err)
		}
	})

	t.Run("invalid rollout too high", func(t *testing.T) {
		opts := &PatchOptions{Rollout: "101"}
		_, err := buildPatchRequest(opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "rollout must be between") {
			t.Errorf("error: %v", err)
		}
	})

	t.Run("invalid rollout not a number", func(t *testing.T) {
		opts := &PatchOptions{Rollout: "abc"}
		_, err := buildPatchRequest(opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "rollout must be between") {
			t.Errorf("error: %v", err)
		}
	})

	t.Run("invalid mandatory", func(t *testing.T) {
		opts := &PatchOptions{Mandatory: "maybe"}
		_, err := buildPatchRequest(opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "mandatory must be true or false") {
			t.Errorf("error: %v", err)
		}
	})

	t.Run("invalid disabled", func(t *testing.T) {
		opts := &PatchOptions{Disabled: "maybe"}
		_, err := buildPatchRequest(opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "disabled must be true or false") {
			t.Errorf("error: %v", err)
		}
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

		id, label, err := resolvePackageForPatch(client, "app-123", "dep-456", "v2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "pkg-2" {
			t.Errorf("id: got %q, want %q", id, "pkg-2")
		}
		if label != "v2" {
			t.Errorf("label: got %q, want %q", label, "v2")
		}
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

		id, label, err := resolvePackageForPatch(client, "app-123", "dep-456", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "pkg-3" {
			t.Errorf("id: got %q, want %q", id, "pkg-3")
		}
		if label != "v3" {
			t.Errorf("label: got %q, want %q", label, "v3")
		}
	})

	t.Run("empty deployment", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return []Package{}, nil
			},
		}

		_, _, err := resolvePackageForPatch(client, "app-123", "dep-456", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no releases found") {
			t.Errorf("error: %v", err)
		}
	})

	t.Run("list packages error", func(t *testing.T) {
		client := &mockClient{
			listPackagesFunc: func(appID, deploymentID string) ([]Package, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		_, _, err := resolvePackageForPatch(client, "app-123", "dep-456", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
