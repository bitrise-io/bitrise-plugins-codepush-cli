package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/codepush"
	"github.com/bitrise-io/bitrise-plugins-codepush-cli/internal/output"
)

// Deployment command flags
var (
	deploymentRenameName string
	deploymentRemoveYes  bool
	deploymentHistoryMax int
	deploymentClearYes   bool
)

var deploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Manage deployments",
	Long:  `Create, list, inspect, rename, and delete CodePush deployments.`,
}

var deploymentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployments",
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)
		deployments, err := client.ListDeployments(cmd.Context(), appID)
		if err != nil {
			return fmt.Errorf("listing deployments: %w", err)
		}

		if globalJSON {
			return outputJSON(deployments)
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

var deploymentAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)
		dep, err := client.CreateDeployment(cmd.Context(), appID, codepush.CreateDeploymentRequest{Name: args[0]})
		if err != nil {
			return fmt.Errorf("creating deployment: %w", err)
		}

		if globalJSON {
			return outputJSON(dep)
		}

		out.Success("Deployment %q created (ID: %s)", dep.Name, dep.ID)
		return nil
	},
}

var deploymentInfoCmd = &cobra.Command{
	Use:   "info <deployment>",
	Short: "Show deployment details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(cmd.Context(), client, appID, args[0], out)
		if err != nil {
			return err
		}

		dep, err := client.GetDeployment(cmd.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("getting deployment: %w", err)
		}

		packages, err := client.ListPackages(cmd.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		if globalJSON {
			info := struct {
				codepush.Deployment
				LatestPackage *codepush.Package `json:"latest_package,omitempty"`
			}{Deployment: *dep}
			if len(packages) > 0 {
				info.LatestPackage = &packages[len(packages)-1]
			}
			return outputJSON(info)
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
				{Key: "Rollout", Value: fmt.Sprintf("%d%%", latest.Rollout)},
			})
		} else {
			out.Info("No releases.")
		}

		return nil
	},
}

var deploymentRenameCmd = &cobra.Command{
	Use:   "rename <deployment>",
	Short: "Rename a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}
		if deploymentRenameName == "" {
			return fmt.Errorf("new name is required: set --name")
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(cmd.Context(), client, appID, args[0], out)
		if err != nil {
			return err
		}

		dep, err := client.RenameDeployment(cmd.Context(), appID, deploymentID, codepush.RenameDeploymentRequest{Name: deploymentRenameName})
		if err != nil {
			return fmt.Errorf("renaming deployment: %w", err)
		}

		if globalJSON {
			return outputJSON(dep)
		}

		out.Success("Deployment renamed to %q", dep.Name)
		return nil
	},
}

var deploymentRemoveCmd = &cobra.Command{
	Use:   "remove <deployment>",
	Short: "Delete a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		if err := out.ConfirmDestructive(
			fmt.Sprintf("This will permanently delete deployment %q and all its releases", args[0]),
			deploymentRemoveYes,
		); err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(cmd.Context(), client, appID, args[0], out)
		if err != nil {
			return err
		}

		if err := client.DeleteDeployment(cmd.Context(), appID, deploymentID); err != nil {
			return fmt.Errorf("deleting deployment: %w", err)
		}

		if globalJSON {
			return outputJSON(struct {
				Deleted string `json:"deleted"`
			}{Deleted: deploymentID})
		}

		out.Success("Deployment %q deleted", args[0])
		return nil
	},
}

var deploymentHistoryCmd = &cobra.Command{
	Use:   "history <deployment>",
	Short: "Show release history for a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(cmd.Context(), client, appID, args[0], out)
		if err != nil {
			return err
		}

		packages, err := client.ListPackages(cmd.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		// Apply limit: show the most recent entries
		if deploymentHistoryMax > 0 && len(packages) > deploymentHistoryMax {
			packages = packages[len(packages)-deploymentHistoryMax:]
		}

		if globalJSON {
			return outputJSON(packages)
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
				fmt.Sprintf("%d%%", p.Rollout), fmt.Sprintf("%v", p.Disabled),
				truncate(p.Description, 30), p.CreatedAt,
			}
		}
		out.Table(headers, rows)

		return nil
	},
}

var deploymentClearCmd = &cobra.Command{
	Use:   "clear <deployment>",
	Short: "Delete all packages from a deployment",
	Long: `Delete all packages (releases) from a deployment.

This is a destructive operation that removes all release history.
Requires --yes to confirm.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appID, token, err := requireCredentials()
		if err != nil {
			return err
		}

		if err := out.ConfirmDestructive(
			fmt.Sprintf("This will permanently delete all releases from %q", args[0]),
			deploymentClearYes,
		); err != nil {
			return err
		}

		client := codepush.NewHTTPClient(defaultAPIURL, token)

		deploymentID, err := codepush.ResolveDeployment(cmd.Context(), client, appID, args[0], out)
		if err != nil {
			return err
		}

		packages, err := client.ListPackages(cmd.Context(), appID, deploymentID)
		if err != nil {
			return fmt.Errorf("listing packages: %w", err)
		}

		if len(packages) == 0 {
			out.Info("No packages to delete.")
			return nil
		}

		deleted := 0
		for _, pkg := range packages {
			if err := client.DeletePackage(cmd.Context(), appID, deploymentID, pkg.ID); err != nil {
				return fmt.Errorf("deleting package %s: %w", pkg.Label, err)
			}
			deleted++
		}

		if globalJSON {
			return outputJSON(struct {
				Deployment string `json:"deployment"`
				Deleted    int    `json:"deleted"`
			}{Deployment: deploymentID, Deleted: deleted})
		}

		out.Success("Deleted %d package(s) from %q", deleted, args[0])
		return nil
	},
}

func registerDeploymentFlags() {
	deploymentRenameCmd.Flags().StringVar(&deploymentRenameName, "name", "", "new deployment name (required)")
	deploymentRemoveCmd.Flags().BoolVar(&deploymentRemoveYes, "yes", false, "skip confirmation prompt")
	deploymentHistoryCmd.Flags().IntVar(&deploymentHistoryMax, "limit", 10, "maximum number of releases to show")
	deploymentClearCmd.Flags().BoolVar(&deploymentClearYes, "yes", false, "skip confirmation prompt")
}
