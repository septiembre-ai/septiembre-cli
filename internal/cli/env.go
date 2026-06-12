package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"github.com/spf13/cobra"
)

// envEntry is the CLI representation of a single environment variable.
// Exported for test assertions.
type envEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewEnvCmd returns the `env` command group.
func NewEnvCmd() *cobra.Command {
	env := &cobra.Command{
		Use:   "env",
		Short: "Manage application environment variables",
		Long: `Commands for reading and writing application environment variables.

SECURITY
  Values are masked (***) by default. Use --reveal to print plaintext values.
  env set replaces ALL env vars (PUT semantics) — omitted keys are deleted.

AGENT USAGE
  septiembre env get <app-id> --org <slug> | jq '.[] | select(.key=="API_KEY")'
  septiembre env set <app-id> --org <slug> KEY=value OTHER=value2`,
	}
	env.AddCommand(newEnvGetCmd())
	env.AddCommand(newEnvSetCmd())
	return env
}

// newEnvGetCmd returns `env get <app-id>`.
// Values are masked by default; --reveal shows plaintext.
func newEnvGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <app-id>",
		Short: "Get environment variables for an app (values masked by default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reveal, _ := cmd.Flags().GetBool("reveal")

			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}

			orgID, err := requireOrgID(cmd.Context(), cmd, c, r)
			if err != nil {
				return err
			}

			env, err := c.GetEnv(cmd.Context(), orgID, args[0])
			if err != nil {
				return handleAPIError(r, err)
			}

			entries := make([]envEntry, 0, len(env))
			for k, v := range env {
				val := "***"
				if reveal {
					val = v
				}
				entries = append(entries, envEntry{Key: k, Value: val})
			}
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Key < entries[j].Key
			})

			if err := r.Render(entries); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
	cmd.Flags().Bool("reveal", false, "Show plaintext env values (default: masked as ***)")
	return cmd
}

// newEnvSetCmd returns `env set <app-id> KEY=VALUE...`.
// This is a full replacement (PUT): omitted keys are deleted.
func newEnvSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <app-id> KEY=VALUE [KEY=VALUE...]",
		Short: "Set environment variables for an app (full replace — omitted keys are deleted)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID := args[0]
			pairs := args[1:]

			vars := make(map[string]string, len(pairs))
			for _, kv := range pairs {
				idx := strings.IndexByte(kv, '=')
				if idx < 0 {
					r := output.NewRenderer(OutputFormat(cmd), cmd.OutOrStdout(), cmd.ErrOrStderr())
					r.RenderError(
						fmt.Sprintf("invalid argument %q: expected KEY=VALUE format", kv),
						"validation_error",
						-1,
					)
					return &ExitError{Code: output.ExitValidation}
				}
				key := kv[:idx]
				if key == "" {
					r := output.NewRenderer(OutputFormat(cmd), cmd.OutOrStdout(), cmd.ErrOrStderr())
					r.RenderError(
						fmt.Sprintf("invalid argument %q: key must not be empty", kv),
						"validation_error",
						-1,
					)
					return &ExitError{Code: output.ExitValidation}
				}
				vars[key] = kv[idx+1:]
			}

			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}

			orgID, err := requireOrgID(cmd.Context(), cmd, c, r)
			if err != nil {
				return err
			}

			if err := c.SetEnv(cmd.Context(), orgID, appID, vars); err != nil {
				return handleAPIError(r, err)
			}

			if err := r.Render(map[string]string{"message": "env updated"}); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}
