package setup

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/auth"
)

var authLoginToken string

var authCmd = &cobra.Command{
	Use:     "auth",
	Short:   "Manage authentication",
	Long:    `Manage the Bitrise API token used for CodePush operations.`,
	GroupID: cmd.GroupSetup,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store a Bitrise API token locally",
	Long: `Store a Bitrise API token in the local config file.

The token is saved to the config directory and used automatically
by commands that require authentication (push, rollback).

Generate a personal access token at: ` + auth.TokenGenerationURL + `

Token resolution order: --token flag > BITRISE_API_TOKEN env var > stored config.`,
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out
		token := authLoginToken
		if token == "" {
			if !out.IsInteractive() {
				return fmt.Errorf("token is required: set --token or BITRISE_API_TOKEN")
			}
			out.Println("")
			out.Info("Generate a token at: %s", auth.TokenGenerationURL)

			input, err := out.SecureInput("Paste your personal access token", "")
			if err != nil {
				return fmt.Errorf("reading token: %w", err)
			}
			token = input
		}

		if token == "" {
			return fmt.Errorf("token is required: provide --token flag or enter interactively")
		}

		var userInfo *auth.UserInfo
		err := out.Spinner("Validating token", func() error {
			var valErr error
			userInfo, valErr = auth.ValidateToken(token)
			return valErr
		})
		if err != nil {
			return fmt.Errorf("token validation failed: %w\n\n  Generate a new token at: %s", err, auth.TokenGenerationURL)
		}

		if err := auth.SaveToken(token); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}

		if userInfo != nil && userInfo.Username != "" {
			if userInfo.Email != "" {
				out.Success("Logged in as %s (%s)", userInfo.Username, userInfo.Email)
			} else {
				out.Success("Logged in as %s", userInfo.Username)
			}
		}

		configPath, err := auth.ConfigFilePath()
		if err != nil {
			out.Warning("could not determine config path: %v", err)
		} else {
			out.Info("Token saved to: %s", configPath)
		}
		return nil
	},
}

var authRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Remove the stored API token",
	Long: `Remove the locally stored Bitrise API token.

After revoking, commands that require authentication will need
a --token flag or BITRISE_API_TOKEN environment variable.`,
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		if err := auth.RemoveToken(); err != nil {
			return fmt.Errorf("removing token: %w", err)
		}

		out.Success("Token revoked successfully")
		return nil
	},
}

func init() {
	authLoginCmd.Flags().StringVar(&authLoginToken, "token", "", "Bitrise API token")
	authCmd.AddCommand(authLoginCmd, authRevokeCmd)
	cmd.RootCmd.AddCommand(authCmd)
}
