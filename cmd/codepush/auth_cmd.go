package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/auth"
)

// Auth flags.
var authLoginToken string

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  `Manage the Bitrise API token used for CodePush operations.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store a Bitrise API token locally",
	Long: `Store a Bitrise API token in the local config file.

The token is saved to the config directory and used automatically
by commands that require authentication (push, rollback).

Generate a personal access token at: ` + auth.TokenGenerationURL + `

Token resolution order: --token flag > BITRISE_API_TOKEN env var > stored config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		token := authLoginToken
		if token == "" {
			out.Println("")
			out.Info("Generate a token at: %s", auth.TokenGenerationURL)
			out.Println("")
			out.Println("  Paste your personal access token: ")
			input, err := auth.ReadTokenSecure()
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.RemoveToken(); err != nil {
			return fmt.Errorf("removing token: %w", err)
		}

		out.Success("Token revoked successfully")
		return nil
	},
}

func registerAuthFlags() {
	authLoginCmd.Flags().StringVar(&authLoginToken, "token", "", "Bitrise API token")
}
