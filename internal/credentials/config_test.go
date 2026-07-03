package credentials

import (
	"path/filepath"
	"strings"
	"testing"
)

// ---- T5: DomainSuffixFromPath precedence RED tests ----

func TestDomainSuffixFromPath_EnvOverride(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"domain_suffix": "staging.example.com"})

	t.Setenv("SEPTIEMBRE_DOMAIN_SUFFIX", "override.example.com")

	suffix := DomainSuffixFromPath(cfgPath)
	if suffix != "override.example.com" {
		t.Errorf("suffix = %q, want 'override.example.com' (env must win)", suffix)
	}
}

func TestDomainSuffixFromPath_ConfigFile(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"domain_suffix": "custom.example.com"})

	t.Setenv("SEPTIEMBRE_DOMAIN_SUFFIX", "")

	suffix := DomainSuffixFromPath(cfgPath)
	if suffix != "custom.example.com" {
		t.Errorf("suffix = %q, want 'custom.example.com' (config must win over default)", suffix)
	}
}

func TestDomainSuffixFromPath_Default(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml") // does not exist

	t.Setenv("SEPTIEMBRE_DOMAIN_SUFFIX", "")

	suffix := DomainSuffixFromPath(cfgPath)
	if suffix != DefaultDomainSuffix {
		t.Errorf("suffix = %q, want default %q", suffix, DefaultDomainSuffix)
	}
}

func TestDomainSuffixFromPath_DefaultIsSepted(t *testing.T) {
	t.Parallel()
	if DefaultDomainSuffix != "septiembre.co" {
		t.Errorf("DefaultDomainSuffix = %q, want 'septiembre.co'", DefaultDomainSuffix)
	}
}

// ---- Unit 6: CognitoDomainFromPath / CognitoClientIDFromPath precedence ----

func TestCognitoDomainFromPath_EnvOverride(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"cognito_domain": "https://staging-auth.example.com"})

	t.Setenv("SEPTIEMBRE_COGNITO_DOMAIN", "https://override-auth.example.com")

	domain := CognitoDomainFromPath(cfgPath)
	if domain != "https://override-auth.example.com" {
		t.Errorf("domain = %q, want 'https://override-auth.example.com' (env must win)", domain)
	}
}

func TestCognitoDomainFromPath_ConfigFile(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"cognito_domain": "https://custom-auth.example.com"})

	t.Setenv("SEPTIEMBRE_COGNITO_DOMAIN", "")

	domain := CognitoDomainFromPath(cfgPath)
	if domain != "https://custom-auth.example.com" {
		t.Errorf("domain = %q, want 'https://custom-auth.example.com' (config must win over default)", domain)
	}
}

func TestCognitoDomainFromPath_Default(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml") // does not exist

	t.Setenv("SEPTIEMBRE_COGNITO_DOMAIN", "")

	domain := CognitoDomainFromPath(cfgPath)
	if domain != DefaultCognitoDomain {
		t.Errorf("domain = %q, want default %q", domain, DefaultCognitoDomain)
	}
}

func TestCognitoDomainFromPath_DefaultHasScheme(t *testing.T) {
	t.Parallel()
	if !strings.Contains(DefaultCognitoDomain, "://") {
		t.Errorf("DefaultCognitoDomain = %q, want a value that includes a scheme", DefaultCognitoDomain)
	}
}

// TestCognitoDomainFromPath_BareDomainGetsScheme proves that a bare domain
// (no scheme) coming from either an env var or a config file key is
// normalized to include https://, so exchangeCode's
// strings.TrimRight(domain, "/")+"/oauth2/token" concatenation can never
// produce a scheme-less, unusable token endpoint URL.
func TestCognitoDomainFromPath_BareDomainGetsScheme(t *testing.T) {
	// Not parallel: uses t.Setenv.
	t.Run("env var bare domain", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		t.Setenv("SEPTIEMBRE_COGNITO_DOMAIN", "bare-env.example.com")

		domain := CognitoDomainFromPath(cfgPath)
		if domain != "https://bare-env.example.com" {
			t.Errorf("domain = %q, want 'https://bare-env.example.com'", domain)
		}
	})

	t.Run("config file bare domain", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		writeYAML(t, cfgPath, map[string]any{"cognito_domain": "bare-config.example.com"})
		t.Setenv("SEPTIEMBRE_COGNITO_DOMAIN", "")

		domain := CognitoDomainFromPath(cfgPath)
		if domain != "https://bare-config.example.com" {
			t.Errorf("domain = %q, want 'https://bare-config.example.com'", domain)
		}
	})
}

func TestCognitoClientIDFromPath_EnvOverride(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"cognito_client_id": "config-client-id"})

	t.Setenv("SEPTIEMBRE_COGNITO_CLIENT_ID", "env-client-id")

	clientID := CognitoClientIDFromPath(cfgPath)
	if clientID != "env-client-id" {
		t.Errorf("clientID = %q, want 'env-client-id' (env must win)", clientID)
	}
}

func TestCognitoClientIDFromPath_ConfigFile(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	writeYAML(t, cfgPath, map[string]any{"cognito_client_id": "config-client-id"})

	t.Setenv("SEPTIEMBRE_COGNITO_CLIENT_ID", "")

	clientID := CognitoClientIDFromPath(cfgPath)
	if clientID != "config-client-id" {
		t.Errorf("clientID = %q, want 'config-client-id' (config must win over default)", clientID)
	}
}

func TestCognitoClientIDFromPath_Default(t *testing.T) {
	// Not parallel: uses t.Setenv.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml") // does not exist

	t.Setenv("SEPTIEMBRE_COGNITO_CLIENT_ID", "")

	clientID := CognitoClientIDFromPath(cfgPath)
	if clientID != DefaultCognitoClientID {
		t.Errorf("clientID = %q, want default %q", clientID, DefaultCognitoClientID)
	}
}
