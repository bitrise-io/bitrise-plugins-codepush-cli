package codepush

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPClientListDeployments(t *testing.T) {
	t.Run("returns deployments", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "test-token" {
				t.Errorf("auth header: got %q, want plain token without Bearer prefix", r.Header.Get("Authorization"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[{"id":"dep-1","name":"Staging"},{"id":"dep-2","name":"Production"}]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		deployments, err := client.ListDeployments(context.Background(),"app-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(deployments) != 2 {
			t.Fatalf("deployments: got %d, want 2", len(deployments))
		}
		if deployments[0].ID != "dep-1" || deployments[0].Name != "Staging" {
			t.Errorf("deployment[0]: got %+v", deployments[0])
		}
		if deployments[1].ID != "dep-2" || deployments[1].Name != "Production" {
			t.Errorf("deployment[1]: got %+v", deployments[1])
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid token"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "bad-token")
		_, err := client.ListDeployments(context.Background(),"app-123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("error should contain status code: %v", err)
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		deployments, err := client.ListDeployments(context.Background(),"app-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(deployments) != 0 {
			t.Errorf("deployments: got %d, want 0", len(deployments))
		}
	})
}

func TestHTTPClientCreateDeployment(t *testing.T) {
	t.Run("creates deployment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("method: got %q, want POST", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content-type: got %q", r.Header.Get("Content-Type"))
			}

			var body CreateDeploymentRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decoding body: %v", err)
			}
			if body.Name != "QA" {
				t.Errorf("name: got %q", body.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"dep-new","name":"QA"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		dep, err := client.CreateDeployment(context.Background(),"app-123", CreateDeploymentRequest{Name: "QA"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dep.ID != "dep-new" {
			t.Errorf("id: got %q", dep.ID)
		}
		if dep.Name != "QA" {
			t.Errorf("name: got %q", dep.Name)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"error":"deployment already exists"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.CreateDeployment(context.Background(),"app-123", CreateDeploymentRequest{Name: "QA"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "409") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientGetDeployment(t *testing.T) {
	t.Run("returns deployment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-456" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("method: got %q, want GET", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"dep-456","name":"Staging","created_at":"2025-01-01T00:00:00Z","key":"abc123"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		dep, err := client.GetDeployment(context.Background(),"app-123", "dep-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dep.ID != "dep-456" {
			t.Errorf("id: got %q", dep.ID)
		}
		if dep.Name != "Staging" {
			t.Errorf("name: got %q", dep.Name)
		}
		if dep.Key != "abc123" {
			t.Errorf("key: got %q", dep.Key)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetDeployment(context.Background(),"app-123", "dep-456")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientRenameDeployment(t *testing.T) {
	t.Run("renames deployment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-456" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodPatch {
				t.Errorf("method: got %q, want PATCH", r.Method)
			}

			var body RenameDeploymentRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decoding body: %v", err)
			}
			if body.Name != "Pre-Production" {
				t.Errorf("name: got %q", body.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"dep-456","name":"Pre-Production"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		dep, err := client.RenameDeployment(context.Background(),"app-123", "dep-456", RenameDeploymentRequest{Name: "Pre-Production"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if dep.Name != "Pre-Production" {
			t.Errorf("name: got %q", dep.Name)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid name"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.RenameDeployment(context.Background(),"app-123", "dep-456", RenameDeploymentRequest{Name: ""})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "400") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientDeleteDeployment(t *testing.T) {
	t.Run("deletes deployment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-456" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodDelete {
				t.Errorf("method: got %q, want DELETE", r.Method)
			}

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		err := client.DeleteDeployment(context.Background(),"app-123", "dep-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		err := client.DeleteDeployment(context.Background(),"app-123", "dep-456")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientGetUploadURL(t *testing.T) {
	t.Run("constructs correct request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789/upload-url"
			if r.URL.Path != expectedPath {
				t.Errorf("path: got %q, want %q", r.URL.Path, expectedPath)
			}

			query := r.URL.Query()
			if query.Get("app_version") != "1.0.0" {
				t.Errorf("app_version: got %q", query.Get("app_version"))
			}
			if query.Get("file_name") != "bundle.zip" {
				t.Errorf("file_name: got %q", query.Get("file_name"))
			}
			if query.Get("file_size_bytes") != "1024" {
				t.Errorf("file_size_bytes: got %q", query.Get("file_size_bytes"))
			}
			if query.Get("mandatory") != "true" {
				t.Errorf("mandatory: got %q", query.Get("mandatory"))
			}
			if query.Get("description") != "test update" {
				t.Errorf("description: got %q", query.Get("description"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url":"https://storage.example.com/upload","method":"PUT","headers":{"content_type":"application/zip"}}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		resp, err := client.GetUploadURL(context.Background(),"app-123", "dep-456", "pkg-789", UploadURLRequest{
			AppVersion:    "1.0.0",
			FileName:      "bundle.zip",
			FileSizeBytes: 1024,
			Description:   "test update",
			Mandatory:     true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.URL != "https://storage.example.com/upload" {
			t.Errorf("url: got %q", resp.URL)
		}
		if resp.Method != "PUT" {
			t.Errorf("method: got %q", resp.Method)
		}
	})

	t.Run("omits optional params when empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			if query.Get("description") != "" {
				t.Errorf("description should be empty, got %q", query.Get("description"))
			}
			if query.Get("mandatory") != "" {
				t.Errorf("mandatory should be empty, got %q", query.Get("mandatory"))
			}
			if query.Get("rollout") != "" {
				t.Errorf("rollout should be empty for 100%%, got %q", query.Get("rollout"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url":"https://example.com/upload","method":"PUT","headers":{}}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetUploadURL(context.Background(),"app-123", "dep-456", "pkg-789", UploadURLRequest{
			AppVersion:    "1.0.0",
			FileName:      "bundle.zip",
			FileSizeBytes: 512,
			Rollout:       100,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("includes rollout when less than 100", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("rollout") != "25" {
				t.Errorf("rollout: got %q, want 25", r.URL.Query().Get("rollout"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url":"https://example.com/upload","method":"PUT","headers":{}}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetUploadURL(context.Background(),"app-123", "dep-456", "pkg-789", UploadURLRequest{
			AppVersion:    "1.0.0",
			FileName:      "bundle.zip",
			FileSizeBytes: 512,
			Rollout:       25,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"deployment not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetUploadURL(context.Background(),"app-123", "dep-456", "pkg-789", UploadURLRequest{
			AppVersion:    "1.0.0",
			FileName:      "bundle.zip",
			FileSizeBytes: 512,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientUploadFile(t *testing.T) {
	t.Run("uploads file with correct headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method: got %q, want PUT", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/zip" {
				t.Errorf("content-type: got %q", r.Header.Get("Content-Type"))
			}
			if r.ContentLength != 11 {
				t.Errorf("content-length: got %d, want 11", r.ContentLength)
			}

			body, _ := io.ReadAll(r.Body)
			if string(body) != "zip content" {
				t.Errorf("body: got %q", string(body))
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPClient("", "test-token")
		err := client.UploadFile(context.Background(),UploadFileRequest{
			URL:           server.URL,
			Method:        http.MethodPut,
			Headers:       map[string]string{"Content-Type": "application/zip"},
			Body:          strings.NewReader("zip content"),
			ContentLength: 11,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles upload failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("URL expired"))
		}))
		defer server.Close()

		client := NewHTTPClient("", "test-token")
		err := client.UploadFile(context.Background(),UploadFileRequest{
			URL:           server.URL,
			Method:        http.MethodPut,
			Body:          strings.NewReader("data"),
			ContentLength: 4,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "403") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientGetPackageStatus(t *testing.T) {
	t.Run("returns status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789/status"
			if r.URL.Path != expectedPath {
				t.Errorf("path: got %q", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"package_id":"pkg-789","status":"done","status_reason":""}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		status, err := client.GetPackageStatus(context.Background(),"app-123", "dep-456", "pkg-789")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if status.PackageID != "pkg-789" {
			t.Errorf("package_id: got %q", status.PackageID)
		}
		if status.Status != "done" {
			t.Errorf("status: got %q", status.Status)
		}
	})

	t.Run("returns failed status with reason", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"package_id":"pkg-789","status":"failed","status_reason":"invalid bundle format"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		status, err := client.GetPackageStatus(context.Background(),"app-123", "dep-456", "pkg-789")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if status.Status != "failed" {
			t.Errorf("status: got %q", status.Status)
		}
		if status.StatusReason != "invalid bundle format" {
			t.Errorf("status_reason: got %q", status.StatusReason)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetPackageStatus(context.Background(),"app-123", "dep-456", "pkg-789")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientListPackages(t *testing.T) {
	t.Run("returns packages", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-456/packages" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "test-token" {
				t.Errorf("auth header: got %q", r.Header.Get("Authorization"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[{"id":"pkg-1","label":"v1","app_version":"1.0.0"},{"id":"pkg-2","label":"v2","app_version":"2.0.0"}]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		packages, err := client.ListPackages(context.Background(),"app-123", "dep-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(packages) != 2 {
			t.Fatalf("packages: got %d, want 2", len(packages))
		}
		if packages[0].ID != "pkg-1" || packages[0].Label != "v1" {
			t.Errorf("package[0]: got %+v", packages[0])
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		packages, err := client.ListPackages(context.Background(),"app-123", "dep-456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(packages) != 0 {
			t.Errorf("packages: got %d, want 0", len(packages))
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"deployment not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.ListPackages(context.Background(),"app-123", "dep-456")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientGetPackage(t *testing.T) {
	t.Run("returns package", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("method: got %q, want GET", r.Method)
			}
			if r.Header.Get("Authorization") != "test-token" {
				t.Errorf("auth header: got %q", r.Header.Get("Authorization"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-789","label":"v3","app_version":"1.0.0","mandatory":true,"rollout":50}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		pkg, err := client.GetPackage(context.Background(),"app-123", "dep-456", "pkg-789")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pkg.ID != "pkg-789" {
			t.Errorf("id: got %q", pkg.ID)
		}
		if pkg.Label != "v3" {
			t.Errorf("label: got %q", pkg.Label)
		}
		if pkg.Mandatory != true {
			t.Error("mandatory should be true")
		}
		if pkg.Rollout != 50 {
			t.Errorf("rollout: got %d", pkg.Rollout)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"package not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetPackage(context.Background(),"app-123", "dep-456", "pkg-789")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientPatchPackage(t *testing.T) {
	t.Run("sends correct PATCH request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodPatch {
				t.Errorf("method: got %q, want PATCH", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content-type: got %q", r.Header.Get("Content-Type"))
			}

			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decoding body: %v", err)
			}
			if body["rollout"] != float64(50) {
				t.Errorf("rollout: got %v", body["rollout"])
			}
			if body["mandatory"] != true {
				t.Errorf("mandatory: got %v", body["mandatory"])
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-789","label":"v3","app_version":"1.0.0","mandatory":true,"rollout":50}`))
		}))
		defer server.Close()

		rollout := 50
		mandatory := true
		client := NewHTTPClient(server.URL, "test-token")
		pkg, err := client.PatchPackage(context.Background(),"app-123", "dep-456", "pkg-789", PatchRequest{
			Rollout:   &rollout,
			Mandatory: &mandatory,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pkg.ID != "pkg-789" {
			t.Errorf("id: got %q", pkg.ID)
		}
		if pkg.Rollout != 50 {
			t.Errorf("rollout: got %d", pkg.Rollout)
		}
	})

	t.Run("omits nil fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)
			if strings.Contains(bodyStr, "mandatory") {
				t.Errorf("body should not contain mandatory: %s", bodyStr)
			}
			if strings.Contains(bodyStr, "disabled") {
				t.Errorf("body should not contain disabled: %s", bodyStr)
			}
			if !strings.Contains(bodyStr, "rollout") {
				t.Errorf("body should contain rollout: %s", bodyStr)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-789","label":"v3","rollout":50}`))
		}))
		defer server.Close()

		rollout := 50
		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.PatchPackage(context.Background(),"app-123", "dep-456", "pkg-789", PatchRequest{
			Rollout: &rollout,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid rollout value"}`))
		}))
		defer server.Close()

		rollout := 50
		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.PatchPackage(context.Background(),"app-123", "dep-456", "pkg-789", PatchRequest{
			Rollout: &rollout,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "400") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientDeletePackage(t *testing.T) {
	t.Run("deletes package", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodDelete {
				t.Errorf("method: got %q, want DELETE", r.Method)
			}

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		err := client.DeletePackage(context.Background(),"app-123", "dep-456", "pkg-789")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"package not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		err := client.DeletePackage(context.Background(),"app-123", "dep-456", "pkg-789")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientRollback(t *testing.T) {
	t.Run("sends correct request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-456/rollback" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("method: got %q, want POST", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content-type: got %q", r.Header.Get("Content-Type"))
			}
			if r.Header.Get("Authorization") != "test-token" {
				t.Errorf("auth header: got %q", r.Header.Get("Authorization"))
			}

			var body RollbackRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decoding body: %v", err)
			}
			if body.PackageID != "pkg-target" {
				t.Errorf("body package_id: got %q", body.PackageID)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-new","label":"v4","app_version":"1.0.0"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		pkg, err := client.Rollback(context.Background(),"app-123", "dep-456", RollbackRequest{PackageID: "pkg-target"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pkg.ID != "pkg-new" {
			t.Errorf("id: got %q", pkg.ID)
		}
		if pkg.Label != "v4" {
			t.Errorf("label: got %q", pkg.Label)
		}
	})

	t.Run("omits empty package_id", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "package_id") {
				t.Errorf("body should not contain package_id: %s", body)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-new","label":"v2"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.Rollback(context.Background(),"app-123", "dep-456", RollbackRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"no releases"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.Rollback(context.Background(),"app-123", "dep-456", RollbackRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestHTTPClientPromote(t *testing.T) {
	t.Run("sends correct request with all fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/connected-apps/app-123/code-push/deployments/dep-src/promote" {
				t.Errorf("path: got %q", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("method: got %q, want POST", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content-type: got %q", r.Header.Get("Content-Type"))
			}

			var body PromoteRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decoding body: %v", err)
			}
			if body.TargetDeploymentID != "dep-dst" {
				t.Errorf("target_deployment_id: got %q", body.TargetDeploymentID)
			}
			if body.AppVersion != "3.0.0" {
				t.Errorf("app_version: got %q", body.AppVersion)
			}
			if body.Mandatory != "true" {
				t.Errorf("mandatory: got %q", body.Mandatory)
			}
			if body.Rollout != "50" {
				t.Errorf("rollout: got %q", body.Rollout)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-promoted","label":"v1","app_version":"3.0.0"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		pkg, err := client.Promote(context.Background(),"app-123", "dep-src", PromoteRequest{
			TargetDeploymentID: "dep-dst",
			AppVersion:         "3.0.0",
			Mandatory:          "true",
			Rollout:            "50",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if pkg.ID != "pkg-promoted" {
			t.Errorf("id: got %q", pkg.ID)
		}
	})

	t.Run("omits empty optional fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)
			if strings.Contains(bodyStr, "package_id") {
				t.Errorf("body should not contain package_id: %s", bodyStr)
			}
			if strings.Contains(bodyStr, "app_version") {
				t.Errorf("body should not contain app_version: %s", bodyStr)
			}
			if strings.Contains(bodyStr, "mandatory") {
				t.Errorf("body should not contain mandatory: %s", bodyStr)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-new","label":"v1"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.Promote(context.Background(),"app-123", "dep-src", PromoteRequest{
			TargetDeploymentID: "dep-dst",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"error":"duplicate release"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.Promote(context.Background(),"app-123", "dep-src", PromoteRequest{TargetDeploymentID: "dep-dst"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "409") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}
