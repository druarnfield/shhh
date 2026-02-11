//go:build windows

package platform

import "errors"

type windowsUserEnv struct{}

func NewUserEnv() UserEnv { return &windowsUserEnv{} }

func (w *windowsUserEnv) Get(key string) (string, EnvSource, error) {
	return "", SourceProcess, errors.New("not yet implemented")
}
func (w *windowsUserEnv) Set(key, value string) error { return errors.New("not yet implemented") }
func (w *windowsUserEnv) Delete(key string) error     { return errors.New("not yet implemented") }
func (w *windowsUserEnv) AppendPath(dir string) error { return errors.New("not yet implemented") }
func (w *windowsUserEnv) RemovePath(dir string) error { return errors.New("not yet implemented") }
func (w *windowsUserEnv) ListPath() ([]PathEntry, error) {
	return nil, errors.New("not yet implemented")
}
