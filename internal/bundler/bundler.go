package bundler

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// DefaultOutputDir is the default output directory for bundle generation.
const DefaultOutputDir = "./codepush-bundle"

// ValidatePlatform checks that the given platform string is valid.
func ValidatePlatform(p Platform) error {
	if p != PlatformIOS && p != PlatformAndroid {
		return fmt.Errorf("--platform must be 'ios' or 'android', got %q", p)
	}
	return nil
}

// ValidateHermesMode checks that the given hermes mode string is valid.
func ValidateHermesMode(h HermesMode) error {
	if h != HermesModeAuto && h != HermesModeOn && h != HermesModeOff {
		return fmt.Errorf("--hermes must be 'auto', 'on', or 'off', got %q", h)
	}
	return nil
}

// BundleOptions holds user-specified options for bundle generation.
type BundleOptions struct {
	Platform         Platform
	EntryFile        string
	OutputDir        string
	BundleName       string
	Dev              bool
	Sourcemap        bool
	HermesMode       HermesMode
	ExtraBundlerOpts []string
	ProjectDir       string
	MetroConfig      string
	SkipInstall      bool
}

// BundleResult contains the output of a successful bundle operation.
type BundleResult struct {
	BundlePath    string
	AssetsDir     string
	SourcemapPath string
	OutputDir     string
	HermesApplied bool
	ProjectType   ProjectType
	Platform      Platform
}

// Bundler is the interface for building a JS bundle.
type Bundler interface {
	Bundle(config *ProjectConfig, opts *BundleOptions) (*BundleResult, error)
}

// CommandExecutor abstracts subprocess execution for testing.
type CommandExecutor interface {
	Run(dir string, stdout io.Writer, stderr io.Writer, name string, args ...string) error
}

// DefaultExecutor implements CommandExecutor using os/exec.
type DefaultExecutor struct{}

// Run executes a command with the given args in the given directory.
func (e *DefaultExecutor) Run(dir string, stdout io.Writer, stderr io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// NewBundler creates the appropriate Bundler implementation based on project type.
func NewBundler(projectType ProjectType, executor CommandExecutor, out *output.Writer) (Bundler, error) {
	switch projectType {
	case ProjectTypeReactNative:
		return &ReactNativeBundler{executor: executor, out: out}, nil
	case ProjectTypeExpo:
		return &ExpoBundler{executor: executor, out: out}, nil
	default:
		return nil, fmt.Errorf("unsupported project type: %s", projectType)
	}
}

// DefaultBundleName returns the platform-specific default bundle filename.
func DefaultBundleName(platform Platform) string {
	switch platform {
	case PlatformIOS:
		return "main.jsbundle"
	case PlatformAndroid:
		return "index.android.bundle"
	default:
		return "index.bundle"
	}
}

// ensureDir creates a directory if it does not exist.
func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}
	return nil
}
