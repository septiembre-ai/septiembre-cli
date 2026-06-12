package cli_test

import (
	"net/http"
	"testing"

	"github.com/septiembre-ai/septiembre-cli/internal/output"
	"encoding/json"
)

// orgsAndAppsHandler returns an http.ServeMux that simulates an org + apps pair.
func orgsAndAppsHandler(orgSlug, orgID string, appsJSON, appJSON string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/orgs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"` + orgID + `","name":"Acme","slug":"` + orgSlug + `","created_at":"2024-01-01T00:00:00Z"}]`))
	})
	if appsJSON != "" {
		mux.HandleFunc("/api/v1/orgs/"+orgID+"/apps", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(appsJSON))
		})
	}
	if appJSON != "" {
		mux.HandleFunc("/api/v1/orgs/"+orgID+"/apps/app-1", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(appJSON))
		})
	}
	return mux
}

// ---- spec B-01a: apps list default JSON output ----

func TestAppsList_JSONOutput(t *testing.T) {
	const appsJSON = `[{"id":"app-1","org_id":"org-1","name":"my-app","label":"my-app","type":"api","aws_region":"us-east-1","github_repo_full":"acme/repo","github_branch":"main","subdomain":"my-app","domain_status":"active","visibility":"private","is_active":true,"created_at":"2024-01-01T00:00:00Z"}]`
	handler := orgsAndAppsHandler("acme", "org-1", appsJSON, "")
	srv := newTestServer(t, handler)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("apps", "list", "--org", "acme")

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("apps list: want exit 0, got %d", got)
	}
	var got []map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("apps list: stdout not valid JSON array: %v\noutput: %s", jsonErr, outBuf)
	}
	if len(got) == 0 {
		t.Errorf("apps list: expected at least one app in output")
	}
}

// ---- spec B-01a: apps list without --org uses flat endpoint ----

func TestAppsList_NoOrg_UsesFlatEndpoint(t *testing.T) {
	const appsJSON = `[{"id":"app-1","org_id":"org-1","name":"my-app","label":"my-app","type":"api","aws_region":"us-east-1","github_repo_full":"acme/repo","github_branch":"main","subdomain":"my-app","domain_status":"active","visibility":"private","is_active":true,"created_at":"2024-01-01T00:00:00Z"}]`
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/apps", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(appsJSON))
	})
	srv := newTestServer(t, mux)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("apps", "list")

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("apps list (no org): want exit 0, got %d", got)
	}
	var got []map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("apps list (no org): stdout not valid JSON: %v\noutput: %s", jsonErr, outBuf)
	}
}

// ---- spec B-02b: apps get not-found → exit 3 ----

func TestAppsGet_NotFound_ExitCode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/orgs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"org-1","name":"Acme","slug":"acme","created_at":"2024-01-01T00:00:00Z"}]`))
	})
	mux.HandleFunc("/api/v1/orgs/org-1/apps/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"app not found"}`))
	})
	srv := newTestServer(t, mux)
	_, errBuf, exec := newTestRoot(t, srv)

	err := exec("apps", "get", "nonexistent", "--org", "acme")

	if got := exitCode(err); got != output.ExitNotFound {
		t.Errorf("404 response: want exit %d (not_found), got %d; stderr: %s", output.ExitNotFound, got, errBuf)
	}
}

// ---- apps get: missing --org → exit 4 (validation) ----

func TestAppsGet_MissingOrg_ExitValidation(t *testing.T) {
	srv := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve orgs as empty list so org resolution returns not-found.
		// But actually with missing slug the validation error fires before the API call.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	_, errBuf, exec := newTestRoot(t, srv)

	err := exec("apps", "get", "app-1")

	if got := exitCode(err); got != output.ExitValidation {
		t.Errorf("missing --org: want exit %d (validation), got %d; stderr: %s", output.ExitValidation, got, errBuf)
	}
	var env map[string]any
	if jsonErr := json.NewDecoder(errBuf).Decode(&env); jsonErr != nil {
		t.Fatalf("missing --org: stderr not JSON: %v\nstderr: %s", jsonErr, errBuf)
	}
	if code, _ := env["code"].(string); code != "validation_error" {
		t.Errorf("missing --org: want code validation_error, got %q", code)
	}
}

// ---- apps get: success ----

func TestAppsGet_Success(t *testing.T) {
	const appJSON = `{"id":"app-1","org_id":"org-1","name":"my-app","label":"my-app","type":"api","aws_region":"us-east-1","github_repo_full":"acme/repo","github_branch":"main","subdomain":"my-app","domain_status":"active","visibility":"private","is_active":true,"created_at":"2024-01-01T00:00:00Z"}`
	handler := orgsAndAppsHandler("acme", "org-1", "", appJSON)
	srv := newTestServer(t, handler)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("apps", "get", "app-1", "--org", "acme")

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("apps get: want exit 0, got %d", got)
	}
	var got map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("apps get: stdout not valid JSON: %v\noutput: %s", jsonErr, outBuf)
	}
	if id, _ := got["id"].(string); id != "app-1" {
		t.Errorf("apps get: want id app-1, got %q", id)
	}
}
