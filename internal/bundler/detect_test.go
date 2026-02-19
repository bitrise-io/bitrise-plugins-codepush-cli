package bundler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectType(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string
		want        ProjectType
		wantErr     bool
	}{
		{
			name:        "react native project",
			packageJSON: `{"dependencies": {"react-native": "0.72.0"}}`,
			want:        ProjectTypeReactNative,
		},
		{
			name:        "expo project",
			packageJSON: `{"dependencies": {"expo": "~49.0.0", "react-native": "0.72.0"}}`,
			want:        ProjectTypeExpo,
		},
		{
			name:        "expo in devDependencies",
			packageJSON: `{"devDependencies": {"expo": "~49.0.0"}}`,
			want:        ProjectTypeExpo,
		},
		{
			name:        "react native in devDependencies",
			packageJSON: `{"devDependencies": {"react-native": "0.72.0"}}`,
			want:        ProjectTypeReactNative,
		},
		{
			name:        "unknown project",
			packageJSON: `{"dependencies": {"express": "4.18.0"}}`,
			wantErr:     true,
		},
		{
			name:    "no package.json",
			wantErr: true,
		},
		{
			name:        "invalid json",
			packageJSON: `{invalid`,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if tt.packageJSON != "" {
				writeFile(t, filepath.Join(dir, "package.json"), tt.packageJSON)
			}

			got, err := detectProjectType(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectEntryFile(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		platform Platform
		want     string
		wantErr  bool
	}{
		{
			name:     "platform-specific entry file",
			files:    map[string]string{"index.ios.js": "", "index.js": ""},
			platform: PlatformIOS,
			want:     "index.ios.js",
		},
		{
			name:     "generic index.js",
			files:    map[string]string{"index.js": ""},
			platform: PlatformIOS,
			want:     "index.js",
		},
		{
			name:     "android platform-specific",
			files:    map[string]string{"index.android.js": "", "index.js": ""},
			platform: PlatformAndroid,
			want:     "index.android.js",
		},
		{
			name: "package.json main field fallback",
			files: map[string]string{
				"package.json": `{"main": "src/index.js"}`,
				"src/index.js": "",
			},
			platform: PlatformIOS,
			want:     "src/index.js",
		},
		{
			name:     "no entry file found",
			files:    map[string]string{"package.json": `{}`},
			platform: PlatformIOS,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			for name, content := range tt.files {
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, path, content)
			}

			got, err := detectEntryFile(dir, tt.platform)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectHermesAndroid(t *testing.T) {
	tests := []struct {
		name    string
		gradle  string
		want    bool
		wantErr bool
	}{
		{
			name:   "hermesEnabled = true",
			gradle: `android { react { hermesEnabled = true } }`,
			want:   true,
		},
		{
			name:   "hermesEnabled.set(true)",
			gradle: `react { hermesEnabled.set(true) }`,
			want:   true,
		},
		{
			name:   "enableHermes: true",
			gradle: `project.ext.react = [ enableHermes: true ]`,
			want:   true,
		},
		{
			name:   "hermesEnabled = false",
			gradle: `react { hermesEnabled = false }`,
			want:   false,
		},
		{
			name:   "no hermes config",
			gradle: `android { defaultConfig {} }`,
			want:   false,
		},
		{
			name: "no gradle file",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if tt.gradle != "" {
				gradleDir := filepath.Join(dir, "android", "app")
				if err := os.MkdirAll(gradleDir, 0o755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(gradleDir, "build.gradle"), tt.gradle)
			}

			got, err := detectHermesAndroid(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectHermesIOS(t *testing.T) {
	tests := []struct {
		name    string
		podfile string
		want    bool
	}{
		{
			name:    "hermes enabled ruby hash syntax",
			podfile: `use_react_native!(:hermes_enabled => true)`,
			want:    true,
		},
		{
			name:    "hermes enabled new syntax",
			podfile: `use_react_native!(hermes_enabled: true)`,
			want:    true,
		},
		{
			name:    "hermes disabled",
			podfile: `:hermes_enabled => false`,
			want:    false,
		},
		{
			name:    "no hermes config",
			podfile: `platform :ios, '13.0'`,
			want:    false,
		},
		{
			name: "no podfile",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if tt.podfile != "" {
				iosDir := filepath.Join(dir, "ios")
				if err := os.MkdirAll(iosDir, 0o755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, filepath.Join(iosDir, "Podfile"), tt.podfile)
			}

			got, _ := detectHermesIOS(dir)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectMetroConfig(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{
			name:  "metro.config.js found",
			files: []string{"metro.config.js"},
			want:  "metro.config.js",
		},
		{
			name:  "metro.config.ts found",
			files: []string{"metro.config.ts"},
			want:  "metro.config.ts",
		},
		{
			name:  "prefers .js over .ts",
			files: []string{"metro.config.js", "metro.config.ts"},
			want:  "metro.config.js",
		},
		{
			name: "no metro config",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			for _, f := range tt.files {
				writeFile(t, filepath.Join(dir, f), "")
			}

			got := detectMetroConfig(dir)
			if tt.want == "" {
				if got != "" {
					t.Errorf("expected empty, got %q", got)
				}
				return
			}
			expected := filepath.Join(dir, tt.want)
			if got != expected {
				t.Errorf("got %q, want %q", got, expected)
			}
		})
	}
}

func TestDetectProject(t *testing.T) {
	t.Run("full react native project detection", func(t *testing.T) {
		dir := t.TempDir()

		// Set up a minimal React Native project structure
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")
		writeFile(t, filepath.Join(dir, "metro.config.js"), "")

		gradleDir := filepath.Join(dir, "android", "app")
		if err := os.MkdirAll(gradleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(gradleDir, "build.gradle"), `react { hermesEnabled = true }`)

		config, err := DetectProject(dir, PlatformAndroid, HermesModeAuto)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.ProjectType != ProjectTypeReactNative {
			t.Errorf("project type: got %v, want %v", config.ProjectType, ProjectTypeReactNative)
		}
		if config.EntryFile != "index.js" {
			t.Errorf("entry file: got %q, want %q", config.EntryFile, "index.js")
		}
		if !config.HermesEnabled {
			t.Error("expected Hermes to be enabled")
		}
		if config.MetroConfig == "" {
			t.Error("expected metro config to be detected")
		}
	})

	t.Run("hermes mode override to off", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		gradleDir := filepath.Join(dir, "android", "app")
		if err := os.MkdirAll(gradleDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(gradleDir, "build.gradle"), `react { hermesEnabled = true }`)

		config, err := DetectProject(dir, PlatformAndroid, HermesModeOff)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if config.HermesEnabled {
			t.Error("expected Hermes to be disabled with HermesModeOff")
		}
	})

	t.Run("hermes mode override to on", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		config, err := DetectProject(dir, PlatformIOS, HermesModeOn)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !config.HermesEnabled {
			t.Error("expected Hermes to be enabled with HermesModeOn")
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := DetectProject("/nonexistent/path", PlatformIOS, HermesModeAuto)
		if err == nil {
			t.Fatal("expected error for nonexistent directory")
		}
	})
}

func TestProjectTypeString(t *testing.T) {
	tests := []struct {
		pt   ProjectType
		want string
	}{
		{ProjectTypeReactNative, "react-native"},
		{ProjectTypeExpo, "expo"},
		{ProjectTypeUnknown, "unknown"},
	}

	for _, tt := range tests {
		if got := tt.pt.String(); got != tt.want {
			t.Errorf("ProjectType(%d).String() = %q, want %q", tt.pt, got, tt.want)
		}
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
