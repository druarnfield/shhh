//go:build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

type windowsProfileManager struct {
	path string
}

// NewProfileManager returns a ProfileManager that manages the PowerShell 7+
// CurrentUserAllHosts profile at Documents\PowerShell\profile.ps1.
func NewProfileManager() ProfileManager {
	home, _ := os.UserHomeDir()
	return &windowsProfileManager{
		path: filepath.Join(home, "Documents", "PowerShell", "profile.ps1"),
	}
}

func (w *windowsProfileManager) Path() string { return w.path }

func (w *windowsProfileManager) Exists() bool {
	_, err := os.Stat(w.path)
	return err == nil
}

func (w *windowsProfileManager) EnsureExists() error {
	if w.Exists() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}
	f, err := os.Create(w.path)
	if err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}
	return f.Close()
}

func (w *windowsProfileManager) Read() (string, error) {
	data, err := os.ReadFile(w.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading profile: %w", err)
	}
	return string(data), nil
}

func (w *windowsProfileManager) ManagedBlock() (string, error) {
	content, err := w.Read()
	if err != nil {
		return "", err
	}
	return extractManagedBlock(content), nil
}

func (w *windowsProfileManager) SetManagedBlock(content string) error {
	if err := w.EnsureExists(); err != nil {
		return err
	}

	current, err := w.Read()
	if err != nil {
		return err
	}

	updated := replaceManagedBlock(current, content)
	if err := os.WriteFile(w.path, []byte(updated), 0644); err != nil {
		return fmt.Errorf("writing profile: %w", err)
	}
	return nil
}

func (w *windowsProfileManager) AppendToManagedBlock(line string) error {
	block, err := w.ManagedBlock()
	if err != nil {
		return err
	}

	if block == "" {
		return w.SetManagedBlock(line)
	}
	return w.SetManagedBlock(block + "\n" + line)
}

func (w *windowsProfileManager) Diff() (string, error) {
	return "", nil
}
