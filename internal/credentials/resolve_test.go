package credentials

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- Scenario B-03a: SEPTIEMBRE_TOKEN env var takes priority ---

func TestResolve_EnvVarTakesPriority(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	// Write a config file with a different token.
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"token": "file-token"})

	t.Setenv("SEPTIEMBRE_TOKEN", "env-token")

	// Provide a config path that has a different token; env must win.
	token, err := resolveWithConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "env-token" {
		t.Errorf("token = %q, want 'env-token' (env var should win)", token)
	}
}

// --- Scenario B-03b: No credentials → ErrNoCredentials ---

func TestResolve_NoCredentials(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml") // does not exist

	// Clear env so no env var is present.
	t.Setenv("SEPTIEMBRE_TOKEN", "")

	_, err := resolveWithConfig(cfgPath)
	if err != ErrNoCredentials {
		t.Errorf("err = %v, want ErrNoCredentials", err)
	}
}

// --- Config file token resolution ---

func TestResolve_ConfigFileToken(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"token": "file-token-abc"})

	// No env var — must fall through to config file.
	t.Setenv("SEPTIEMBRE_TOKEN", "")

	token, err := resolveWithConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "file-token-abc" {
		t.Errorf("token = %q, want 'file-token-abc'", token)
	}
}

// --- Empty env var falls through to config file ---

func TestResolve_EmptyEnvFallsToConfig(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"token": "fallback-token"})

	t.Setenv("SEPTIEMBRE_TOKEN", "")

	token, err := resolveWithConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "fallback-token" {
		t.Errorf("token = %q, want 'fallback-token'", token)
	}
}

// --- Config file permissions (0600) ---

func TestLoadConfig_CreatesFileWith0600(t *testing.T) {
	// Not parallel: writes to a temp dir and checks file perms.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// SaveToken should create the file with 0600.
	if err := SaveToken(cfgPath, "sapi_abc123"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// On Unix the mode must be exactly 0600. Skip exact check on Windows.
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("file mode = %04o, want 0600", mode)
	}
}

// --- Config path uses os.UserConfigDir ---

func TestDefaultConfigPath_ContainsSeptiembre(t *testing.T) {
	t.Parallel()

	path := DefaultConfigPath()
	if path == "" {
		t.Fatal("DefaultConfigPath returned empty string")
	}
	// Must contain "septiembre" in some form.
	if !contains(path, "septiembre") {
		t.Errorf("config path %q does not contain 'septiembre'", path)
	}
}

// --- API base URL from config ---

func TestAPIBaseURL_DefaultWhenNoEnvOrConfig(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml") // does not exist

	t.Setenv("SEPTIEMBRE_API_URL", "")

	url := apiBaseURLWithConfig(cfgPath)
	if url != DefaultAPIBaseURL {
		t.Errorf("url = %q, want %q", url, DefaultAPIBaseURL)
	}
}

func TestAPIBaseURL_EnvOverrides(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	t.Setenv("SEPTIEMBRE_API_URL", "http://localhost:8080")

	url := apiBaseURLWithConfig(cfgPath)
	if url != "http://localhost:8080" {
		t.Errorf("url = %q, want 'http://localhost:8080'", url)
	}
}

// --- helpers ---

func writeYAML(t *testing.T, path string, data map[string]any) {
	t.Helper()
	// Use JSON marshal because config.yaml uses simple key:value pairs.
	// For a test helper this is fine; viper reads both.
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Convert to simple YAML (key: value\n).
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var sb []byte
	for k, v := range m {
		sb = append(sb, []byte(k+": "+stringify(v)+"\n")...)
	}
	if err := os.WriteFile(path, sb, 0600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}

func stringify(v any) string {
	b, _ := json.Marshal(v)
	// Strip surrounding quotes for simple strings.
	s := string(b)
	if len(s) >= 2 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}
	return s
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsRune(s, sub))
}

func containsRune(s, sub string) bool {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
