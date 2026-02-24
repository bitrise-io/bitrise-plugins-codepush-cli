package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err)
		assert.Empty(t, token)
	})

	t.Run("returns error for malformed JSON", func(t *testing.T) {
		dir := setupTestDir(t)

		require.NoError(t, os.WriteFile(filepath.Join(dir, configFileName), []byte("not json"), 0o600))

		_, err := LoadToken()
		require.Error(t, err)
		assert.ErrorContains(t, err, "decoding config file")
	})
}

func TestSaveToken(t *testing.T) {
	t.Run("save and load round-trip", func(t *testing.T) {
		setupTestDir(t)

		require.NoError(t, SaveToken("my-secret-token"))

		token, err := LoadToken()
		require.NoError(t, err)
		assert.Equal(t, "my-secret-token", token)
	})

	t.Run("overwrites previous token", func(t *testing.T) {
		setupTestDir(t)

		require.NoError(t, SaveToken("first-token"))
		require.NoError(t, SaveToken("second-token"))

		token, err := LoadToken()
		require.NoError(t, err)
		assert.Equal(t, "second-token", token)
	})

	t.Run("creates config directory", func(t *testing.T) {
		base := t.TempDir()
		nested := filepath.Join(base, "nested", "config")
		configDirFunc = func() (string, error) { return nested, nil }
		t.Cleanup(func() { configDirFunc = defaultConfigDir })

		require.NoError(t, SaveToken("token"))

		info, err := os.Stat(nested)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("config file has restricted permissions", func(t *testing.T) {
		dir := setupTestDir(t)

		require.NoError(t, SaveToken("token"))

		info, err := os.Stat(filepath.Join(dir, configFileName))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("config directory has restricted permissions", func(t *testing.T) {
		base := t.TempDir()
		dir := filepath.Join(base, "newdir")
		configDirFunc = func() (string, error) { return dir, nil }
		t.Cleanup(func() { configDirFunc = defaultConfigDir })

		require.NoError(t, SaveToken("token"))

		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	})
}

func TestRemoveToken(t *testing.T) {
	t.Run("removes existing config file", func(t *testing.T) {
		dir := setupTestDir(t)

		require.NoError(t, SaveToken("token-to-remove"))
		require.NoError(t, RemoveToken())

		_, err := os.Stat(filepath.Join(dir, configFileName))
		assert.True(t, os.IsNotExist(err))

		token, err := LoadToken()
		require.NoError(t, err)
		assert.Empty(t, token)
	})

	t.Run("no error when file does not exist", func(t *testing.T) {
		setupTestDir(t)

		require.NoError(t, RemoveToken())
	})
}

func TestConfigFilePath(t *testing.T) {
	dir := setupTestDir(t)

	path, err := ConfigFilePath()
	require.NoError(t, err)

	expected := filepath.Join(dir, configFileName)
	assert.Equal(t, expected, path)
}

func TestConfigDirError(t *testing.T) {
	configDirFunc = func() (string, error) { return "", fmt.Errorf("no home dir") }
	t.Cleanup(func() { configDirFunc = defaultConfigDir })

	_, err := LoadToken()
	require.Error(t, err)

	err = SaveToken("token")
	require.Error(t, err)

	err = RemoveToken()
	require.Error(t, err)

	_, err = ConfigFilePath()
	require.Error(t, err)
}

func TestValidateToken(t *testing.T) {
	t.Run("valid token returns user info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "valid-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"username":"testuser","email":"test@example.com"}}`))
		}))
		defer server.Close()

		userInfo, err := validateTokenWithURL("valid-token", server.URL, &http.Client{})
		require.NoError(t, err)
		require.NotNil(t, userInfo)
		assert.Equal(t, "testuser", userInfo.Username)
		assert.Equal(t, "test@example.com", userInfo.Email)
	})

	t.Run("valid token with unparseable body returns nil user info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`not json`))
		}))
		defer server.Close()

		userInfo, err := validateTokenWithURL("valid-token", server.URL, &http.Client{})
		require.NoError(t, err)
		assert.Nil(t, userInfo)
	})

	t.Run("invalid token returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
		}))
		defer server.Close()

		_, err := validateTokenWithURL("bad-token", server.URL, &http.Client{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid token")
	})

	t.Run("server error returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := validateTokenWithURL("some-token", server.URL, &http.Client{})
		require.Error(t, err)
		assert.ErrorContains(t, err, "500")
	})
}

func TestTokenGenerationURL(t *testing.T) {
	assert.NotEmpty(t, TokenGenerationURL)
	assert.Contains(t, TokenGenerationURL, "bitrise.io")
}
