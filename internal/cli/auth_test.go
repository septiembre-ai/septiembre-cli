package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/septiembre-ai/septiembre-cli/internal/cli"
	"github.com/septiembre-ai/septiembre-cli/internal/output"
)

// ---- shared test helpers ----

// newTestServer starts a local httptest server backed by handler and returns
// its URL and a cleanup function.
func newTestServer(t *testing.T, handler http.Handler) string {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv.URL
}

// newTestRoot returns a root cobra command wired to a test server.
// SEPTIEMBRE_TOKEN and SEPTIEMBRE_API_URL are set via t.Setenv so they are
// restored after the test. stdout and stderr are captured in the returned buffers.
func newTestRoot(t *testing.T, serverURL string) (*bytes.Buffer, *bytes.Buffer, func(args ...string) error) {
	t.Helper()
	t.Setenv("SEPTIEMBRE_TOKEN", "sapi_test_deadbeef1234567890123456789012")
	if serverURL != "" {
		t.Setenv("SEPTIEMBRE_API_URL", serverURL)
	}

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	exec := func(args ...string) error {
		root := cli.NewRootCmd()
		root.SetOut(outBuf)
		root.SetErr(errBuf)
		root.SetArgs(args)
		outBuf.Reset()
		errBuf.Reset()
		return root.Execute()
	}
	return outBuf, errBuf, exec
}

// exitCode extracts the exit code from an *ExitError, returns 0 for nil.
func exitCode(err error) int {
	if err == nil {
		return output.ExitOK
	}
	var ee *cli.ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return output.ExitGeneral
}

// ---- auth login stub (spec: must not look like a crash) ----

func TestAuthLogin_Stub(t *testing.T) {
	_, errBuf, exec := newTestRoot(t, "")
	err := exec("auth", "login")

	if got := exitCode(err); got != output.ExitGeneral {
		t.Errorf("auth login: want exit %d, got %d", output.ExitGeneral, got)
	}
	var env map[string]any
	if jsonErr := json.NewDecoder(errBuf).Decode(&env); jsonErr != nil {
		t.Fatalf("auth login: stderr is not JSON: %v\nstderr: %s", jsonErr, errBuf)
	}
	if code, _ := env["code"].(string); code != "not_implemented" {
		t.Errorf("auth login: want code not_implemented, got %q", code)
	}
}

// ---- spec B-03b: no credentials → exit 2 ----

func TestNoCredentials_ExitAuth(t *testing.T) {
	t.Setenv("SEPTIEMBRE_TOKEN", "")
	t.Setenv("SEPTIEMBRE_ORG", "")

	_, errBuf, _ := newTestRoot(t, "")
	// Override the token env again because newTestRoot sets it.
	t.Setenv("SEPTIEMBRE_TOKEN", "")

	root := cli.NewRootCmd()
	outBuf := new(bytes.Buffer)
	eb := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(eb)
	root.SetArgs([]string{"--config", "/nonexistent/septiembre.yaml", "orgs", "list"})
	err := root.Execute()

	_ = errBuf // not used for this assertion path

	if got := exitCode(err); got != output.ExitAuth {
		t.Errorf("no credentials: want exit %d (auth), got %d; stderr: %s", output.ExitAuth, got, eb)
	}

	var env map[string]any
	if jsonErr := json.NewDecoder(eb).Decode(&env); jsonErr != nil {
		t.Fatalf("no credentials: stderr not JSON: %v\nstderr: %s", jsonErr, eb)
	}
	if code, _ := env["code"].(string); code != "auth_error" {
		t.Errorf("no credentials: want code auth_error, got %q", code)
	}
}

// ---- spec B-09a: --help --json parseable ----

func TestHelpJSON_Parseable(t *testing.T) {
	outBuf, _, exec := newTestRoot(t, "")
	err := exec("--help", "--json")

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("--help --json: want exit 0, got %d", got)
	}
	var got map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("--help --json output not valid JSON: %v\noutput: %s", jsonErr, outBuf)
	}
	if _, ok := got["commands"]; !ok {
		t.Errorf("--help --json missing 'commands' key; got %v", got)
	}
}

// ---- spec B-10a: --version emits JSON ----

func TestVersion_JSON(t *testing.T) {
	outBuf, _, exec := newTestRoot(t, "")
	err := exec("--version")

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("--version: want exit 0, got %d", got)
	}
	var got map[string]string
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("--version output not valid JSON: %v\noutput: %s", jsonErr, outBuf)
	}
	for _, field := range []string{"version", "commit", "built_at"} {
		if _, ok := got[field]; !ok {
			t.Errorf("--version missing field %q; keys: %v", field, got)
		}
	}
}

// ---- spec B-02a: auth failure → exit 2 ----

func TestAuthFailure_ExitCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	})
	srv := newTestServer(t, handler)
	_, errBuf, exec := newTestRoot(t, srv)

	err := exec("orgs", "list")

	if got := exitCode(err); got != output.ExitAuth {
		t.Errorf("401 response: want exit %d (auth), got %d", output.ExitAuth, got)
	}
	var env map[string]any
	if jsonErr := json.NewDecoder(errBuf).Decode(&env); jsonErr != nil {
		t.Fatalf("401 response: stderr not JSON: %v\nstderr: %s", jsonErr, errBuf)
	}
	if code, _ := env["code"].(string); code != "auth_error" {
		t.Errorf("401 response: want code auth_error, got %q", code)
	}
}

// ---- spec B-04a: auth token create shows raw token with warning field ----

func TestAuthTokenCreate_ShowsTokenOnce(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": "tok-1",
			"name": "ci",
			"token": "sapi_abc123def456",
			"last_four": "f456",
			"status": "active",
			"created_at": "2024-01-01T00:00:00Z"
		}`))
	})
	srv := newTestServer(t, handler)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("auth", "token", "create", "--name", "ci")

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("token create: want exit 0, got %d", got)
	}

	var got map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("token create: stdout not JSON: %v\noutput: %s", jsonErr, outBuf)
	}
	if token, _ := got["token"].(string); token != "sapi_abc123def456" {
		t.Errorf("token create: want token sapi_abc123def456, got %q", token)
	}
	if warning, _ := got["warning"].(string); warning == "" {
		t.Errorf("token create: missing 'warning' field in output")
	}
}

// ---- spec B-01a: default output is JSON ----

func TestOrgsListJSON_DefaultOutput(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"org-1","name":"Acme","slug":"acme","created_at":"2024-01-01T00:00:00Z"}]`))
	})
	srv := newTestServer(t, handler)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("orgs", "list")

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("orgs list: want exit 0, got %d", got)
	}
	var got []map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("orgs list: stdout not valid JSON array: %v\noutput: %s", jsonErr, outBuf)
	}
	if len(got) == 0 {
		t.Errorf("orgs list: expected at least one org in output")
	}
}
