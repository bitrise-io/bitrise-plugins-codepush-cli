// Package bundler provides JavaScript bundle generation for React Native and Expo projects.
package bundler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ProjectType represents the detected project type.
type ProjectType int

const (
	// ProjectTypeUnknown indicates the project type could not be detected.
	ProjectTypeUnknown ProjectType = iota
	// ProjectTypeReactNative indicates a bare React Native project.
	ProjectTypeReactNative
	// ProjectTypeExpo indicates an Expo-managed project.
	ProjectTypeExpo
)

// String returns the display name of the project type.
func (p ProjectType) String() string {
	switch p {
	case ProjectTypeReactNative:
		return "react-native"
	case ProjectTypeExpo:
		return "expo"
	default:
		return "unknown"
	}
}

// Platform represents the target mobile platform.
type Platform string

const (
	// PlatformIOS targets iOS devices.
	PlatformIOS Platform = "ios"
	// PlatformAndroid targets Android devices.
	PlatformAndroid Platform = "android"
)

// HermesMode represents the Hermes override setting.
type HermesMode string

const (
	// HermesModeAuto detects Hermes configuration from the project.
	HermesModeAuto HermesMode = "auto"
	// HermesModeOn forces Hermes compilation.
	HermesModeOn HermesMode = "on"
	// HermesModeOff disables Hermes compilation.
	HermesModeOff HermesMode = "off"
)

// ProjectConfig holds the auto-detected project configuration.
type ProjectConfig struct {
	ProjectDir    string
	ProjectType   ProjectType
	Platform      Platform
	EntryFile     string
	MetroConfig   string
	HermesEnabled bool
	HermescPath   string
}

// packageJSON represents the relevant fields of a package.json file.
type packageJSON struct {
	Main            string            `json:"main"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// DetectProject inspects the project directory and returns a ProjectConfig.
func DetectProject(projectDir string, platform Platform, hermesMode HermesMode) (*ProjectConfig, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("resolving project directory: %w", err)
	}

	if _, err := os.Stat(absDir); err != nil {
		return nil, fmt.Errorf("project directory does not exist: %w", err)
	}

	projectType, err := detectProjectType(absDir)
	if err != nil {
		return nil, err
	}

	entryFile, err := detectEntryFile(absDir, platform)
	if err != nil {
		return nil, err
	}

	hermesEnabled := false
	hermescPath := ""

	switch hermesMode {
	case HermesModeOn:
		hermesEnabled = true
	case HermesModeOff:
		hermesEnabled = false
	default:
		hermesEnabled, _ = detectHermes(absDir, platform)
	}

	if hermesEnabled {
		hermescPath, _ = findHermesc(absDir)
	}

	metroConfig := detectMetroConfig(absDir)

	return &ProjectConfig{
		ProjectDir:    absDir,
		ProjectType:   projectType,
		Platform:      platform,
		EntryFile:     entryFile,
		MetroConfig:   metroConfig,
		HermesEnabled: hermesEnabled,
		HermescPath:   hermescPath,
	}, nil
}

// detectProjectType reads package.json and determines the project type.
func detectProjectType(projectDir string) (ProjectType, error) {
	pkgPath := filepath.Join(projectDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return ProjectTypeUnknown, fmt.Errorf("no package.json found in %s: is this a React Native or Expo project?", projectDir)
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ProjectTypeUnknown, fmt.Errorf("parsing package.json: %w", err)
	}

	// Check for Expo first (Expo projects also have react-native as a dependency)
	if _, ok := pkg.Dependencies["expo"]; ok {
		return ProjectTypeExpo, nil
	}
	if _, ok := pkg.DevDependencies["expo"]; ok {
		return ProjectTypeExpo, nil
	}

	if _, ok := pkg.Dependencies["react-native"]; ok {
		return ProjectTypeReactNative, nil
	}
	if _, ok := pkg.DevDependencies["react-native"]; ok {
		return ProjectTypeReactNative, nil
	}

	return ProjectTypeUnknown, fmt.Errorf("could not detect project type: package.json does not list react-native or expo as a dependency")
}

// detectEntryFile searches for the JS entry file.
// Priority: index.<platform>.js, then index.js, then package.json "main" field.
func detectEntryFile(projectDir string, platform Platform) (string, error) {
	platformSpecific := fmt.Sprintf("index.%s.js", platform)
	candidates := []string{platformSpecific, "index.js"}

	for _, candidate := range candidates {
		path := filepath.Join(projectDir, candidate)
		if _, err := os.Stat(path); err == nil {
			return candidate, nil
		}
	}

	// Fall back to package.json "main" field
	pkgPath := filepath.Join(projectDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err == nil {
		var pkg packageJSON
		if err := json.Unmarshal(data, &pkg); err == nil && pkg.Main != "" {
			mainPath := filepath.Join(projectDir, pkg.Main)
			if _, err := os.Stat(mainPath); err == nil {
				return pkg.Main, nil
			}
		}
	}

	return "", fmt.Errorf("entry file not found: tried %s and index.js in %s", platformSpecific, projectDir)
}

// detectHermes checks the project for Hermes configuration.
func detectHermes(projectDir string, platform Platform) (bool, error) {
	switch platform {
	case PlatformAndroid:
		return detectHermesAndroid(projectDir)
	case PlatformIOS:
		return detectHermesIOS(projectDir)
	default:
		return false, nil
	}
}

// detectHermesAndroid checks android/app/build.gradle for Hermes configuration.
func detectHermesAndroid(projectDir string) (bool, error) {
	gradlePaths := []string{
		filepath.Join(projectDir, "android", "app", "build.gradle"),
		filepath.Join(projectDir, "android", "app", "build.gradle.kts"),
	}

	for _, gradlePath := range gradlePaths {
		data, err := os.ReadFile(gradlePath)
		if err != nil {
			continue
		}

		content := string(data)
		// Check for various Hermes configuration patterns across RN versions
		hermesPatterns := []string{
			"hermesEnabled = true",
			"hermesEnabled.set(true)",
			"enableHermes: true",
			"enableHermes = true",
		}
		for _, pattern := range hermesPatterns {
			if strings.Contains(content, pattern) {
				return true, nil
			}
		}

		// Check for explicit disable
		disablePatterns := []string{
			"hermesEnabled = false",
			"hermesEnabled.set(false)",
			"enableHermes: false",
			"enableHermes = false",
		}
		for _, pattern := range disablePatterns {
			if strings.Contains(content, pattern) {
				return false, nil
			}
		}
	}

	// Default: Hermes is enabled by default in RN 0.70+
	return false, nil
}

// detectHermesIOS checks ios/Podfile for Hermes configuration.
func detectHermesIOS(projectDir string) (bool, error) {
	podfilePath := filepath.Join(projectDir, "ios", "Podfile")
	data, err := os.ReadFile(podfilePath)
	if err != nil {
		return false, nil
	}

	content := string(data)
	enablePatterns := []string{
		":hermes_enabled => true",
		"hermes_enabled: true",
	}
	for _, pattern := range enablePatterns {
		if strings.Contains(content, pattern) {
			return true, nil
		}
	}

	disablePatterns := []string{
		":hermes_enabled => false",
		"hermes_enabled: false",
	}
	for _, pattern := range disablePatterns {
		if strings.Contains(content, pattern) {
			return false, nil
		}
	}

	return false, nil
}

// findHermesc locates the hermesc binary in node_modules.
func findHermesc(projectDir string) (string, error) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Map Go OS/arch names to hermesc directory conventions
	var osTriplet string
	switch {
	case osName == "darwin" && archName == "arm64":
		osTriplet = "osx-bin"
	case osName == "darwin" && archName == "amd64":
		osTriplet = "osx-bin"
	case osName == "linux" && archName == "amd64":
		osTriplet = "linux64-bin"
	default:
		osTriplet = osName + "-bin"
	}

	// Check known hermesc locations in order of preference
	candidates := []string{
		filepath.Join(projectDir, "node_modules", "hermes-engine", osTriplet, "hermesc"),
		filepath.Join(projectDir, "node_modules", "react-native", "sdks", "hermesc", osTriplet, "hermesc"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("hermesc binary not found in node_modules")
}

// detectMetroConfig searches for metro.config.js or metro.config.ts.
func detectMetroConfig(projectDir string) string {
	candidates := []string{
		filepath.Join(projectDir, "metro.config.js"),
		filepath.Join(projectDir, "metro.config.ts"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
