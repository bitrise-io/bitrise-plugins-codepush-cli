package bitrise

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsBitriseEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		want        bool
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
			if got != tt.want {
				t.Errorf("IsBitriseEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetBuildMetadata(t *testing.T) {
	t.Setenv("BITRISE_DEPLOY_DIR", "/tmp/deploy")
	t.Setenv("BITRISE_BUILD_NUMBER", "42")
	t.Setenv("GIT_CLONE_COMMIT_HASH", "abc123")

	meta := GetBuildMetadata()

	if meta.DeployDir != "/tmp/deploy" {
		t.Errorf("DeployDir = %q, want %q", meta.DeployDir, "/tmp/deploy")
	}
	if meta.BuildNumber != "42" {
		t.Errorf("BuildNumber = %q, want %q", meta.BuildNumber, "42")
	}
	if meta.CommitHash != "abc123" {
		t.Errorf("CommitHash = %q, want %q", meta.CommitHash, "abc123")
	}
}

func TestWriteToDeployDir(t *testing.T) {
	t.Run("writes file successfully", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("BITRISE_DEPLOY_DIR", dir)

		path, err := WriteToDeployDir("test.json", []byte(`{"key": "value"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := filepath.Join(dir, "test.json")
		if path != expected {
			t.Errorf("path = %q, want %q", path, expected)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading file: %v", err)
		}
		if string(data) != `{"key": "value"}` {
			t.Errorf("content = %q, want %q", string(data), `{"key": "value"}`)
		}
	})

	t.Run("creates deploy directory if missing", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nested", "deploy")
		t.Setenv("BITRISE_DEPLOY_DIR", dir)

		path, err := WriteToDeployDir("test.txt", []byte("hello"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(path); err != nil {
			t.Errorf("file not created: %v", err)
		}
	})

	t.Run("error when deploy dir not set", func(t *testing.T) {
		t.Setenv("BITRISE_DEPLOY_DIR", "")

		_, err := WriteToDeployDir("test.txt", []byte("hello"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
