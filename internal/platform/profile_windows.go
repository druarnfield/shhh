//go:build windows

package platform

import "errors"

type windowsProfileManager struct{}

func NewProfileManager() ProfileManager { return &windowsProfileManager{} }
func (w *windowsProfileManager) Path() string {
	return ""
}
func (w *windowsProfileManager) Read() (string, error) {
	return "", errors.New("not yet implemented")
}
func (w *windowsProfileManager) ManagedBlock() (string, error) {
	return "", errors.New("not yet implemented")
}
func (w *windowsProfileManager) SetManagedBlock(content string) error {
	return errors.New("not yet implemented")
}
func (w *windowsProfileManager) AppendToManagedBlock(line string) error {
	return errors.New("not yet implemented")
}
func (w *windowsProfileManager) Diff() (string, error) {
	return "", errors.New("not yet implemented")
}
func (w *windowsProfileManager) Exists() bool      { return false }
func (w *windowsProfileManager) EnsureExists() error { return errors.New("not yet implemented") }
