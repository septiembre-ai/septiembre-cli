package cli

import (
	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewLogsCmd returns the `logs <app-id>` command.
func NewLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <app-id>",
		Short: "Fetch a log snapshot for an application (non-streaming)",
		Long: `Fetch a recent snapshot of CloudWatch log events for a Lambda-backed application.

The command returns immediately with the latest log events. Streaming / watch
mode is deferred to a future release.

If the environment has not been provisioned yet, an empty events array is returned
(not an error).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envID, _ := cmd.Flags().GetString("env-id")

			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}

			orgID, err := requireOrgID(cmd.Context(), cmd, c, r)
			if err != nil {
				return err
			}

			logs, err := c.GetLogs(cmd.Context(), orgID, args[0], envID)
			if err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(logs); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
	cmd.Flags().String("env-id", "", "Environment UUID (defaults to the production environment when empty)")
	return cmd
}
