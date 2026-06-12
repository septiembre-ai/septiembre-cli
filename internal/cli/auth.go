package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewAuthCmd returns the `auth` parent command with its subcommands wired in.
func NewAuthCmd() *cobra.Command {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication and personal access tokens",
		Long: `Commands for authentication and personal access token (PAT) management.

AGENT USAGE
  For CI and agents, create a PAT and set SEPTIEMBRE_TOKEN:
    septiembre auth token create --name ci-deploy
    export SEPTIEMBRE_TOKEN=sapi_<hex>

  Device-flow browser login is coming in a future release.`,
	}

	auth.AddCommand(newAuthLoginCmd())
	auth.AddCommand(newAuthWhoamiCmd())
	auth.AddCommand(newAuthTokenCmd())
	return auth
}

// newAuthLoginCmd returns the `auth login` stub command (spec B-04).
// Device-flow login is a future fast-follow; v1 uses PATs for headless auth.
func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate interactively (coming soon — use SEPTIEMBRE_TOKEN for now)",
		Long: `Browser-based device-flow login is not yet available.

For CI and agents, use a personal access token instead:
  1. Create a token:  septiembre auth token create --name ci-deploy
  2. Set env var:     export SEPTIEMBRE_TOKEN=sapi_<hex>
  3. Verify:         septiembre auth whoami

Tokens are created at: POST https://api.septiembre.ai/api/v1/auth/tokens`,
		RunE: func(cmd *cobra.Command, args []string) error {
			enc := json.NewEncoder(cmd.ErrOrStderr())
			enc.SetIndent("", "  ")
			_ = enc.Encode(map[string]any{
				"error": "device-flow login is not yet available — " +
					"set SEPTIEMBRE_TOKEN to a PAT (sapi_<hex>) for CI/agents. " +
					"Create one: septiembre auth token create --name <name>",
				"code": "not_implemented",
			})
			return &ExitError{Code: output.ExitGeneral}
		},
	}
}

// newAuthWhoamiCmd returns the `auth whoami` command.
func newAuthWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the identity of the currently authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}
			user, err := c.Whoami(cmd.Context())
			if err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(user); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}

// newAuthTokenCmd returns the `auth token` parent command.
func newAuthTokenCmd() *cobra.Command {
	token := &cobra.Command{
		Use:   "token",
		Short: "Manage personal access tokens (PATs)",
	}
	token.AddCommand(newAuthTokenCreateCmd())
	token.AddCommand(newAuthTokenListCmd())
	token.AddCommand(newAuthTokenRevokeCmd())
	return token
}

// createTokenOutput is the stdout shape for `auth token create`.
// The Token field holds the raw PAT value; the Warning field reminds the user
// to store it immediately (the raw value is never shown again).
type createTokenOutput struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Token     string     `json:"token"`
	LastFour  string     `json:"last_four"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Warning   string     `json:"warning"`
}

// newAuthTokenCreateCmd returns the `auth token create` command (spec B-04a).
func newAuthTokenCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a personal access token (shown once)",
		Long: `Create a personal access token (PAT) for headless authentication.

The raw token (sapi_<hex>) is printed exactly once in the JSON response.
Store it immediately — it cannot be retrieved again.

EXIT CODES
  0  token created
  2  not authenticated
  4  validation error (e.g. future expires_at required)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			expiresStr, _ := cmd.Flags().GetString("expires-at")

			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}

			var expiresAt *time.Time
			if expiresStr != "" {
				t, parseErr := time.Parse(time.RFC3339, expiresStr)
				if parseErr != nil {
					r.RenderError(
						fmt.Sprintf("invalid --expires-at value %q: must be RFC 3339 (e.g. 2026-12-31T00:00:00Z)", expiresStr),
						"validation_error",
						-1,
					)
					return &ExitError{Code: output.ExitValidation}
				}
				expiresAt = &t
			}

			resp, err := c.CreateToken(cmd.Context(), name, expiresAt)
			if err != nil {
				return handleAPIError(r, err)
			}

			out := createTokenOutput{
				ID:        resp.ID,
				Name:      resp.Name,
				Token:     resp.Token,
				LastFour:  resp.LastFour,
				Status:    resp.Status,
				ExpiresAt: resp.ExpiresAt,
				CreatedAt: resp.CreatedAt,
				Warning:   "Token shown once — store it securely. It cannot be retrieved again.",
			}
			if err := r.Render(out); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
	cmd.Flags().String("name", "cli-token", "Human-readable token name (e.g. ci-deploy)")
	cmd.Flags().String("expires-at", "", "Token expiry in RFC 3339 format (e.g. 2026-12-31T00:00:00Z). Omit for no expiry.")
	return cmd
}

// newAuthTokenListCmd returns the `auth token list` command (spec A-03).
func newAuthTokenListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List personal access tokens owned by the current user",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}
			tokens, err := c.ListTokens(cmd.Context())
			if err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(tokens); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}

// newAuthTokenRevokeCmd returns the `auth token revoke <id>` command (spec A-04).
func newAuthTokenRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <token-id>",
		Short: "Revoke a personal access token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, r, err := newAuthenticatedClient(cmd)
			if err != nil {
				return err
			}
			if err := c.RevokeToken(cmd.Context(), args[0]); err != nil {
				return handleAPIError(r, err)
			}
			if err := r.Render(map[string]string{"message": "token revoked"}); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}
