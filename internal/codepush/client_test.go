package codepush

import (
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
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("auth header: got %q", r.Header.Get("Authorization"))
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[{"id":"dep-1","name":"Staging"},{"id":"dep-2","name":"Production"}]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		deployments, err := client.ListDeployments("app-123")
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
		_, err := client.ListDeployments("app-123")
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
		deployments, err := client.ListDeployments("app-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(deployments) != 0 {
			t.Errorf("deployments: got %d, want 0", len(deployments))
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
		resp, err := client.GetUploadURL("app-123", "dep-456", "pkg-789", UploadURLRequest{
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
		_, err := client.GetUploadURL("app-123", "dep-456", "pkg-789", UploadURLRequest{
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
		_, err := client.GetUploadURL("app-123", "dep-456", "pkg-789", UploadURLRequest{
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
		_, err := client.GetUploadURL("app-123", "dep-456", "pkg-789", UploadURLRequest{
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
		err := client.UploadFile(
			server.URL,
			http.MethodPut,
			map[string]string{"Content-Type": "application/zip"},
			strings.NewReader("zip content"),
			11,
		)
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
		err := client.UploadFile(server.URL, http.MethodPut, nil, strings.NewReader("data"), 4)
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
		status, err := client.GetPackageStatus("app-123", "dep-456", "pkg-789")
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
		status, err := client.GetPackageStatus("app-123", "dep-456", "pkg-789")
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
		_, err := client.GetPackageStatus("app-123", "dep-456", "pkg-789")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}
