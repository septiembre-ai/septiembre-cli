package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// DefaultAPIBaseURL is the production API endpoint.
// Override with SEPTIEMBRE_API_URL for local development or testing.
const DefaultAPIBaseURL = "https://api.septiembre.ai"

// DefaultDomainSuffix is the production domain suffix for composed app URLs.
// Override with SEPTIEMBRE_DOMAIN_SUFFIX env var or the "domain_suffix" config key.
const DefaultDomainSuffix = "septiembre.co"

// DomainSuffixFromPath returns the domain suffix to use for app URL composition.
// Precedence: SEPTIEMBRE_DOMAIN_SUFFIX env var → config "domain_suffix" key → DefaultDomainSuffix.
// Uses the same resolution pattern as DefaultOrgSlugFromPath and apiBaseURLWithConfig.
func DomainSuffixFromPath(configPath string) string {
	if s := os.Getenv("SEPTIEMBRE_DOMAIN_SUFFIX"); s != "" {
		return s
	}
	v, err := loadConfig(configPath)
	if err != nil {
		return DefaultDomainSuffix
	}
	if s := v.GetString("domain_suffix"); s != "" {
		return s
	}
	return DefaultDomainSuffix
}

// DefaultConfigPath returns the OS-appropriate path for the CLI config file.
//
//   - Linux/macOS: $XDG_CONFIG_HOME/septiembre/config.yaml (usually ~/.config/septiembre/config.yaml)
//   - Windows:     %APPDATA%\septiembre\config.yaml
//
// Uses filepath.Join so paths are always OS-native.
func DefaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to home directory when UserConfigDir fails.
		dir, _ = os.UserHomeDir()
	}
	return filepath.Join(dir, "septiembre", "config.yaml")
}

// loadConfig reads the YAML config file at path using viper and returns the
// resulting key-value map. It returns an empty map (not an error) when the
// file does not exist, so callers can still rely on defaults.
func loadConfig(path string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) || os.IsNotExist(err) {
			return v, nil
		}
		return nil, err
	}
	return v, nil
}

// SaveToken writes the given token into the config file at path under the
// "token" key. It creates parent directories as needed and sets the file
// mode to 0600 (owner read/write only).
func SaveToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Load existing config so we don't overwrite other keys.
	_ = v.ReadInConfig()

	v.Set("token", token)

	if err := v.WriteConfigAs(path); err != nil {
		return err
	}

	// Enforce 0600 — viper uses os.Create internally which may allow group reads.
	return os.Chmod(path, 0600)
}

// apiBaseURLWithConfig returns the API base URL to use.
// Precedence: SEPTIEMBRE_API_URL env var → config file api_url → DefaultAPIBaseURL.
func apiBaseURLWithConfig(configPath string) string {
	if u := os.Getenv("SEPTIEMBRE_API_URL"); u != "" {
		return u
	}
	v, err := loadConfig(configPath)
	if err != nil {
		return DefaultAPIBaseURL
	}
	if u := v.GetString("api_url"); u != "" {
		return u
	}
	return DefaultAPIBaseURL
}

// APIBaseURL returns the API base URL using the default config path.
func APIBaseURL() string {
	return apiBaseURLWithConfig(DefaultConfigPath())
}

// APIBaseURLFromPath returns the API base URL using the given config file path.
// Intended for tests and commands that honour the --config flag.
func APIBaseURLFromPath(configPath string) string {
	return apiBaseURLWithConfig(configPath)
}

// ResolveWithConfigPath is the testable version of Resolve that uses an explicit
// config file path instead of DefaultConfigPath(). Use this in command code so
// the --config flag is respected.
func ResolveWithConfigPath(configPath string) (string, error) {
	return resolveWithConfig(configPath)
}

// DefaultOrgSlug returns the default org slug from the environment or config.
// Precedence: SEPTIEMBRE_ORG env var → config file "org" key.
func DefaultOrgSlug() string {
	return DefaultOrgSlugFromPath(DefaultConfigPath())
}

// DefaultOrgSlugFromPath returns the default org slug using the given config path.
// Intended for commands that honour the --config flag.
func DefaultOrgSlugFromPath(configPath string) string {
	if o := os.Getenv("SEPTIEMBRE_ORG"); o != "" {
		return o
	}
	v, err := loadConfig(configPath)
	if err != nil {
		return ""
	}
	return v.GetString("org")
}

// DefaultCognitoDomain is the production Cognito hosted-UI domain used by
// `septiembre login`, including its scheme. It is a var (not a const) so it
// can be overridden via -ldflags at build time, mirroring
// DefaultCognitoClientID. The scheme MUST always be present: exchangeCode
// builds the token endpoint URL as strings.TrimRight(domain, "/") +
// "/oauth2/token" and assumes the domain already includes one.
var DefaultCognitoDomain = "https://auth.septiembre.ai"

// DefaultCognitoClientID is the public Cognito app client ID used by
// `septiembre login`. It has no baked-in default — it MUST be injected at
// build time via -ldflags (see .goreleaser.yaml) since the app client is
// registered per environment.
var DefaultCognitoClientID string

// CognitoDomainFromPath returns the Cognito hosted-UI domain to use.
// Precedence: SEPTIEMBRE_COGNITO_DOMAIN env var → config file
// "cognito_domain" key → DefaultCognitoDomain. Uses the same resolution
// pattern as apiBaseURLWithConfig. The resolved value is always normalized
// to include a scheme (see normalizeCognitoDomain).
func CognitoDomainFromPath(configPath string) string {
	if d := os.Getenv("SEPTIEMBRE_COGNITO_DOMAIN"); d != "" {
		return normalizeCognitoDomain(d)
	}
	v, err := loadConfig(configPath)
	if err != nil {
		return normalizeCognitoDomain(DefaultCognitoDomain)
	}
	if d := v.GetString("cognito_domain"); d != "" {
		return normalizeCognitoDomain(d)
	}
	return normalizeCognitoDomain(DefaultCognitoDomain)
}

// CognitoClientIDFromPath returns the Cognito public app client ID to use.
// Precedence: SEPTIEMBRE_COGNITO_CLIENT_ID env var → config file
// "cognito_client_id" key → DefaultCognitoClientID (build-time default).
// Uses the same resolution pattern as apiBaseURLWithConfig.
func CognitoClientIDFromPath(configPath string) string {
	if c := os.Getenv("SEPTIEMBRE_COGNITO_CLIENT_ID"); c != "" {
		return c
	}
	v, err := loadConfig(configPath)
	if err != nil {
		return DefaultCognitoClientID
	}
	if c := v.GetString("cognito_client_id"); c != "" {
		return c
	}
	return DefaultCognitoClientID
}

// normalizeCognitoDomain ensures domain includes an explicit scheme. A bare
// domain (e.g. hand-edited into the config file without "https://") would
// otherwise let exchangeCode's strings.TrimRight(domain, "/") +
// "/oauth2/token" concatenation produce a scheme-less, unusable URL.
func normalizeCognitoDomain(domain string) string {
	if strings.Contains(domain, "://") {
		return domain
	}
	return "https://" + domain
}
