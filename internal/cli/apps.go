package cli

import (
	"github.com/septiembre-ai/septiembre-cli/internal/credentials"
	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewAppsCmd returns the `apps` command group.
func NewAppsCmd() *cobra.Command {
	apps := &cobra.Command{
		Use:   "apps",
		Short: "Manage applications",
		Long: `Commands for listing and inspecting Septiembre applications.

AGENT USAGE
  Default output is JSON. Pipe to jq for field extraction:
    septiembre apps list | jq '.[].id'
    septiembre apps get <app-id> --org <slug> | jq '.subdomain'`,
	}
	apps.AddCommand(newAppsListCmd())
	apps.AddCommand(newAppsGetCmd())
	return apps
}

// newAppsListCmd returns `apps list [--org <slug>]`.
// Without --org (and no config default) the flat GET /api/v1/apps is used to
// list all apps visible to the caller across every org. With --org only apps
// in that specific org are returned.
func newAppsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List applications (all orgs by default, or scoped with --org)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}

			// If a slug is available (flag or config), use the org-scoped list.
			slug, _ := cmd.Root().PersistentFlags().GetString("org")
			if slug == "" {
				slug = credentials.DefaultOrgSlugFromPath(configFlagOrDefault(cmd))
			}

			if slug != "" {
				orgID, oErr := requireOrgID(cmd.Context(), cmd, c, r)
				if oErr != nil {
					return oErr
				}
				apps, aErr := c.ListApps(cmd.Context(), orgID)
				if aErr != nil {
					return handleAPIError(r, aErr)
				}
				if err := r.Render(apps); err != nil {
					return &ExitError{Code: output.ExitGeneral}
				}
				return nil
			}

			// No org context — use the flat cross-org endpoint.
			apps, aErr := c.ListAllApps(cmd.Context())
			if aErr != nil {
				return handleAPIError(r, aErr)
			}
			if err := r.Render(apps); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}

// newAppsGetCmd returns `apps get <app-id> --org <slug>`.
// --org is required: the API path is /orgs/{orgID}/apps/{appID}.
func newAppsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <app-id>",
		Short: "Get details for a specific application (requires --org)",
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

			app, err := c.GetApp(cmd.Context(), orgID, args[0])
			if err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(app); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}
