package credentials

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// DefaultAPIBaseURL is the production API endpoint.
// Override with SEPTIEMBRE_API_URL for local development or testing.
const DefaultAPIBaseURL = "https://api.septiembre.ai"

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
