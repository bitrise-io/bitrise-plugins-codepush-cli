package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	configDirFunc = func() (string, error) { return dir, nil }
	t.Cleanup(func() { configDirFunc = defaultConfigDir })
	return dir
}

func TestLoadToken(t *testing.T) {
	t.Run("returns empty when no config file exists", func(t *testing.T) {
		setupTestDir(t)

		token, err := LoadToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "" {
			t.Errorf("token: got %q, want empty", token)
		}
	})

	t.Run("returns error for malformed JSON", func(t *testing.T) {
		dir := setupTestDir(t)

		if err := os.WriteFile(filepath.Join(dir, configFileName), []byte("not json"), 0o600); err != nil {
			t.Fatalf("writing file: %v", err)
		}

		_, err := LoadToken()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "decoding config file") {
			t.Errorf("error should mention decoding: %v", err)
		}
	})
}

func TestSaveToken(t *testing.T) {
	t.Run("save and load round-trip", func(t *testing.T) {
		setupTestDir(t)

		if err := SaveToken("my-secret-token"); err != nil {
			t.Fatalf("SaveToken: %v", err)
		}

		token, err := LoadToken()
		if err != nil {
			t.Fatalf("LoadToken: %v", err)
		}
		if token != "my-secret-token" {
			t.Errorf("token: got %q, want %q", token, "my-secret-token")
		}
	})

	t.Run("overwrites previous token", func(t *testing.T) {
		setupTestDir(t)

		if err := SaveToken("first-token"); err != nil {
			t.Fatalf("SaveToken(first): %v", err)
		}
		if err := SaveToken("second-token"); err != nil {
			t.Fatalf("SaveToken(second): %v", err)
		}

		token, err := LoadToken()
		if err != nil {
			t.Fatalf("LoadToken: %v", err)
		}
		if token != "second-token" {
			t.Errorf("token: got %q, want %q", token, "second-token")
		}
	})

	t.Run("creates config directory", func(t *testing.T) {
		base := t.TempDir()
		nested := filepath.Join(base, "nested", "config")
		configDirFunc = func() (string, error) { return nested, nil }
		t.Cleanup(func() { configDirFunc = defaultConfigDir })

		if err := SaveToken("token"); err != nil {
			t.Fatalf("SaveToken: %v", err)
		}

		info, err := os.Stat(nested)
		if err != nil {
			t.Fatalf("config dir not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("config path is not a directory")
		}
	})

	t.Run("config file has restricted permissions", func(t *testing.T) {
		dir := setupTestDir(t)

		if err := SaveToken("token"); err != nil {
			t.Fatalf("SaveToken: %v", err)
		}

		info, err := os.Stat(filepath.Join(dir, configFileName))
		if err != nil {
			t.Fatalf("stat config file: %v", err)
		}
		perm := info.Mode().Perm()
		if perm != 0o600 {
			t.Errorf("file permissions: got %o, want 600", perm)
		}
	})

	t.Run("config directory has restricted permissions", func(t *testing.T) {
		base := t.TempDir()
		dir := filepath.Join(base, "newdir")
		configDirFunc = func() (string, error) { return dir, nil }
		t.Cleanup(func() { configDirFunc = defaultConfigDir })

		if err := SaveToken("token"); err != nil {
			t.Fatalf("SaveToken: %v", err)
		}

		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat config dir: %v", err)
		}
		perm := info.Mode().Perm()
		if perm != 0o700 {
			t.Errorf("directory permissions: got %o, want 700", perm)
		}
	})
}

func TestRemoveToken(t *testing.T) {
	t.Run("removes existing config file", func(t *testing.T) {
		dir := setupTestDir(t)

		if err := SaveToken("token-to-remove"); err != nil {
			t.Fatalf("SaveToken: %v", err)
		}

		if err := RemoveToken(); err != nil {
			t.Fatalf("RemoveToken: %v", err)
		}

		if _, err := os.Stat(filepath.Join(dir, configFileName)); !os.IsNotExist(err) {
			t.Error("config file should not exist after RemoveToken")
		}

		token, err := LoadToken()
		if err != nil {
			t.Fatalf("LoadToken: %v", err)
		}
		if token != "" {
			t.Errorf("token should be empty after remove, got %q", token)
		}
	})

	t.Run("no error when file does not exist", func(t *testing.T) {
		setupTestDir(t)

		if err := RemoveToken(); err != nil {
			t.Fatalf("RemoveToken on missing file: %v", err)
		}
	})
}

func TestConfigFilePath(t *testing.T) {
	dir := setupTestDir(t)

	path, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(dir, configFileName)
	if path != expected {
		t.Errorf("path: got %q, want %q", path, expected)
	}
}

func TestConfigDirError(t *testing.T) {
	configDirFunc = func() (string, error) { return "", fmt.Errorf("no home dir") }
	t.Cleanup(func() { configDirFunc = defaultConfigDir })

	_, err := LoadToken()
	if err == nil {
		t.Fatal("expected error from LoadToken, got nil")
	}

	err = SaveToken("token")
	if err == nil {
		t.Fatal("expected error from SaveToken, got nil")
	}

	err = RemoveToken()
	if err == nil {
		t.Fatal("expected error from RemoveToken, got nil")
	}

	_, err = ConfigFilePath()
	if err == nil {
		t.Fatal("expected error from ConfigFilePath, got nil")
	}
}

func TestValidateToken(t *testing.T) {
	t.Run("valid token returns user info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "valid-token" {
				t.Errorf("auth header: got %q, want plain token without Bearer prefix", got)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"username":"testuser","email":"test@example.com"}}`))
		}))
		defer server.Close()

		userInfo, err := validateTokenWithURL("valid-token", server.URL, &http.Client{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if userInfo == nil {
			t.Fatal("expected user info, got nil")
		}
		if userInfo.Username != "testuser" {
			t.Errorf("username: got %q, want %q", userInfo.Username, "testuser")
		}
		if userInfo.Email != "test@example.com" {
			t.Errorf("email: got %q, want %q", userInfo.Email, "test@example.com")
		}
	})

	t.Run("valid token with unparseable body returns nil user info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`not json`))
		}))
		defer server.Close()

		userInfo, err := validateTokenWithURL("valid-token", server.URL, &http.Client{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if userInfo != nil {
			t.Errorf("expected nil user info for unparseable body, got %+v", userInfo)
		}
	})

	t.Run("invalid token returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
		}))
		defer server.Close()

		_, err := validateTokenWithURL("bad-token", server.URL, &http.Client{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid token") {
			t.Errorf("error should mention invalid token: %v", err)
		}
	})

	t.Run("server error returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := validateTokenWithURL("some-token", server.URL, &http.Client{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error should contain status code: %v", err)
		}
	})
}

func TestTokenGenerationURL(t *testing.T) {
	if TokenGenerationURL == "" {
		t.Error("TokenGenerationURL should not be empty")
	}
	if !strings.Contains(TokenGenerationURL, "bitrise.io") {
		t.Errorf("TokenGenerationURL should point to bitrise.io: %s", TokenGenerationURL)
	}
}
