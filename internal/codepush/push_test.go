package codepush

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockClient struct {
	listDeploymentsFunc  func(appID string) ([]Deployment, error)
	getUploadURLFunc     func(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error)
	uploadFileFunc       func(uploadURL, method string, headers map[string]string, body io.Reader, contentLength int64) error
	getPackageStatusFunc func(appID, deploymentID, packageID string) (*PackageStatus, error)
}

func (m *mockClient) ListDeployments(appID string) ([]Deployment, error) {
	if m.listDeploymentsFunc != nil {
		return m.listDeploymentsFunc(appID)
	}
	return nil, nil
}

func (m *mockClient) GetUploadURL(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error) {
	if m.getUploadURLFunc != nil {
		return m.getUploadURLFunc(appID, deploymentID, packageID, req)
	}
	return &UploadURLResponse{URL: "https://example.com/upload", Method: "PUT"}, nil
}

func (m *mockClient) UploadFile(uploadURL, method string, headers map[string]string, body io.Reader, contentLength int64) error {
	if m.uploadFileFunc != nil {
		return m.uploadFileFunc(uploadURL, method, headers, body, contentLength)
	}
	return nil
}

func (m *mockClient) GetPackageStatus(appID, deploymentID, packageID string) (*PackageStatus, error) {
	if m.getPackageStatusFunc != nil {
		return m.getPackageStatusFunc(appID, deploymentID, packageID)
	}
	return &PackageStatus{PackageID: packageID, Status: StatusDone}, nil
}

var fastPollConfig = PollConfig{
	MaxAttempts: 3,
	Interval:    1 * time.Millisecond,
}

func TestPush(t *testing.T) {
	t.Run("successful end to end", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)
		var capturedReq UploadURLRequest
		var capturedUploadBody []byte

		client := &mockClient{
			getUploadURLFunc: func(appID, deploymentID, packageID string, req UploadURLRequest) (*UploadURLResponse, error) {
				capturedReq = req
				if appID != "app-123" {
					t.Errorf("appID: got %q", appID)
				}
				return &UploadURLResponse{
					URL:     "https://storage.example.com/upload",
					Method:  "PUT",
					Headers: map[string]string{"Content-Type": "application/zip"},
				}, nil
			},
			uploadFileFunc: func(uploadURL, method string, headers map[string]string, body io.Reader, contentLength int64) error {
				if uploadURL != "https://storage.example.com/upload" {
					t.Errorf("uploadURL: got %q", uploadURL)
				}
				if method != "PUT" {
					t.Errorf("method: got %q", method)
				}
				capturedUploadBody, _ = io.ReadAll(body)
				return nil
			},
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{PackageID: packageID, Status: StatusDone}, nil
			},
		}

		opts := &PushOptions{
			AppID:        "app-123",
			DeploymentID: "00000000-0000-0000-0000-000000000001",
			Token:        "test-token",
			APIURL:       "https://api.example.com",
			AppVersion:   "1.0.0",
			Description:  "test update",
			Mandatory:    true,
			Rollout:      100,
			BundlePath:   bundleDir,
		}

		result, err := PushWithConfig(client, opts, fastPollConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.AppVersion != "1.0.0" {
			t.Errorf("app_version: got %q", result.AppVersion)
		}
		if result.Status != StatusDone {
			t.Errorf("status: got %q", result.Status)
		}
		if result.PackageID == "" {
			t.Error("package_id should not be empty")
		}
		if result.FileSizeBytes == 0 {
			t.Error("file_size_bytes should not be 0")
		}

		if capturedReq.AppVersion != "1.0.0" {
			t.Errorf("upload req app_version: got %q", capturedReq.AppVersion)
		}
		if capturedReq.Mandatory != true {
			t.Error("upload req mandatory should be true")
		}
		if len(capturedUploadBody) == 0 {
			t.Error("upload body should not be empty")
		}
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

		_, err := PushWithConfig(client, opts, fastPollConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resolvedDeploymentID != "dep-bbb" {
			t.Errorf("resolved deployment ID: got %q, want %q", resolvedDeploymentID, "dep-bbb")
		}
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

		_, err := PushWithConfig(client, opts, fastPollConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if listCalled {
			t.Error("ListDeployments should not be called when UUID is provided")
		}
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

		_, err := PushWithConfig(client, opts, fastPollConfig)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "NonExistent") {
			t.Errorf("error should mention deployment name: %v", err)
		}
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

		_, err := PushWithConfig(client, opts, fastPollConfig)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "upload URL") {
			t.Errorf("error should mention upload URL: %v", err)
		}
	})

	t.Run("upload failure", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)

		client := &mockClient{
			uploadFileFunc: func(uploadURL, method string, headers map[string]string, body io.Reader, contentLength int64) error {
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

		_, err := PushWithConfig(client, opts, fastPollConfig)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "uploading package") {
			t.Errorf("error should mention uploading: %v", err)
		}
	})

	t.Run("poll returns failed", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)

		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{
					PackageID:    packageID,
					Status:       StatusFailed,
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

		_, err := PushWithConfig(client, opts, fastPollConfig)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid bundle format") {
			t.Errorf("error should contain status reason: %v", err)
		}
	})

	t.Run("poll timeout", func(t *testing.T) {
		bundleDir := createTestBundleDir(t)

		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{PackageID: packageID, Status: StatusProcessing}, nil
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

		_, err := PushWithConfig(client, opts, PollConfig{MaxAttempts: 2, Interval: 1 * time.Millisecond})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Errorf("error should mention timeout: %v", err)
		}
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

		_, err := PushWithConfig(client, opts, fastPollConfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Push no longer exports to Bitrise deploy dir; the CLI layer handles that
		summaryPath := filepath.Join(deployDir, "codepush-push-summary.json")
		if _, err := os.Stat(summaryPath); err == nil {
			t.Error("push should not export summary; that responsibility moved to CLI layer")
		}
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
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestResolveDeployment(t *testing.T) {
	t.Run("UUID passthrough", func(t *testing.T) {
		client := &mockClient{}
		id, err := resolveDeployment(client, "app-123", "00000000-0000-0000-0000-000000000001")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "00000000-0000-0000-0000-000000000001" {
			t.Errorf("id: got %q", id)
		}
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

		id, err := resolveDeployment(client, "app-123", "Production")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "dep-bbb" {
			t.Errorf("id: got %q, want %q", id, "dep-bbb")
		}
	})

	t.Run("name not found", func(t *testing.T) {
		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return []Deployment{{ID: "dep-aaa", Name: "Staging"}}, nil
			},
		}

		_, err := resolveDeployment(client, "app-123", "Production")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Production") {
			t.Errorf("error should mention deployment name: %v", err)
		}
	})

	t.Run("list deployments error", func(t *testing.T) {
		client := &mockClient{
			listDeploymentsFunc: func(appID string) ([]Deployment, error) {
				return nil, fmt.Errorf("network error")
			},
		}

		_, err := resolveDeployment(client, "app-123", "Production")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPollStatus(t *testing.T) {
	t.Run("returns on done", func(t *testing.T) {
		callCount := 0
		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				callCount++
				if callCount < 3 {
					return &PackageStatus{PackageID: packageID, Status: StatusProcessing}, nil
				}
				return &PackageStatus{PackageID: packageID, Status: StatusDone}, nil
			},
		}

		status, err := pollStatus(client, "app", "dep", "pkg", PollConfig{MaxAttempts: 5, Interval: 1 * time.Millisecond})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status.Status != StatusDone {
			t.Errorf("status: got %q", status.Status)
		}
		if callCount != 3 {
			t.Errorf("call count: got %d, want 3", callCount)
		}
	})

	t.Run("returns error on failed", func(t *testing.T) {
		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{PackageID: packageID, Status: StatusFailed, StatusReason: "bad format"}, nil
			},
		}

		_, err := pollStatus(client, "app", "dep", "pkg", fastPollConfig)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "bad format") {
			t.Errorf("error should contain reason: %v", err)
		}
	})

	t.Run("times out", func(t *testing.T) {
		client := &mockClient{
			getPackageStatusFunc: func(appID, deploymentID, packageID string) (*PackageStatus, error) {
				return &PackageStatus{PackageID: packageID, Status: StatusProcessing}, nil
			},
		}

		_, err := pollStatus(client, "app", "dep", "pkg", PollConfig{MaxAttempts: 2, Interval: 1 * time.Millisecond})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Errorf("error should mention timeout: %v", err)
		}
	})
}

func createTestBundleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundle")
	if err := os.Mkdir(bundleDir, 0o755); err != nil {
		t.Fatalf("creating bundle dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "main.jsbundle"), []byte("bundle"), 0o644); err != nil {
		t.Fatalf("writing bundle file: %v", err)
	}
	return bundleDir
}
