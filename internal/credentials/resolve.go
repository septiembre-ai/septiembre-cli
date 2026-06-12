// Package credentials implements the credential resolution chain for septiembre CLI.
//
// Resolution order (first hit wins):
//  1. SEPTIEMBRE_TOKEN environment variable — primary path; required for CI/agents.
//  2. Config file token field (~/.config/septiembre/config.yaml on Linux/macOS,
//     %APPDATA%\septiembre\config.yaml on Windows).
//
// OS keychain support is intentionally deferred to the human login slice (device flow).
// CI users MUST use SEPTIEMBRE_TOKEN. This is documented in README.md.
package credentials

import (
	"errors"
	"os"
)

// ErrNoCredentials is returned when no token can be found via any resolution path.
var ErrNoCredentials = errors.New("no credentials found: set SEPTIEMBRE_TOKEN or run 'septiembre auth login'")

// Resolve returns the API token using the default resolution chain.
// It is the entry point for all CLI commands that need authentication.
func Resolve() (string, error) {
	return resolveWithConfig(DefaultConfigPath())
}

// resolveWithConfig is the testable core of Resolve. It accepts an explicit
// config file path so tests can use t.TempDir() instead of the real config dir.
func resolveWithConfig(configPath string) (string, error) {
	// Step 1: SEPTIEMBRE_TOKEN environment variable.
	if token := os.Getenv("SEPTIEMBRE_TOKEN"); token != "" {
		return token, nil
	}

	// Step 2: Config file token field.
	v, err := loadConfig(configPath)
	if err != nil {
		return "", err
	}
	if token := v.GetString("token"); token != "" {
		return token, nil
	}

	// No credential found via any path.
	return "", ErrNoCredentials
}
