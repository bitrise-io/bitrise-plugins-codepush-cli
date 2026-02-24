package codepush

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClientListDeployments(t *testing.T) {
	t.Run("returns deployments", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments", r.URL.Path)
			assert.Equal(t, "test-token", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[{"id":"dep-1","name":"Staging"},{"id":"dep-2","name":"Production"}]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		deployments, err := client.ListDeployments(context.Background(), "app-123")
		require.NoError(t, err)

		require.Len(t, deployments, 2)
		assert.Equal(t, "dep-1", deployments[0].ID)
		assert.Equal(t, "Staging", deployments[0].Name)
		assert.Equal(t, "dep-2", deployments[1].ID)
		assert.Equal(t, "Production", deployments[1].Name)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid token"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "bad-token")
		_, err := client.ListDeployments(context.Background(), "app-123")
		require.Error(t, err)
		assert.ErrorContains(t, err, "401")
	})

	t.Run("handles empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		deployments, err := client.ListDeployments(context.Background(), "app-123")
		require.NoError(t, err)
		assert.Empty(t, deployments)
	})
}

func TestHTTPClientCreateDeployment(t *testing.T) {
	t.Run("creates deployment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body CreateDeploymentRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "QA", body.Name)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"dep-new","name":"QA"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		dep, err := client.CreateDeployment(context.Background(), "app-123", CreateDeploymentRequest{Name: "QA"})
		require.NoError(t, err)

		assert.Equal(t, "dep-new", dep.ID)
		assert.Equal(t, "QA", dep.Name)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"error":"deployment already exists"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.CreateDeployment(context.Background(), "app-123", CreateDeploymentRequest{Name: "QA"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "409")
	})
}

func TestHTTPClientGetDeployment(t *testing.T) {
	t.Run("returns deployment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-456", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"dep-456","name":"Staging","created_at":"2025-01-01T00:00:00Z","key":"abc123"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		dep, err := client.GetDeployment(context.Background(), "app-123", "dep-456")
		require.NoError(t, err)

		assert.Equal(t, "dep-456", dep.ID)
		assert.Equal(t, "Staging", dep.Name)
		assert.Equal(t, "abc123", dep.Key)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetDeployment(context.Background(), "app-123", "dep-456")
		require.Error(t, err)
		assert.ErrorContains(t, err, "404")
	})
}

func TestHTTPClientRenameDeployment(t *testing.T) {
	t.Run("renames deployment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-456", r.URL.Path)
			assert.Equal(t, http.MethodPatch, r.Method)

			var body RenameDeploymentRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "Pre-Production", body.Name)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"dep-456","name":"Pre-Production"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		dep, err := client.RenameDeployment(context.Background(), "app-123", "dep-456", RenameDeploymentRequest{Name: "Pre-Production"})
		require.NoError(t, err)

		assert.Equal(t, "Pre-Production", dep.Name)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid name"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.RenameDeployment(context.Background(), "app-123", "dep-456", RenameDeploymentRequest{Name: ""})
		require.Error(t, err)
		assert.ErrorContains(t, err, "400")
	})
}

func TestHTTPClientDeleteDeployment(t *testing.T) {
	t.Run("deletes deployment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-456", r.URL.Path)
			assert.Equal(t, http.MethodDelete, r.Method)

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		err := client.DeleteDeployment(context.Background(), "app-123", "dep-456")
		require.NoError(t, err)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		err := client.DeleteDeployment(context.Background(), "app-123", "dep-456")
		require.Error(t, err)
		assert.ErrorContains(t, err, "404")
	})
}

func TestHTTPClientGetUploadURL(t *testing.T) {
	t.Run("constructs correct request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789/upload-url"
			assert.Equal(t, expectedPath, r.URL.Path)

			query := r.URL.Query()
			assert.Equal(t, "1.0.0", query.Get("app_version"))
			assert.Equal(t, "bundle.zip", query.Get("file_name"))
			assert.Equal(t, "1024", query.Get("file_size_bytes"))
			assert.Equal(t, "true", query.Get("mandatory"))
			assert.Equal(t, "test update", query.Get("description"))

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url":"https://storage.example.com/upload","method":"PUT","headers":{"content_type":"application/zip"}}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		resp, err := client.GetUploadURL(context.Background(), "app-123", "dep-456", "pkg-789", UploadURLRequest{
			AppVersion:    "1.0.0",
			FileName:      "bundle.zip",
			FileSizeBytes: 1024,
			Description:   "test update",
			Mandatory:     true,
		})
		require.NoError(t, err)

		assert.Equal(t, "https://storage.example.com/upload", resp.URL)
		assert.Equal(t, "PUT", resp.Method)
	})

	t.Run("omits optional params when empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			assert.Empty(t, query.Get("description"))
			assert.Empty(t, query.Get("mandatory"))
			assert.Empty(t, query.Get("rollout"))

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url":"https://example.com/upload","method":"PUT","headers":{}}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetUploadURL(context.Background(), "app-123", "dep-456", "pkg-789", UploadURLRequest{
			AppVersion:    "1.0.0",
			FileName:      "bundle.zip",
			FileSizeBytes: 512,
			Rollout:       100,
		})
		require.NoError(t, err)
	})

	t.Run("includes rollout when less than 100", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "25", r.URL.Query().Get("rollout"))

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url":"https://example.com/upload","method":"PUT","headers":{}}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetUploadURL(context.Background(), "app-123", "dep-456", "pkg-789", UploadURLRequest{
			AppVersion:    "1.0.0",
			FileName:      "bundle.zip",
			FileSizeBytes: 512,
			Rollout:       25,
		})
		require.NoError(t, err)
	})

	t.Run("handles API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"deployment not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetUploadURL(context.Background(), "app-123", "dep-456", "pkg-789", UploadURLRequest{
			AppVersion:    "1.0.0",
			FileName:      "bundle.zip",
			FileSizeBytes: 512,
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "404")
	})
}

func TestHTTPClientUploadFile(t *testing.T) {
	t.Run("uploads file with correct headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			assert.Equal(t, "application/zip", r.Header.Get("Content-Type"))
			assert.Equal(t, int64(11), r.ContentLength)

			body, _ := io.ReadAll(r.Body)
			assert.Equal(t, "zip content", string(body))

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewHTTPClient("", "test-token")
		err := client.UploadFile(context.Background(), UploadFileRequest{
			URL:           server.URL,
			Method:        http.MethodPut,
			Headers:       map[string]string{"Content-Type": "application/zip"},
			Body:          strings.NewReader("zip content"),
			ContentLength: 11,
		})
		require.NoError(t, err)
	})

	t.Run("handles upload failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("URL expired"))
		}))
		defer server.Close()

		client := NewHTTPClient("", "test-token")
		err := client.UploadFile(context.Background(), UploadFileRequest{
			URL:           server.URL,
			Method:        http.MethodPut,
			Body:          strings.NewReader("data"),
			ContentLength: 4,
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "403")
	})
}

func TestHTTPClientGetPackageStatus(t *testing.T) {
	t.Run("returns status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedPath := "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789/status"
			assert.Equal(t, expectedPath, r.URL.Path)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"package_id":"pkg-789","status":"done","status_reason":""}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		status, err := client.GetPackageStatus(context.Background(), "app-123", "dep-456", "pkg-789")
		require.NoError(t, err)

		assert.Equal(t, "pkg-789", status.PackageID)
		assert.Equal(t, "done", status.Status)
	})

	t.Run("returns failed status with reason", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"package_id":"pkg-789","status":"failed","status_reason":"invalid bundle format"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		status, err := client.GetPackageStatus(context.Background(), "app-123", "dep-456", "pkg-789")
		require.NoError(t, err)

		assert.Equal(t, "failed", status.Status)
		assert.Equal(t, "invalid bundle format", status.StatusReason)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetPackageStatus(context.Background(), "app-123", "dep-456", "pkg-789")
		require.Error(t, err)
		assert.ErrorContains(t, err, "500")
	})
}

func TestHTTPClientListPackages(t *testing.T) {
	t.Run("returns packages", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-456/packages", r.URL.Path)
			assert.Equal(t, "test-token", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[{"id":"pkg-1","label":"v1","app_version":"1.0.0"},{"id":"pkg-2","label":"v2","app_version":"2.0.0"}]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		packages, err := client.ListPackages(context.Background(), "app-123", "dep-456")
		require.NoError(t, err)

		require.Len(t, packages, 2)
		assert.Equal(t, "pkg-1", packages[0].ID)
		assert.Equal(t, "v1", packages[0].Label)
	})

	t.Run("handles empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[]}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		packages, err := client.ListPackages(context.Background(), "app-123", "dep-456")
		require.NoError(t, err)
		assert.Empty(t, packages)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"deployment not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.ListPackages(context.Background(), "app-123", "dep-456")
		require.Error(t, err)
		assert.ErrorContains(t, err, "404")
	})
}

func TestHTTPClientGetPackage(t *testing.T) {
	t.Run("returns package", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "test-token", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-789","label":"v3","app_version":"1.0.0","mandatory":true,"rollout":50}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		pkg, err := client.GetPackage(context.Background(), "app-123", "dep-456", "pkg-789")
		require.NoError(t, err)

		assert.Equal(t, "pkg-789", pkg.ID)
		assert.Equal(t, "v3", pkg.Label)
		assert.True(t, pkg.Mandatory)
		assert.Equal(t, float64(50), pkg.Rollout)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"package not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.GetPackage(context.Background(), "app-123", "dep-456", "pkg-789")
		require.Error(t, err)
		assert.ErrorContains(t, err, "404")
	})
}

func TestHTTPClientPatchPackage(t *testing.T) {
	t.Run("sends correct PATCH request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789", r.URL.Path)
			assert.Equal(t, http.MethodPatch, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body map[string]interface{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, float64(50), body["rollout"])
			assert.Equal(t, true, body["mandatory"])

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-789","label":"v3","app_version":"1.0.0","mandatory":true,"rollout":50}`))
		}))
		defer server.Close()

		rollout := 50
		mandatory := true
		client := NewHTTPClient(server.URL, "test-token")
		pkg, err := client.PatchPackage(context.Background(), "app-123", "dep-456", "pkg-789", PatchRequest{
			Rollout:   &rollout,
			Mandatory: &mandatory,
		})
		require.NoError(t, err)

		assert.Equal(t, "pkg-789", pkg.ID)
		assert.Equal(t, float64(50), pkg.Rollout)
	})

	t.Run("omits nil fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)
			assert.NotContains(t, bodyStr, "mandatory")
			assert.NotContains(t, bodyStr, "disabled")
			assert.Contains(t, bodyStr, "rollout")

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-789","label":"v3","rollout":50}`))
		}))
		defer server.Close()

		rollout := 50
		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.PatchPackage(context.Background(), "app-123", "dep-456", "pkg-789", PatchRequest{
			Rollout: &rollout,
		})
		require.NoError(t, err)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid rollout value"}`))
		}))
		defer server.Close()

		rollout := 50
		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.PatchPackage(context.Background(), "app-123", "dep-456", "pkg-789", PatchRequest{
			Rollout: &rollout,
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "400")
	})
}

func TestHTTPClientDeletePackage(t *testing.T) {
	t.Run("deletes package", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-456/packages/pkg-789", r.URL.Path)
			assert.Equal(t, http.MethodDelete, r.Method)

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		err := client.DeletePackage(context.Background(), "app-123", "dep-456", "pkg-789")
		require.NoError(t, err)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"package not found"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		err := client.DeletePackage(context.Background(), "app-123", "dep-456", "pkg-789")
		require.Error(t, err)
		assert.ErrorContains(t, err, "404")
	})
}

func TestHTTPClientRollback(t *testing.T) {
	t.Run("sends correct request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-456/rollback", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "test-token", r.Header.Get("Authorization"))

			var body RollbackRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "pkg-target", body.PackageID)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-new","label":"v4","app_version":"1.0.0"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		pkg, err := client.Rollback(context.Background(), "app-123", "dep-456", RollbackRequest{PackageID: "pkg-target"})
		require.NoError(t, err)

		assert.Equal(t, "pkg-new", pkg.ID)
		assert.Equal(t, "v4", pkg.Label)
	})

	t.Run("omits empty package_id", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			assert.NotContains(t, string(body), "package_id")

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-new","label":"v2"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.Rollback(context.Background(), "app-123", "dep-456", RollbackRequest{})
		require.NoError(t, err)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"no releases"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.Rollback(context.Background(), "app-123", "dep-456", RollbackRequest{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "404")
	})
}

func TestHTTPClientPromote(t *testing.T) {
	t.Run("sends correct request with all fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/connected-apps/app-123/code-push/deployments/dep-src/promote", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var body PromoteRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "dep-dst", body.TargetDeploymentID)
			assert.Equal(t, "3.0.0", body.AppVersion)
			assert.Equal(t, "true", body.Mandatory)
			assert.Equal(t, "50", body.Rollout)

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-promoted","label":"v1","app_version":"3.0.0"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		pkg, err := client.Promote(context.Background(), "app-123", "dep-src", PromoteRequest{
			TargetDeploymentID: "dep-dst",
			AppVersion:         "3.0.0",
			Mandatory:          "true",
			Rollout:            "50",
		})
		require.NoError(t, err)

		assert.Equal(t, "pkg-promoted", pkg.ID)
	})

	t.Run("omits empty optional fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)
			assert.NotContains(t, bodyStr, "package_id")
			assert.NotContains(t, bodyStr, "app_version")
			assert.NotContains(t, bodyStr, "mandatory")

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"pkg-new","label":"v1"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.Promote(context.Background(), "app-123", "dep-src", PromoteRequest{
			TargetDeploymentID: "dep-dst",
		})
		require.NoError(t, err)
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`{"error":"duplicate release"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(server.URL, "test-token")
		_, err := client.Promote(context.Background(), "app-123", "dep-src", PromoteRequest{TargetDeploymentID: "dep-dst"})
		require.Error(t, err)
		assert.ErrorContains(t, err, "409")
	})
}
