package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func autostartPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "autostart", "openperipherals.desktop"), nil
}

func (s *PeripheralService) Autostart() bool {
	path, err := autostartPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func (s *PeripheralService) AutostartElevated() bool { return false }

func (s *PeripheralService) SetAutostart(enabled bool) error {
	path, err := autostartPath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if strings.ContainsAny(exe, "\r\n\t") {
		return errors.New("unsupported character in executable path")
	}
	// Escape first for Exec's quoted argument, then for the desktop entry's
	// value syntax. Percent signs are field codes even inside quotes.
	arg := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "`", "\\`", "$", "\\$", "%", "%%").Replace(exe)
	arg = strings.ReplaceAll(arg, "\\", "\\\\")
	entry := "[Desktop Entry]\nType=Application\nName=OpenPeripherals\nExec=\"" + arg + "\"\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".openperipherals-*.desktop.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.WriteString(entry)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
