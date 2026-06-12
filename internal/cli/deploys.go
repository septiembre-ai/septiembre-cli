package cli

import (
	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewDeploysCmd returns the `deploys` command group.
func NewDeploysCmd() *cobra.Command {
	deploys := &cobra.Command{
		Use:   "deploys",
		Short: "Manage application deployments",
		Long: `Commands for triggering and inspecting Septiembre deployments.

AGENT USAGE
  Trigger and poll:
    septiembre deploys trigger <app-id> --org <slug> --tag v1.2.3
    septiembre deploys status <app-id> <deploy-id> --org <slug>

  List recent deployments:
    septiembre deploys list <app-id> --org <slug> | jq '.[0].status'`,
	}
	deploys.AddCommand(newDeploysTriggerCmd())
	deploys.AddCommand(newDeploysListCmd())
	deploys.AddCommand(newDeploysStatusCmd())
	return deploys
}

// newDeploysTriggerCmd returns `deploys trigger <app-id> [--tag <tag>] [--env-id <id>]`.
func newDeploysTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger <app-id>",
		Short: "Trigger a new deployment for an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tag, _ := cmd.Flags().GetString("tag")
			envID, _ := cmd.Flags().GetString("env-id")

			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}

			orgID, err := requireOrgID(cmd.Context(), cmd, c, r)
			if err != nil {
				return err
			}

			deploy, err := c.TriggerDeployment(cmd.Context(), orgID, args[0], tag, envID)
			if err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(deploy); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
	cmd.Flags().String("tag", "", "Release tag to deploy (e.g. v1.2.3). Required for API and SSE app types.")
	cmd.Flags().String("env-id", "", "Environment UUID (defaults to the production environment when empty)")
	return cmd
}

// newDeploysListCmd returns `deploys list <app-id>`.
func newDeploysListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <app-id>",
		Short: "List recent deployments for an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}

			orgID, err := requireOrgID(cmd.Context(), cmd, c, r)
			if err != nil {
				return err
			}

			deploys, err := c.ListDeployments(cmd.Context(), orgID, args[0])
			if err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(deploys); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}

// newDeploysStatusCmd returns `deploys status <app-id> <deploy-id>`.
func newDeploysStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <app-id> <deploy-id>",
		Short: "Get the status of a specific deployment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}

			orgID, err := requireOrgID(cmd.Context(), cmd, c, r)
			if err != nil {
				return err
			}

			deploy, err := c.GetDeployment(cmd.Context(), orgID, args[0], args[1])
			if err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(deploy); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}
