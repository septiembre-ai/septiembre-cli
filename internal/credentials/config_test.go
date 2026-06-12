package credentials

import (
	"path/filepath"
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
