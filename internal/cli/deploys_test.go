package cli_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/septiembre-ai/septiembre-cli/internal/output"
)

const (
	testOrgSlug = "acme"
	testOrgID   = "org-1"
	testAppID   = "app-1"
	testDeployID = "deploy-1"
)

func deploys_mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/orgs", orgsHandler(testOrgSlug, testOrgID))
	return mux
}

// ---- deploys trigger: success ----

func TestDeploysTrigger_Success(t *testing.T) {
	mux := deploys_mux()
	mux.HandleFunc("/api/v1/orgs/"+testOrgID+"/apps/"+testAppID+"/deployments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"` + testDeployID + `","app_id":"` + testAppID + `","status":"pending","release_tag":"v1.0.0","environment_id":"","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`))
	})
	srv := newTestServer(t, mux)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("deploys", "trigger", testAppID, "--org", testOrgSlug, "--tag", "v1.0.0")

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("deploys trigger: want exit 0, got %d", got)
	}
	var got map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("deploys trigger: stdout not valid JSON: %v\noutput: %s", jsonErr, outBuf)
	}
	if id, _ := got["id"].(string); id != testDeployID {
		t.Errorf("deploys trigger: want id %q, got %q", testDeployID, id)
	}
	if status, _ := got["status"].(string); status != "pending" {
		t.Errorf("deploys trigger: want status pending, got %q", status)
	}
}

// ---- deploys list: success ----

func TestDeploysList_Success(t *testing.T) {
	mux := deploys_mux()
	mux.HandleFunc("/api/v1/orgs/"+testOrgID+"/apps/"+testAppID+"/deployments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "expected GET", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"` + testDeployID + `","app_id":"` + testAppID + `","status":"success","release_tag":"v1.0.0","environment_id":"","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}]`))
	})
	srv := newTestServer(t, mux)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("deploys", "list", testAppID, "--org", testOrgSlug)

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("deploys list: want exit 0, got %d", got)
	}
	var got []map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("deploys list: stdout not valid JSON array: %v\noutput: %s", jsonErr, outBuf)
	}
	if len(got) == 0 {
		t.Errorf("deploys list: expected at least one deployment")
	}
}

// ---- deploys status: success ----

func TestDeploysStatus_Success(t *testing.T) {
	mux := deploys_mux()
	mux.HandleFunc("/api/v1/orgs/"+testOrgID+"/apps/"+testAppID+"/deployments/"+testDeployID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"` + testDeployID + `","app_id":"` + testAppID + `","status":"success","release_tag":"v1.0.0","environment_id":"","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}`))
	})
	srv := newTestServer(t, mux)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("deploys", "status", testAppID, testDeployID, "--org", testOrgSlug)

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("deploys status: want exit 0, got %d", got)
	}
	var got map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("deploys status: stdout not valid JSON: %v\noutput: %s", jsonErr, outBuf)
	}
	if status, _ := got["status"].(string); status != "success" {
		t.Errorf("deploys status: want status success, got %q", status)
	}
}

// ---- logs: success (non-streaming snapshot) ----
// When --env-id is omitted the CLI resolves the app's default environment via
// GET /environments first, then fetches logs at the env-scoped path — the
// exact route shape chi serves in cloud-api (no empty path segments).

const testEnvID = "env-1"

func registerEnvironmentsHandler(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/orgs/"+testOrgID+"/apps/"+testAppID+"/environments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"` + testEnvID + `","app_id":"` + testAppID + `","name":"production","branch":"main","is_default":true,"is_active":true}]`))
	})
}

func TestLogs_SnapshotResolvesDefaultEnv(t *testing.T) {
	mux := deploys_mux()
	registerEnvironmentsHandler(mux)
	mux.HandleFunc("/api/v1/orgs/"+testOrgID+"/apps/"+testAppID+"/environments/"+testEnvID+"/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"function_name":"my-fn","log_group":"/aws/lambda/my-fn","events":[{"timestamp":1700000000000,"message":"hello"}]}`))
	})
	srv := newTestServer(t, mux)
	outBuf, _, exec := newTestRoot(t, srv)

	err := exec("logs", testAppID, "--org", testOrgSlug)

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("logs: want exit 0, got %d", got)
	}
	var got map[string]any
	if jsonErr := json.NewDecoder(outBuf).Decode(&got); jsonErr != nil {
		t.Fatalf("logs: stdout not valid JSON: %v\noutput: %s", jsonErr, outBuf)
	}
}

// ---- logs: explicit --env-id skips environment resolution ----

func TestLogs_ExplicitEnvID(t *testing.T) {
	mux := deploys_mux()
	mux.HandleFunc("/api/v1/orgs/"+testOrgID+"/apps/"+testAppID+"/environments/"+testEnvID+"/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"function_name":"my-fn","log_group":"/aws/lambda/my-fn","events":[]}`))
	})
	srv := newTestServer(t, mux)
	_, _, exec := newTestRoot(t, srv)

	err := exec("logs", testAppID, "--org", testOrgSlug, "--env-id", testEnvID)

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("logs --env-id: want exit 0, got %d", got)
	}
}

// ---- logs: 204 No Content returns empty (not an error) ----

func TestLogs_EmptyOn204(t *testing.T) {
	mux := deploys_mux()
	registerEnvironmentsHandler(mux)
	mux.HandleFunc("/api/v1/orgs/"+testOrgID+"/apps/"+testAppID+"/environments/"+testEnvID+"/logs", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	srv := newTestServer(t, mux)
	_, _, exec := newTestRoot(t, srv)

	err := exec("logs", testAppID, "--org", testOrgSlug)

	if got := exitCode(err); got != output.ExitOK {
		t.Errorf("logs 204: want exit 0, got %d", got)
	}
}
