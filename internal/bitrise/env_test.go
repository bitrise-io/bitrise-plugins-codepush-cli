package bitrise

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBitriseEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    bool
	}{
		{
			name:    "not bitrise environment",
			envVars: map[string]string{},
			want:    false,
		},
		{
			name:    "build number set",
			envVars: map[string]string{"BITRISE_BUILD_NUMBER": "42"},
			want:    true,
		},
		{
			name:    "deploy dir set",
			envVars: map[string]string{"BITRISE_DEPLOY_DIR": "/tmp/deploy"},
			want:    true,
		},
		{
			name: "both set",
			envVars: map[string]string{
				"BITRISE_BUILD_NUMBER": "42",
				"BITRISE_DEPLOY_DIR":   "/tmp/deploy",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			t.Setenv("BITRISE_BUILD_NUMBER", "")
			t.Setenv("BITRISE_DEPLOY_DIR", "")

			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			got := IsBitriseEnvironment()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetBuildMetadata(t *testing.T) {
	t.Setenv("BITRISE_DEPLOY_DIR", "/tmp/deploy")
	t.Setenv("BITRISE_BUILD_NUMBER", "42")
	t.Setenv("GIT_CLONE_COMMIT_HASH", "abc123")

	meta := GetBuildMetadata()

	assert.Equal(t, "/tmp/deploy", meta.DeployDir)
	assert.Equal(t, "42", meta.BuildNumber)
	assert.Equal(t, "abc123", meta.CommitHash)
}

func TestWriteToDeployDir(t *testing.T) {
	t.Run("writes file successfully", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("BITRISE_DEPLOY_DIR", dir)

		path, err := WriteToDeployDir("test.json", []byte(`{"key": "value"}`))
		require.NoError(t, err)

		expected := filepath.Join(dir, "test.json")
		assert.Equal(t, expected, path)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, `{"key": "value"}`, string(data))
	})

	t.Run("creates deploy directory if missing", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nested", "deploy")
		t.Setenv("BITRISE_DEPLOY_DIR", dir)

		path, err := WriteToDeployDir("test.txt", []byte("hello"))
		require.NoError(t, err)

		_, err = os.Stat(path)
		assert.NoError(t, err)
	})

	t.Run("error when deploy dir not set", func(t *testing.T) {
		t.Setenv("BITRISE_DEPLOY_DIR", "")

		_, err := WriteToDeployDir("test.txt", []byte("hello"))
		require.Error(t, err)
	})
}

func TestExportEnvVar(t *testing.T) {
	t.Run("skips silently when envman not on PATH", func(t *testing.T) {
		// Use a PATH that definitely doesn't contain envman
		t.Setenv("PATH", t.TempDir())

		err := ExportEnvVar("TEST_KEY", "test_value")
		require.NoError(t, err)
	})

	t.Run("calls envman when available", func(t *testing.T) {
		_, err := exec.LookPath("envman")
		if err != nil {
			t.Skip("envman not available on this system")
		}

		err = ExportEnvVar("TEST_KEY", "test_value")
		require.NoError(t, err)
	})
}
