package deployment

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/cmd"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/cmdutil"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

var (
	renameName string
	removeYes  bool
	historyMax int
	clearYes   bool
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

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)
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

		rows := make([][]string, len(deployments))
		for i, d := range deployments {
			rows[i] = []string{d.Name, d.ID}
		}
		out.Table([]string{"NAME", "ID"}, rows)

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

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)
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

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

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

		packages, err := client.ListPackages(c.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		if cmd.JSONOutput {
			info := struct {
				codepush.Deployment
				LatestPackage *codepush.Package `json:"latest_package,omitempty"`
			}{Deployment: *dep}
			if len(packages) > 0 {
				info.LatestPackage = &packages[len(packages)-1]
			}
			return cmdutil.OutputJSON(info)
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

		if len(packages) > 0 {
			latest := packages[len(packages)-1]
			out.Step("Latest release")
			out.Result([]output.KeyValue{
				{Key: "Label", Value: latest.Label},
				{Key: "App version", Value: latest.AppVersion},
				{Key: "Mandatory", Value: fmt.Sprintf("%v", latest.Mandatory)},
				{Key: "Rollout", Value: fmt.Sprintf("%.0f%%", latest.Rollout)},
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

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

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

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

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

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

		var argValue string
		if len(args) > 0 {
			argValue = args[0]
		}

		deploymentID, err := cmdutil.ResolveDeploymentInteractive(c.Context(), client, appID, argValue, "CODEPUSH_DEPLOYMENT", out)
		if err != nil {
			return err
		}

		packages, err := client.ListPackages(c.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		if historyMax > 0 && len(packages) > historyMax {
			packages = packages[len(packages)-historyMax:]
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(packages)
		}

		if len(packages) == 0 {
			out.Info("No releases found.")
			return nil
		}

		headers := []string{"LABEL", "APP VERSION", "MANDATORY", "ROLLOUT", "DISABLED", "DESCRIPTION", "CREATED"}
		rows := make([][]string, len(packages))
		for i, p := range packages {
			rows[i] = []string{
				p.Label, p.AppVersion, fmt.Sprintf("%v", p.Mandatory),
				fmt.Sprintf("%.0f%%", p.Rollout), fmt.Sprintf("%v", p.Disabled),
				cmdutil.Truncate(p.Description, 30), p.CreatedAt,
			}
		}
		out.Table(headers, rows)

		return nil
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear [deployment]",
	Short: "Delete all packages from a deployment",
	Long: `Delete all packages (releases) from a deployment.

This is a destructive operation that removes all release history.
Requires --yes to confirm.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(c *cobra.Command, args []string) error {
		out := cmd.Out

		appID, token, err := cmdutil.RequireCredentials(cmd.AppID, out)
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(cmd.DefaultAPIURL, token)

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

		packages, err := client.ListPackages(c.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		if len(packages) == 0 {
			out.Info("No packages to delete.")
			return nil
		}

		deleted := 0
		for _, pkg := range packages {
			if err := client.DeletePackage(c.Context(), appID, deploymentID, pkg.ID); err != nil {
				return fmt.Errorf("deleting package %s: %w", pkg.Label, err)
			}
			deleted++
		}

		if cmd.JSONOutput {
			return cmdutil.OutputJSON(struct {
				Deployment string `json:"deployment"`
				Deleted    int    `json:"deleted"`
			}{Deployment: deploymentID, Deleted: deleted})
		}

		out.Success("Deleted %d package(s) from %q", deleted, displayName)
		return nil
	},
}

func init() {
	cmd.RootCmd.AddGroup(&cobra.Group{ID: cmd.GroupDeployment, Title: "Deployment Management:"})

	renameCmd.Flags().StringVar(&renameName, "name", "", "new deployment name (required)")
	removeCmd.Flags().BoolVar(&removeYes, "yes", false, "skip confirmation prompt")
	historyCmd.Flags().IntVar(&historyMax, "limit", 10, "maximum number of releases to show")
	clearCmd.Flags().BoolVar(&clearYes, "yes", false, "skip confirmation prompt")

	deploymentCmd.AddCommand(listCmd, addCmd, infoCmd, renameCmd, removeCmd, historyCmd, clearCmd)
	cmd.RootCmd.AddCommand(deploymentCmd)
}
