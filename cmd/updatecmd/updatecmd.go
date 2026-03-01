package updatecmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/cmdutil"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var (
	updateLabel     string
	updateRemoveYes bool
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Inspect updates (releases)",
	Long:    `View details and processing status of CodePush updates.`,
	GroupID: cmd.GroupUpdate,
}

var infoCmd = &cobra.Command{
	Use:   "info [deployment]",
	Short: "Show update details",
	Long: `Show details for a specific update in a deployment.

By default shows the latest update. Use --label to specify a version.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmdutil.APIURL(cmdutil.ResolveServerURL(cmd.ServerURL, out)), token)

		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, argValue, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		updateID, _, err := codepush.ResolveUpdateForPatch(c.Context(), client, appID, deploymentID, updateLabel, out)
		if err != nil {
			return err
		}

		pkg, err := client.GetUpdate(c.Context(), appID, deploymentID, updateID)
		if err != nil {
			return fmt.Errorf("getting update: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(pkg)
		}

		out.Step("Update: %s", pkg.Label)
		pairs := []output.KeyValue{
			{Key: "ID", Value: pkg.ID},
			{Key: "App version", Value: pkg.AppVersion},
			{Key: "Mandatory", Value: strconv.FormatBool(pkg.Mandatory)},
			{Key: "Disabled", Value: strconv.FormatBool(pkg.Disabled)},
			{Key: "Rollout", Value: fmt.Sprintf("%.0f%%", pkg.Rollout)},
		}
		if pkg.Description != "" {
			pairs = append(pairs, output.KeyValue{Key: "Description", Value: pkg.Description})
		}
		pairs = append(pairs, output.KeyValue{Key: "Size", Value: cmdutil.FormatBytes(pkg.FileSizeBytes)})
		if pkg.Hash != "" {
			pairs = append(pairs, output.KeyValue{Key: "Hash", Value: pkg.Hash})
		}
		if pkg.CreatedAt != "" {
			pairs = append(pairs, output.KeyValue{Key: "Created", Value: pkg.CreatedAt})
		}
		if pkg.CreatedBy != nil && pkg.CreatedBy.Email != "" {
			pairs = append(pairs, output.KeyValue{Key: "Created by", Value: pkg.CreatedBy.Email})
		}
		out.Result(pairs)

		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [deployment]",
	Short: "Show update processing status",
	Long: `Show the processing status of a specific update.

By default shows the latest update. Use --label to specify a version.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmdutil.APIURL(cmdutil.ResolveServerURL(cmd.ServerURL, out)), token)

		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, argValue, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		updateID, updLabel, err := codepush.ResolveUpdateForPatch(c.Context(), client, appID, deploymentID, updateLabel, out)
		if err != nil {
			return err
		}

		status, err := client.GetUpdateStatus(c.Context(), appID, deploymentID, updateID)
		if err != nil {
			return fmt.Errorf("getting update status: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(status)
		}

		pairs := []output.KeyValue{
			{Key: "Update", Value: updLabel},
			{Key: "Status", Value: status.Status},
		}
		if status.StatusReason != "" {
			pairs = append(pairs, output.KeyValue{Key: "Reason", Value: status.StatusReason})
		}
		out.Result(pairs)

		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [deployment]",
	Short: "Delete an update from a deployment",
	Long: `Delete a specific update from a deployment.

Requires --label to identify the update and --yes to confirm deletion.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}
		if updateLabel == "" {
			return errors.New("label is required: set --label to identify the update to delete")
		}

		if err := out.ConfirmDestructive(
			fmt.Sprintf("This will permanently delete update %q", updateLabel),
			updateRemoveYes,
		); err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmdutil.APIURL(cmdutil.ResolveServerURL(cmd.ServerURL, out)), token)

		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, argValue, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		updateID, _, err := codepush.ResolveUpdateForPatch(c.Context(), client, appID, deploymentID, updateLabel, out)
		if err != nil {
			return err
		}

		if err := client.DeleteUpdate(c.Context(), appID, deploymentID, updateID); err != nil {
			return fmt.Errorf("deleting update: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(struct {
				Deleted string `json:"deleted"`
				Label   string `json:"label"`
			}{Deleted: updateID, Label: updateLabel})
		}

		out.Success("Update %q deleted", updateLabel)
		return nil
	},
}

func init() {
	cmd.RootCmd.AddGroup(&cobra.Group{ID: cmd.GroupUpdate, Title: "Update Management:"})

	infoCmd.Flags().StringVar(&updateLabel, "label", "", "specific release label (defaults to latest)")
	statusCmd.Flags().StringVar(&updateLabel, "label", "", "specific release label (defaults to latest)")
	removeCmd.Flags().StringVar(&updateLabel, "label", "", "release label to delete (required)")
	removeCmd.Flags().BoolVar(&updateRemoveYes, "yes", false, "skip confirmation prompt")

	updateCmd.AddCommand(infoCmd, statusCmd, removeCmd)
	cmd.RootCmd.AddCommand(updateCmd)
}
