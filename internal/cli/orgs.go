package cli

import (
	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewOrgsCmd returns the `orgs` command group.
func NewOrgsCmd() *cobra.Command {
	orgs := &cobra.Command{
		Use:   "orgs",
		Short: "Manage organizations",
	}
	orgs.AddCommand(newOrgsListCmd())
	return orgs
}

// newOrgsListCmd returns `orgs list` which prints all orgs the caller belongs to.
func newOrgsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organizations you belong to",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}
			orgs, err := c.ListOrgs(cmd.Context())
			if err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(orgs); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}
