package bundler

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOSTriplet returns the hermesc directory name for the current OS/arch,
// matching the logic in findHermesc.
func testOSTriplet() string {
	switch {
	case runtime.GOOS == "darwin":
		return "osx-bin"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "linux64-bin"
	default:
		return runtime.GOOS + "-bin"
	}
}

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
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
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
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				writeFile(t, path, content)
			}

			got, err := detectEntryFile(dir, tt.platform)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectHermesAndroid(t *testing.T) {
	tests := []struct {
		name   string
		gradle string
		want   hermesDetection
	}{
		{
			name:   "hermesEnabled = true",
			gradle: `android { react { hermesEnabled = true } }`,
			want:   hermesEnabled,
		},
		{
			name:   "hermesEnabled.set(true)",
			gradle: `react { hermesEnabled.set(true) }`,
			want:   hermesEnabled,
		},
		{
			name:   "enableHermes: true",
			gradle: `project.ext.react = [ enableHermes: true ]`,
			want:   hermesEnabled,
		},
		{
			name:   "hermesEnabled = false",
			gradle: `react { hermesEnabled = false }`,
			want:   hermesDisabled,
		},
		{
			name:   "no hermes config",
			gradle: `android { defaultConfig {} }`,
			want:   hermesNotFound,
		},
		{
			name: "no gradle file",
			want: hermesNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if tt.gradle != "" {
				gradleDir := filepath.Join(dir, "android", "app")
				require.NoError(t, os.MkdirAll(gradleDir, 0o755))
				writeFile(t, filepath.Join(gradleDir, "build.gradle"), tt.gradle)
			}

			assert.Equal(t, tt.want, detectHermesAndroid(dir))
		})
	}
}

func TestDetectHermesIOS(t *testing.T) {
	tests := []struct {
		name    string
		podfile string
		want    hermesDetection
	}{
		{
			name:    "hermes enabled ruby hash syntax",
			podfile: `use_react_native!(:hermes_enabled => true)`,
			want:    hermesEnabled,
		},
		{
			name:    "hermes enabled new syntax",
			podfile: `use_react_native!(hermes_enabled: true)`,
			want:    hermesEnabled,
		},
		{
			name:    "hermes disabled",
			podfile: `:hermes_enabled => false`,
			want:    hermesDisabled,
		},
		{
			name:    "no hermes config",
			podfile: `platform :ios, '13.0'`,
			want:    hermesNotFound,
		},
		{
			name: "no podfile",
			want: hermesNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if tt.podfile != "" {
				iosDir := filepath.Join(dir, "ios")
				require.NoError(t, os.MkdirAll(iosDir, 0o755))
				writeFile(t, filepath.Join(iosDir, "Podfile"), tt.podfile)
			}

			assert.Equal(t, tt.want, detectHermesIOS(dir))
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
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, filepath.Join(dir, tt.want), got)
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
		require.NoError(t, os.MkdirAll(gradleDir, 0o755))
		writeFile(t, filepath.Join(gradleDir, "build.gradle"), `react { hermesEnabled = true }`)

		config, err := DetectProject(dir, PlatformAndroid, HermesModeAuto)
		require.NoError(t, err)

		assert.Equal(t, ProjectTypeReactNative, config.ProjectType)
		assert.Equal(t, "index.js", config.EntryFile)
		assert.True(t, config.HermesEnabled)
		assert.NotEmpty(t, config.MetroConfig)
	})

	t.Run("hermes mode override to off", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		gradleDir := filepath.Join(dir, "android", "app")
		require.NoError(t, os.MkdirAll(gradleDir, 0o755))
		writeFile(t, filepath.Join(gradleDir, "build.gradle"), `react { hermesEnabled = true }`)

		config, err := DetectProject(dir, PlatformAndroid, HermesModeOff)
		require.NoError(t, err)

		assert.False(t, config.HermesEnabled)
	})

	t.Run("hermes mode override to on", func(t *testing.T) {
		dir := t.TempDir()

		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		writeFile(t, filepath.Join(dir, "index.js"), "")

		config, err := DetectProject(dir, PlatformIOS, HermesModeOn)
		require.NoError(t, err)

		assert.True(t, config.HermesEnabled)
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := DetectProject("/nonexistent/path", PlatformIOS, HermesModeAuto)
		require.Error(t, err)
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
		assert.Equal(t, tt.want, tt.pt.String())
	}
}

func TestParseRNMajorMinor(t *testing.T) {
	tests := []struct {
		version string
		want    int
	}{
		{"0.72.0", 72},
		{"0.70.0", 70},
		{"0.69.12", 69},
		{"^0.72.0", 72},
		{"~0.71.3", 71},
		{">=0.70.0", 70},
		{"0.68.0", 68},
		{"1.0.0", 100},
		{"invalid", 0},
		{"", 0},
		{"0", 0},
		{"0.abc.0", 0},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, parseRNMajorMinor(tt.version), "parseRNMajorMinor(%q)", tt.version)
	}
}

func TestIsHermesDefaultVersion(t *testing.T) {
	t.Run("RN 0.72 defaults to hermes", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)
		assert.True(t, isHermesDefaultVersion(dir))
	})

	t.Run("RN 0.70 defaults to hermes", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.70.0"}}`)
		assert.True(t, isHermesDefaultVersion(dir))
	})

	t.Run("RN 0.69 does not default to hermes", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.69.0"}}`)
		assert.False(t, isHermesDefaultVersion(dir))
	})

	t.Run("semver range prefix", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "^0.73.0"}}`)
		assert.True(t, isHermesDefaultVersion(dir))
	})

	t.Run("no package.json", func(t *testing.T) {
		dir := t.TempDir()
		assert.False(t, isHermesDefaultVersion(dir))
	})

	t.Run("no react-native dependency", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"expo": "~49.0.0"}}`)
		assert.False(t, isHermesDefaultVersion(dir))
	})

	t.Run("react-native in devDependencies", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"devDependencies": {"react-native": "0.72.0"}}`)
		assert.True(t, isHermesDefaultVersion(dir))
	})
}

func TestDetectHermesVersionFallback(t *testing.T) {
	t.Run("no config file but RN >= 0.70 enables hermes", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)

		enabled, err := detectHermes(dir, PlatformAndroid)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("no config file and RN < 0.70 disables hermes", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.68.0"}}`)

		enabled, err := detectHermes(dir, PlatformAndroid)
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("explicit disable overrides version fallback", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "package.json"), `{"dependencies": {"react-native": "0.72.0"}}`)

		gradleDir := filepath.Join(dir, "android", "app")
		require.NoError(t, os.MkdirAll(gradleDir, 0o755))
		writeFile(t, filepath.Join(gradleDir, "build.gradle"), `react { hermesEnabled = false }`)

		enabled, err := detectHermes(dir, PlatformAndroid)
		require.NoError(t, err)
		assert.False(t, enabled)
	})
}

func TestFindHermesc(t *testing.T) {
	osTriplet := testOSTriplet()

	t.Run("finds hermesc in hermes-engine", func(t *testing.T) {
		dir := t.TempDir()

		hermescDir := filepath.Join(dir, "node_modules", "hermes-engine", osTriplet)
		os.MkdirAll(hermescDir, 0o755)
		writeFile(t, filepath.Join(hermescDir, "hermesc"), "#!/bin/sh")

		path, err := findHermesc(dir)
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(path))
	})

	t.Run("finds hermesc in react-native sdks", func(t *testing.T) {
		dir := t.TempDir()

		hermescDir := filepath.Join(dir, "node_modules", "react-native", "sdks", "hermesc", osTriplet)
		os.MkdirAll(hermescDir, 0o755)
		writeFile(t, filepath.Join(hermescDir, "hermesc"), "#!/bin/sh")

		path, err := findHermesc(dir)
		require.NoError(t, err)
		assert.NotEmpty(t, path)
	})

	t.Run("prefers hermes-engine over react-native", func(t *testing.T) {
		dir := t.TempDir()

		loc1 := filepath.Join(dir, "node_modules", "hermes-engine", osTriplet)
		os.MkdirAll(loc1, 0o755)
		writeFile(t, filepath.Join(loc1, "hermesc"), "primary")

		loc2 := filepath.Join(dir, "node_modules", "react-native", "sdks", "hermesc", osTriplet)
		os.MkdirAll(loc2, 0o755)
		writeFile(t, filepath.Join(loc2, "hermesc"), "secondary")

		path, err := findHermesc(dir)
		require.NoError(t, err)

		// Should prefer hermes-engine location
		assert.Contains(t, path, "hermes-engine")
	})

	t.Run("returns error when not found", func(t *testing.T) {
		dir := t.TempDir()

		_, err := findHermesc(dir)
		require.Error(t, err)
		assert.ErrorContains(t, err, "hermesc binary not found")
	})
}

func TestFindExpoBundleOutput(t *testing.T) {
	t.Run("finds bundle in bundles directory", func(t *testing.T) {
		dir := t.TempDir()
		bundleDir := filepath.Join(dir, "bundles")
		os.MkdirAll(bundleDir, 0o755)
		writeFile(t, filepath.Join(bundleDir, "ios-abc123.js"), "bundle content")

		path, err := findExpoBundleOutput(dir, PlatformIOS)
		require.NoError(t, err)
		assert.NotEmpty(t, path)
	})

	t.Run("finds bundle in _expo/static/js directory", func(t *testing.T) {
		dir := t.TempDir()
		jsDir := filepath.Join(dir, "_expo", "static", "js", "ios")
		os.MkdirAll(jsDir, 0o755)
		writeFile(t, filepath.Join(jsDir, "entry-abc123.js"), "bundle content")

		path, err := findExpoBundleOutput(dir, PlatformIOS)
		require.NoError(t, err)
		assert.NotEmpty(t, path)
	})

	t.Run("single js file fallback succeeds", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "bundle.js"), "bundle content")

		path, err := findExpoBundleOutput(dir, PlatformIOS)
		require.NoError(t, err)
		assert.NotEmpty(t, path)
	})

	t.Run("multiple js files in fallback returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "one.js"), "content")
		writeFile(t, filepath.Join(dir, "two.js"), "content")

		_, err := findExpoBundleOutput(dir, PlatformIOS)
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not determine which is the bundle")
	})

	t.Run("empty directory returns error", func(t *testing.T) {
		dir := t.TempDir()

		_, err := findExpoBundleOutput(dir, PlatformIOS)
		require.Error(t, err)
	})
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
