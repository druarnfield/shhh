package mock

import (
	"fmt"
	"strings"

	"github.com/druarnfield/shhh/internal/platform"
)

// ---------------------------------------------------------------------------
// UserEnv — in-memory implementation of platform.UserEnv
// ---------------------------------------------------------------------------

type UserEnv struct {
	vars map[string]string
	path []string
}

func NewUserEnv() platform.UserEnv {
	return &UserEnv{
		vars: make(map[string]string),
		path: nil,
	}
}

func (u *UserEnv) Get(key string) (string, platform.EnvSource, error) {
	val, ok := u.vars[key]
	if !ok {
		return "", 0, fmt.Errorf("environment variable %q not set", key)
	}
	return val, platform.SourceUser, nil
}

func (u *UserEnv) Set(key, value string) error {
	u.vars[key] = value
	return nil
}

func (u *UserEnv) Delete(key string) error {
	delete(u.vars, key)
	return nil
}

func (u *UserEnv) AppendPath(dir string) error {
	for _, d := range u.path {
		if d == dir {
			return nil // deduplicate
		}
	}
	u.path = append(u.path, dir)
	return nil
}

func (u *UserEnv) RemovePath(dir string) error {
	filtered := u.path[:0]
	for _, d := range u.path {
		if d != dir {
			filtered = append(filtered, d)
		}
	}
	u.path = filtered
	return nil
}

func (u *UserEnv) ListPath() ([]platform.PathEntry, error) {
	entries := make([]platform.PathEntry, len(u.path))
	for i, d := range u.path {
		entries[i] = platform.PathEntry{
			Dir:    d,
			Source: platform.SourceUser,
			Exists: true,
		}
	}
	return entries, nil
}

// ---------------------------------------------------------------------------
// ProfileManager — in-memory implementation of platform.ProfileManager
// ---------------------------------------------------------------------------

type ProfileManager struct {
	path         string
	exists       bool
	content      string // user's own content (outside managed block)
	managedBlock string
}

func NewProfileManager(path string) platform.ProfileManager {
	return &ProfileManager{
		path: path,
	}
}

func (pm *ProfileManager) Path() string {
	return pm.path
}

func (pm *ProfileManager) Exists() bool {
	return pm.exists
}

func (pm *ProfileManager) EnsureExists() error {
	pm.exists = true
	return nil
}

func (pm *ProfileManager) Read() (string, error) {
	var b strings.Builder

	if pm.content != "" {
		b.WriteString(pm.content)
	}

	if pm.managedBlock != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(platform.ManagedBlockStart)
		b.WriteString("\n")
		b.WriteString(pm.managedBlock)
		b.WriteString("\n")
		b.WriteString(platform.ManagedBlockEnd)
		b.WriteString("\n")
	}

	return b.String(), nil
}

func (pm *ProfileManager) ManagedBlock() (string, error) {
	return pm.managedBlock, nil
}

func (pm *ProfileManager) SetManagedBlock(content string) error {
	pm.managedBlock = content
	return nil
}

func (pm *ProfileManager) AppendToManagedBlock(line string) error {
	if pm.managedBlock == "" {
		pm.managedBlock = line
	} else {
		pm.managedBlock = pm.managedBlock + "\n" + line
	}
	return nil
}

func (pm *ProfileManager) Diff() (string, error) {
	return "", nil
}
