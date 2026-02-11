//go:build !windows

package platform

type StubUserEnv struct{}

func NewUserEnv() UserEnv { return &StubUserEnv{} }
func (s *StubUserEnv) Get(key string) (string, EnvSource, error) {
	return "", SourceProcess, ErrNotSupported
}
func (s *StubUserEnv) Set(key, value string) error { return ErrNotSupported }
func (s *StubUserEnv) Delete(key string) error     { return ErrNotSupported }
func (s *StubUserEnv) AppendPath(dir string) error { return ErrNotSupported }
func (s *StubUserEnv) RemovePath(dir string) error { return ErrNotSupported }
func (s *StubUserEnv) ListPath() ([]PathEntry, error) {
	return nil, ErrNotSupported
}
