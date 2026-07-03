package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/septiembre-ai/septiembre-cli/internal/authflow"
	"github.com/septiembre-ai/septiembre-cli/internal/client"
	"github.com/septiembre-ai/septiembre-cli/internal/credentials"
	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"github.com/spf13/cobra"
)

// runAuthFlow runs one browser login attempt. It is a package-level var so
// tests can override it with a stub that injects a fake BrowserOpener and
// HTTPClient into authflow.Config — no real browser or Cognito server is
// ever exercised by this package's own tests.
var runAuthFlow = authflow.Run

// NewLoginCmd returns the `septiembre login` command: browser-based
// Authorization Code + PKCE login against Cognito, minting a PAT on success.
// This complements (does not replace) `auth login`'s PAT-bootstrap flow —
// see NewAuthCmd.
func NewLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate via browser login (Cognito Authorization Code + PKCE)",
		Long: `Authenticate by opening a browser to Cognito's hosted UI, then mint and save a
personal access token (PAT).

Re-running this command overwrites the stored token only after a new PAT is
minted successfully; a failed attempt never alters the existing stored token.

VALIDATION / INPUTS
  Cognito configuration: resolved via SEPTIEMBRE_COGNITO_DOMAIN /
  SEPTIEMBRE_COGNITO_CLIENT_ID env vars, then the "cognito_domain" /
  "cognito_client_id" config keys, then the build-time default. The client
  id must be provisioned (see docs/cognito-cli-client.md in the cloud-api repo); an empty client
  id fails fast before any browser or network call.

EXIT CODES
  0  authenticated
  1  all loopback callback ports busy, or callback timeout
  2  identity provider error, code exchange failure, or invalid callback
  4  state (CSRF) mismatch, or Cognito client id not configured
  *  PAT minting or /auth/me failure — mapped via the existing API error codes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := output.NewRenderer(OutputFormat(cmd), cmd.OutOrStdout(), cmd.ErrOrStderr())

			configPath := configFlagOrDefault(cmd)
			cognitoDomain := credentials.CognitoDomainFromPath(configPath)
			clientID := credentials.CognitoClientIDFromPath(configPath)
			if clientID == "" {
				r.RenderError(
					"cognito client id not configured — this build was not provisioned with a Cognito app client (see docs/cognito-cli-client.md in the cloud-api repo)",
					"validation_error",
					-1,
				)
				return &ExitError{Code: output.ExitValidation}
			}

			result, err := runAuthFlow(cmd.Context(), authflow.Config{
				CognitoDomain: cognitoDomain,
				ClientID:      clientID,
				Writer:        cmd.ErrOrStderr(),
			})
			if err != nil {
				return mapAuthFlowError(r, err)
			}

			apiBaseURL := credentials.APIBaseURLFromPath(configPath)
			hostname, hostErr := os.Hostname()
			if hostErr != nil || hostname == "" {
				hostname = "unknown"
			}

			mintClient := client.New(apiBaseURL, result.IDToken)
			tokenResp, err := mintClient.CreateToken(cmd.Context(), "cli-"+hostname, nil)
			if err != nil {
				return handleAPIError(r, err)
			}

			if err := credentials.SaveToken(configPath, tokenResp.Token); err != nil {
				r.RenderError(fmt.Sprintf("failed to save token: %v", err), "config_error", 500)
				return &ExitError{Code: output.ExitGeneral}
			}

			whoamiClient := client.New(apiBaseURL, tokenResp.Token)
			user, err := whoamiClient.Whoami(cmd.Context())
			if err != nil {
				return handleAPIError(r, err)
			}

			if err := r.Render(loginOutput{
				Status:     "authenticated",
				ConfigPath: configPath,
				User:       user,
			}); err != nil {
				return &ExitError{Code: output.ExitGeneral}
			}
			return nil
		},
	}
}

// mapAuthFlowError maps authflow.Run's typed errors to the CLI's exit-code
// contract using errors.As (never string matching or type equality), per
// the design's Error Taxonomy. CallbackError is deliberately bucketed with
// IdPError/ExchangeError as auth_error/exit 2 — a malformed callback is an
// auth-flow failure, not a client-side validation error. Exit-1 rows follow
// the existing config_error pattern: httpStatus 500 is cosmetic (Renderer's
// own mapping also yields exit 1), and the explicit *ExitError governs the
// process exit.
func mapAuthFlowError(r *output.Renderer, err error) error {
	var portsBusy *authflow.PortsBusyError
	if errors.As(err, &portsBusy) {
		r.RenderError(err.Error(), "port_error", 500)
		return &ExitError{Code: output.ExitGeneral}
	}

	var timeoutErr *authflow.TimeoutError
	if errors.As(err, &timeoutErr) {
		r.RenderError(err.Error(), "timeout_error", 500)
		return &ExitError{Code: output.ExitGeneral}
	}

	var idpErr *authflow.IdPError
	if errors.As(err, &idpErr) {
		r.RenderError(err.Error(), "auth_error", 401)
		return &ExitError{Code: output.ExitAuth}
	}

	var exchangeErr *authflow.ExchangeError
	if errors.As(err, &exchangeErr) {
		r.RenderError(err.Error(), "auth_error", 401)
		return &ExitError{Code: output.ExitAuth}
	}

	var callbackErr *authflow.CallbackError
	if errors.As(err, &callbackErr) {
		r.RenderError(err.Error(), "auth_error", 401)
		return &ExitError{Code: output.ExitAuth}
	}

	var stateMismatch *authflow.StateMismatchError
	if errors.As(err, &stateMismatch) {
		r.RenderError(err.Error(), "validation_error", -1)
		return &ExitError{Code: output.ExitValidation}
	}

	// Unreachable in practice — Run only ever returns the six typed errors
	// above (see authflow.Run's doc comment) — but handled defensively,
	// mirroring handleAPIError's own fallback for unexpected errors.
	r.RenderError(err.Error(), "general_error", 0)
	return &ExitError{Code: output.ExitGeneral}
}
