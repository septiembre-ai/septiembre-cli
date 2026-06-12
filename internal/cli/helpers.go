package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/septiembre-ai/septiembre-cli/internal/client"
	"github.com/septiembre-ai/septiembre-cli/internal/credentials"
	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"github.com/spf13/cobra"
)

// ExitError is returned by command RunE to signal a specific exit code.
// Commands write the error to stderr via the Renderer before returning ExitError
// so main.go calls os.Exit without printing again.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}

// configFlagOrDefault returns the --config flag value, falling back to
// credentials.DefaultConfigPath() when the flag is absent or empty.
func configFlagOrDefault(cmd *cobra.Command) string {
	if path, _ := cmd.Root().PersistentFlags().GetString("config"); path != "" {
		return path
	}
	return credentials.DefaultConfigPath()
}

// newAuthenticatedClient resolves credentials and returns a typed API client
// together with a Renderer that writes to cmd's stdout/stderr.
//
// On credential failure it writes an auth-error envelope to stderr and returns
// a non-nil error (an *ExitError with Code=ExitAuth). The caller must return
// that error from RunE without further action.
func newAuthenticatedClient(cmd *cobra.Command) (*client.Client, *output.Renderer, error) {
	r := output.NewRenderer(OutputFormat(cmd), cmd.OutOrStdout(), cmd.ErrOrStderr())

	configPath := configFlagOrDefault(cmd)
	token, err := credentials.ResolveWithConfigPath(configPath)
	if err != nil {
		r.RenderError(
			"no credentials found — set SEPTIEMBRE_TOKEN or run 'septiembre auth token create'",
			"auth_error",
			401,
		)
		return nil, r, &ExitError{Code: output.ExitAuth}
	}

	baseURL := credentials.APIBaseURLFromPath(configPath)
	c := client.New(baseURL, token)
	return c, r, nil
}

// handleAPIError renders an *client.APIError to stderr and returns an *ExitError
// with the appropriate exit code. For unexpected non-API errors it falls back to
// ExitGeneral.
func handleAPIError(r *output.Renderer, err error) error {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		code := r.RenderError(apiErr.Message, apiErr.Code, apiErr.HTTPStatus)
		return &ExitError{Code: code}
	}
	// Unexpected error (e.g. JSON decode bug) — treat as general.
	r.RenderError(err.Error(), "general_error", 0)
	return &ExitError{Code: output.ExitGeneral}
}

// requireOrgID resolves the org slug from --org flag (or config default) and
// returns the org UUID by calling ListOrgs. It renders a validation error and
// returns an *ExitError when the slug is absent or not found.
func requireOrgID(ctx context.Context, cmd *cobra.Command, c *client.Client, r *output.Renderer) (string, error) {
	slug, _ := cmd.Root().PersistentFlags().GetString("org")
	if slug == "" {
		slug = credentials.DefaultOrgSlugFromPath(configFlagOrDefault(cmd))
	}
	if slug == "" {
		r.RenderError(
			"org is required — specify --org <slug> or set 'org' in ~/.config/septiembre/config.yaml",
			"validation_error",
			-1,
		)
		return "", &ExitError{Code: output.ExitValidation}
	}

	orgs, err := c.ListOrgs(ctx)
	if err != nil {
		return "", handleAPIError(r, err)
	}
	for _, o := range orgs {
		if o.Slug == slug {
			return o.ID, nil
		}
	}

	r.RenderError(fmt.Sprintf("org %q not found", slug), "not_found", 404)
	return "", &ExitError{Code: output.ExitNotFound}
}
