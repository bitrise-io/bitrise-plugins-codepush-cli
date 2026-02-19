package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
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

// TokenGenerationURL is the URL where users can create a personal access token.
const TokenGenerationURL = "https://app.bitrise.io/me/account/security"

// UserInfo contains the authenticated user's identity.
type UserInfo struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ValidateToken checks the token against the Bitrise API.
// Returns the authenticated user's info, or an error if the token is invalid.
func ValidateToken(token string) (*UserInfo, error) {
	return validateTokenWithURL(token, validateURL, &http.Client{})
}

func validateTokenWithURL(token, url string, client *http.Client) (*UserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating validation request: %w", err)
	}

	req.Header.Set("Authorization", token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("validating token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("invalid token: the API returned 401 Unauthorized")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("token validation failed: the API returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		Data UserInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil //nolint: user info is best-effort
	}

	return &result.Data, nil
}

// ReadTokenSecure reads a token from stdin with masked input.
// Falls back to plain text reading if the terminal does not support secure input.
func ReadTokenSecure() (string, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		tokenBytes, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return "", fmt.Errorf("reading secure input: %w", err)
		}
		return strings.TrimSpace(string(tokenBytes)), nil
	}

	// Non-terminal stdin (piped input)
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return "", fmt.Errorf("reading token from stdin: %w", err)
	}
	return strings.TrimSpace(input), nil
}
