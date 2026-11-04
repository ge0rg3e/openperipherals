package appdata

import (
	"os"
	"path/filepath"
)

// Dir is the per-user root for OpenPeripherals settings and webview data.
func Dir() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "openperipherals"), nil
}
