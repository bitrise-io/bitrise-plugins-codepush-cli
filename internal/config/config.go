// Package config handles project-level configuration stored in .codepush.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileName is the project-level config file name.
const FileName = ".codepush.json"

// ProjectConfig represents the project-level configuration file.
type ProjectConfig struct {
	AppID string `json:"app_id"`
}

// configDirFunc allows tests to override the directory where the config file is read from.
var configDirFunc = defaultConfigDir

func defaultConfigDir() (string, error) {
	return os.Getwd()
}

// Load reads the project config from the current directory.
// Returns (nil, nil) if the file does not exist.
func Load() (*ProjectConfig, error) {
	dir, err := configDirFunc()
	if err != nil {
		return nil, fmt.Errorf("determining working directory: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", FileName, err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", FileName, err)
	}

	return &cfg, nil
}

// Save writes the project config to the given directory.
func Save(dir string, cfg *ProjectConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(filepath.Join(dir, FileName), data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", FileName, err)
	}

	return nil
}

// FilePath returns the path of the config file in the current directory.
func FilePath() (string, error) {
	dir, err := configDirFunc()
	if err != nil {
		return "", fmt.Errorf("determining working directory: %w", err)
	}
	return filepath.Join(dir, FileName), nil
}
