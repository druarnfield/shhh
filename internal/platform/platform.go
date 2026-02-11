package platform

import "errors"

var ErrNotSupported = errors.New("not supported on this platform")

type UserEnv interface {
	Get(key string) (value string, source EnvSource, err error)
	Set(key, value string) error
	Delete(key string) error
	AppendPath(dir string) error
	RemovePath(dir string) error
	ListPath() ([]PathEntry, error)
}

type EnvSource int

const (
	SourceProcess EnvSource = iota
	SourceUser
	SourceSystem
)

func (s EnvSource) String() string {
	switch s {
	case SourceProcess:
		return "process"
	case SourceUser:
		return "user"
	case SourceSystem:
		return "system"
	default:
		return "unknown"
	}
}

type PathEntry struct {
	Dir    string
	Source EnvSource
	Exists bool
}

type ProfileManager interface {
	Path() string
	Read() (string, error)
	ManagedBlock() (string, error)
	SetManagedBlock(content string) error
	AppendToManagedBlock(line string) error
	Diff() (string, error)
	Exists() bool
	EnsureExists() error
}

const (
	ManagedBlockStart = "# >>> shhh managed - do not edit >>>"
	ManagedBlockEnd   = "# <<< shhh managed <<<"
)
