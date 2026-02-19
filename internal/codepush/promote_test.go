package codepush

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

		result, err := Promote(client, opts, testOut)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.PackageID != "pkg-promoted" {
			t.Errorf("package_id: got %q", result.PackageID)
		}
		if capturedSourceDepID != "00000000-0000-0000-0000-000000000001" {
			t.Errorf("source deployment: got %q", capturedSourceDepID)
		}
		if capturedReq.TargetDeploymentID != "00000000-0000-0000-0000-000000000002" {
			t.Errorf("target deployment in request: got %q", capturedReq.TargetDeploymentID)
		}
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

		_, err := Promote(client, opts, testOut)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedReq.AppVersion != "3.0.0" {
			t.Errorf("app_version: got %q", capturedReq.AppVersion)
		}
		if capturedReq.Description != "production release" {
			t.Errorf("description: got %q", capturedReq.Description)
		}
		if capturedReq.Mandatory != "true" {
			t.Errorf("mandatory: got %q", capturedReq.Mandatory)
		}
		if capturedReq.Rollout != "50" {
			t.Errorf("rollout: got %q", capturedReq.Rollout)
		}
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

		_, err := Promote(client, opts, testOut)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedReq.PackageID != "pkg-2" {
			t.Errorf("package_id: got %q, want %q", capturedReq.PackageID, "pkg-2")
		}
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

		_, err := Promote(client, opts, testOut)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedSourceDepID != "dep-aaa" {
			t.Errorf("source deployment: got %q, want %q", capturedSourceDepID, "dep-aaa")
		}
		if capturedDestDepID != "dep-bbb" {
			t.Errorf("dest deployment: got %q, want %q", capturedDestDepID, "dep-bbb")
		}
	})

	t.Run("same source and destination error", func(t *testing.T) {
		opts := &PromoteOptions{
			AppID:              "app-123",
			SourceDeploymentID: "Staging",
			DestDeploymentID:   "Staging",
			Token:              "test-token",
		}

		_, err := Promote(&mockClient{}, opts, testOut)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "must be different") {
			t.Errorf("error should mention different: %v", err)
		}
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

		_, err := Promote(client, opts, testOut)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "promote failed") {
			t.Errorf("error should mention promote failed: %v", err)
		}
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

		_, err := Promote(client, opts, testOut)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		summaryPath := filepath.Join(deployDir, "codepush-promote-summary.json")
		data, err := os.ReadFile(summaryPath)
		if err != nil {
			t.Fatalf("reading summary: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, `"label": "v1"`) {
			t.Errorf("summary should contain label: %s", content)
		}
		if !strings.Contains(content, `"dest_deployment_id"`) {
			t.Errorf("summary should contain dest_deployment_id: %s", content)
		}
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
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
