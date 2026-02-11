//go:build !windows

package platform

type StubProfileManager struct{}

func NewProfileManager() ProfileManager                              { return &StubProfileManager{} }
func (s *StubProfileManager) Path() string                           { return "" }
func (s *StubProfileManager) Read() (string, error)                  { return "", ErrNotSupported }
func (s *StubProfileManager) ManagedBlock() (string, error)          { return "", ErrNotSupported }
func (s *StubProfileManager) SetManagedBlock(content string) error   { return ErrNotSupported }
func (s *StubProfileManager) AppendToManagedBlock(line string) error { return ErrNotSupported }
func (s *StubProfileManager) Diff() (string, error)                  { return "", ErrNotSupported }
func (s *StubProfileManager) Exists() bool                           { return false }
func (s *StubProfileManager) EnsureExists() error                    { return ErrNotSupported }
