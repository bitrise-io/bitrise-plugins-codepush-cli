package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	configDirName  = "codepush"
	configFileName = "config.json"
	validateURL    = "https://api.bitrise.io/v0.1/me"
)

// Config represents the persisted CLI configuration.
type Config struct {
	Token string `json:"token"`
}

// configDirFunc allows tests to override the config directory.
var configDirFunc = defaultConfigDir

func defaultConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determining config directory: %w", err)
	}
	return filepath.Join(base, configDirName), nil
}

func configFilePath() (string, error) {
	dir, err := configDirFunc()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// SaveToken persists the API token to the config file.
func SaveToken(token string) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	config := Config{Token: token}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// LoadToken reads the stored API token from the config file.
// Returns an empty string and no error if the config file does not exist.
func LoadToken() (string, error) {
	path, err := configFilePath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("decoding config file: %w", err)
	}

	return config.Token, nil
}

// RemoveToken deletes the config file, effectively revoking the stored token.
// Returns no error if the config file does not exist.
func RemoveToken() error {
	path, err := configFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("removing config file: %w", err)
	}

	return nil
}

// ConfigFilePath returns the path where the config file is stored.
func ConfigFilePath() (string, error) {
	return configFilePath()
}

// ValidateToken checks the token against the Bitrise API.
// Returns nil if the token is valid, or an error describing the failure.
func ValidateToken(token string) error {
	return validateTokenWithURL(token, validateURL, &http.Client{})
}

func validateTokenWithURL(token, url string, client *http.Client) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating validation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("validating token: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("invalid token: the API returned 401 Unauthorized")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("token validation failed: the API returned HTTP %d", resp.StatusCode)
	}

	return nil
}
