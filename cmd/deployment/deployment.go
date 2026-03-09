package deployment

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/cmdutil"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var (
	renameName           string
	removeYes            bool
	historyMax           int
	listDisplayKeys      bool
	historyDisplayAuthor bool
	clearYes             bool
)

var deploymentCmd = &cobra.Command{
	Use:     "deployment",
	Short:   "Manage deployments",
	Long:    `Create, list, inspect, rename, and delete CodePush deployments.`,
	GroupID: cmd.GroupDeployment,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployments",
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmdutil.APIURL(cmdutil.ResolveServerURL(cmd.ServerURL, out)), token)
		deployments, err := client.ListDeployments(c.Context(), appID)
		if err != nil {
			return fmt.Errorf("listing deployments: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(deployments)
		}

		if len(deployments) == 0 {
			out.Info("No deployments found.")
			return nil
		}

		headers := []string{"NAME", "ID"}
		if listDisplayKeys {
			headers = append(headers, "KEY")
		}
		rows := make([][]string, len(deployments))
		for i, d := range deployments {
			row := []string{d.Name, d.ID}
			if listDisplayKeys {
				row = append(row, d.Key)
			}
			rows[i] = row
		}
		out.Table(headers, rows)

		return nil
	},
}

var addCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Create a new deployment",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		var name string
		if len(args) > 0 {
			name = args[0]
		}
		name, err = cmdutil.ResolveInputInteractive(name, "Enter deployment name", "e.g. Staging, Production", out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmdutil.APIURL(cmdutil.ResolveServerURL(cmd.ServerURL, out)), token)
		dep, err := client.CreateDeployment(c.Context(), appID, codepush.CreateDeploymentRequest{Name: name})
		if err != nil {
			return fmt.Errorf("creating deployment: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(dep)
		}

		out.Success("Deployment %q created (ID: %s)", dep.Name, dep.ID)
		return nil
	},
}

var infoCmd = &cobra.Command{
	Use:   "info [deployment]",
	Short: "Show deployment details",
	Args:  cobra.MaximumNArgs(1),
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

		dep, err := client.GetDeployment(c.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("getting deployment: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(dep)
		}

		out.Step("Deployment: %s", dep.Name)
		pairs := []output.KeyValue{
			{Key: "ID", Value: dep.ID},
		}
		if dep.Key != "" {
			pairs = append(pairs, output.KeyValue{Key: "Key", Value: dep.Key})
		}
		if dep.CreatedAt != "" {
			pairs = append(pairs, output.KeyValue{Key: "Created", Value: dep.CreatedAt})
		}
		out.Result(pairs)

		if dep.LatestUpdate != nil {
			out.Step("Latest release")
			out.Result([]output.KeyValue{
				{Key: "Label", Value: dep.LatestUpdate.Label},
				{Key: "App version", Value: dep.LatestUpdate.AppVersion},
				{Key: "Mandatory", Value: strconv.FormatBool(dep.LatestUpdate.Mandatory)},
				{Key: "Rollout", Value: fmt.Sprintf("%.0f%%", dep.LatestUpdate.Rollout)},
			})
		} else {
			out.Info("No releases.")
		}

		return nil
	},
}

var renameCmd = &cobra.Command{
	Use:   "rename [deployment]",
	Short: "Rename a deployment",
	Args:  cobra.MaximumNArgs(1),
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

		newName, err := cmdutil.ResolveInputInteractive(renameName, "Enter new deployment name", "e.g. Staging, Production", out)
		if err != nil {
			return err
		}

		dep, err := client.RenameDeployment(c.Context(), appID, deploymentID, codepush.RenameDeploymentRequest{Name: newName})
		if err != nil {
			return fmt.Errorf("renaming deployment: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(dep)
		}

		out.Success("Deployment renamed to %q", dep.Name)
		return nil
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [deployment]",
	Short: "Delete a deployment",
	Args:  cobra.MaximumNArgs(1),
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

		displayName := argValue
		if displayName == "" {
			displayName = deploymentID
		}

		if err := out.ConfirmDestructive(
			fmt.Sprintf("This will permanently delete deployment %q and all its releases", displayName),
			removeYes,
		); err != nil {
			return err
		}

		if err := client.DeleteDeployment(c.Context(), appID, deploymentID); err != nil {
			return fmt.Errorf("deleting deployment: %w", err)
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(struct {
				Deleted string `json:"deleted"`
			}{Deleted: deploymentID})
		}

		out.Success("Deployment %q deleted", displayName)
		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history [deployment]",
	Short: "Show release history for a deployment",
	Args:  cobra.MaximumNArgs(1),
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

		updates, err := client.ListUpdates(c.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing updates: %w", err)
		}

		if historyMax > 0 && len(updates) > historyMax {
			updates = updates[len(updates)-historyMax:]
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(updates)
		}

		if len(updates) == 0 {
			out.Info("No releases found.")
			return nil
		}

		headers := []string{"LABEL", "APP VERSION", "MANDATORY", "ROLLOUT", "DISABLED", "DESCRIPTION", "CREATED"}
		if historyDisplayAuthor {
			headers = append(headers, "AUTHOR")
		}
		rows := make([][]string, len(updates))
		for i, u := range updates {
			row := []string{
				u.Label, u.AppVersion, strconv.FormatBool(u.Mandatory),
				fmt.Sprintf("%.0f%%", u.Rollout), strconv.FormatBool(u.Disabled),
				cmdutil.Truncate(u.Description, 30), u.CreatedAt,
			}
			if historyDisplayAuthor {
				author := ""
				if u.CreatedBy != nil {
					author = u.CreatedBy.Username
					if author == "" {
						author = u.CreatedBy.Email
					}
				}
				row = append(row, author)
			}
			rows[i] = row
		}
		out.Table(headers, rows)

		return nil
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear [deployment]",
	Short: "Delete all updates from a deployment",
	Long: `Delete all updates (releases) from a deployment.

This is a destructive operation that removes all release history.
Requires --yes to confirm.`,
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

		displayName := argValue
		if displayName == "" {
			displayName = deploymentID
		}

		if err := out.ConfirmDestructive(
			fmt.Sprintf("This will permanently delete all releases from %q", displayName),
			clearYes,
		); err != nil {
			return err
		}

		updates, err := client.ListUpdates(c.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing updates: %w", err)
		}

		if len(updates) == 0 {
			out.Info("No updates to delete.")
			return nil
		}

		deleted := 0
		for _, u := range updates {
			if err := client.DeleteUpdate(c.Context(), appID, deploymentID, u.ID); err != nil {
				return fmt.Errorf("deleting update %s: %w", u.Label, err)
			}
			deleted++
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(struct {
				Deployment string `json:"deployment"`
				Deleted    int    `json:"deleted"`
			}{Deployment: deploymentID, Deleted: deleted})
		}

		out.Success("Deleted %d update(s) from %q", deleted, displayName)
		return nil
	},
}

func init() {
	cmd.RootCmd.AddGroup(&cobra.Group{ID: cmd.GroupDeployment, Title: "Deployment Management:"})

	listCmd.Flags().BoolVarP(&listDisplayKeys, "display-keys", "k", false, "include the deployment key column in the list table")
	renameCmd.Flags().StringVar(&renameName, "name", "", "new deployment name (required)")
	removeCmd.Flags().BoolVar(&removeYes, "yes", false, "skip confirmation prompt")
	historyCmd.Flags().IntVar(&historyMax, "limit", 10, "maximum number of releases to show")
	historyCmd.Flags().BoolVarP(&historyDisplayAuthor, "display-author", "a", false, "include the author column in the history table")
	clearCmd.Flags().BoolVar(&clearYes, "yes", false, "skip confirmation prompt")

	deploymentCmd.AddCommand(listCmd, addCmd, infoCmd, renameCmd, removeCmd, historyCmd, clearCmd)
	cmd.RootCmd.AddCommand(deploymentCmd)
}
