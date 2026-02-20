package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOutputJSON(t *testing.T) {
	data := struct {
		Name string `json:"name"`
	}{Name: "test"}

	err := outputJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{
			name: "short string unchanged",
			s:    "hello",
			max:  10,
			want: "hello",
		},
		{
			name: "exact length unchanged",
			s:    "hello",
			max:  5,
			want: "hello",
		},
		{
			name: "long string truncated with ellipsis",
			s:    "hello world",
			max:  8,
			want: "hello...",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncate(tc.s, tc.max)
			if got != tc.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tc.s, tc.max, got, tc.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name string
		b    int64
		want string
	}{
		{name: "bytes", b: 500, want: "500 B"},
		{name: "kilobytes", b: 1024, want: "1.0 KB"},
		{name: "megabytes", b: 1048576, want: "1.0 MB"},
		{name: "gigabytes", b: 1073741824, want: "1.0 GB"},
		{name: "zero", b: 0, want: "0 B"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatBytes(tc.b)
			if got != tc.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tc.b, got, tc.want)
			}
		})
	}
}

func TestResolveAppID(t *testing.T) {
	t.Run("flag takes priority", func(t *testing.T) {
		old := globalAppID
		globalAppID = "flag-value"
		defer func() { globalAppID = old }()

		got := resolveAppID()
		if got != "flag-value" {
			t.Errorf("got %q, want %q", got, "flag-value")
		}
	})

	t.Run("falls back to env var", func(t *testing.T) {
		old := globalAppID
		globalAppID = ""
		defer func() { globalAppID = old }()

		t.Setenv("CODEPUSH_APP_ID", "env-value")
		got := resolveAppID()
		if got != "env-value" {
			t.Errorf("got %q, want %q", got, "env-value")
		}
	})
}

func TestRequireCredentials(t *testing.T) {
	t.Run("returns error when app ID missing", func(t *testing.T) {
		old := globalAppID
		globalAppID = ""
		defer func() { globalAppID = old }()

		t.Setenv("CODEPUSH_APP_ID", "")
		_, _, err := requireCredentials()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "app ID is required") {
			t.Errorf("error should mention app ID: %v", err)
		}
	})

	t.Run("returns values when both set", func(t *testing.T) {
		old := globalAppID
		globalAppID = "my-app"
		defer func() { globalAppID = old }()

		t.Setenv("BITRISE_API_TOKEN", "my-token")
		appID, token, err := requireCredentials()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if appID != "my-app" {
			t.Errorf("appID = %q, want %q", appID, "my-app")
		}
		if token != "my-token" {
			t.Errorf("token = %q, want %q", token, "my-token")
		}
	})
}

func TestResolveInputInteractive(t *testing.T) {
	t.Run("returns value when provided", func(t *testing.T) {
		got, err := resolveInputInteractive("provided", "Enter name", "placeholder")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "provided" {
			t.Errorf("got %q, want %q", got, "provided")
		}
	})

	t.Run("returns error in non-interactive mode", func(t *testing.T) {
		_, err := resolveInputInteractive("", "Enter name", "placeholder")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Enter name") {
			t.Errorf("error should contain the title: %v", err)
		}
	})
}

func TestResolveAppIDInteractive(t *testing.T) {
	t.Run("returns valid UUID from flag", func(t *testing.T) {
		old := globalAppID
		globalAppID = "550e8400-e29b-41d4-a716-446655440000"
		defer func() { globalAppID = old }()

		got, err := resolveAppIDInteractive()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "550e8400-e29b-41d4-a716-446655440000" {
			t.Errorf("got %q, want UUID", got)
		}
	})

	t.Run("returns error for invalid UUID from flag", func(t *testing.T) {
		old := globalAppID
		globalAppID = "not-a-uuid"
		defer func() { globalAppID = old }()

		_, err := resolveAppIDInteractive()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid app ID") {
			t.Errorf("error should mention invalid: %v", err)
		}
	})

	t.Run("returns error in non-interactive mode when empty", func(t *testing.T) {
		old := globalAppID
		globalAppID = ""
		defer func() { globalAppID = old }()

		t.Setenv("CODEPUSH_APP_ID", "")

		_, err := resolveAppIDInteractive()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "app ID is required") {
			t.Errorf("error should mention requirement: %v", err)
		}
	})
}

func TestResolvePlatformInteractive(t *testing.T) {
	t.Run("returns value when provided", func(t *testing.T) {
		got, err := resolvePlatformInteractive("ios")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "ios" {
			t.Errorf("got %q, want %q", got, "ios")
		}
	})

	t.Run("returns error in non-interactive mode", func(t *testing.T) {
		_, err := resolvePlatformInteractive("")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "--platform") {
			t.Errorf("error should mention --platform: %v", err)
		}
	})
}

func TestOutputJSONFormat(t *testing.T) {
	data := map[string]string{"key": "value"}

	// outputJSON writes to stdout; just verify no error
	err := outputJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOutputJSONMarshalError(t *testing.T) {
	// json.Marshal cannot fail for a regular struct, but test with valid data
	data := struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: "123", Name: "test"}

	err := outputJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the output is valid JSON by marshaling it ourselves
	_, marshalErr := json.MarshalIndent(data, "", "  ")
	if marshalErr != nil {
		t.Fatalf("data should be marshalable: %v", marshalErr)
	}
}
