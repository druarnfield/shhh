//go:build windows

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	userEnvKey   = `Environment`
	systemEnvKey = `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
)

type windowsUserEnv struct{}

func NewUserEnv() UserEnv { return &windowsUserEnv{} }

func (w *windowsUserEnv) Get(key string) (string, EnvSource, error) {
	// Check user environment first.
	if val, err := getRegistryString(registry.CURRENT_USER, userEnvKey, key); err == nil {
		return val, SourceUser, nil
	}

	// Check system environment.
	if val, err := getRegistryString(registry.LOCAL_MACHINE, systemEnvKey, key); err == nil {
		return val, SourceSystem, nil
	}

	// Fall back to process environment.
	if val, ok := os.LookupEnv(key); ok {
		return val, SourceProcess, nil
	}

	return "", 0, fmt.Errorf("environment variable %q not set", key)
}

func (w *windowsUserEnv) Set(key, value string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, userEnvKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening user environment key: %w", err)
	}
	defer k.Close()

	if err := k.SetStringValue(key, value); err != nil {
		return fmt.Errorf("setting %q: %w", key, err)
	}

	broadcastSettingChange()
	return nil
}

func (w *windowsUserEnv) Delete(key string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, userEnvKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening user environment key: %w", err)
	}
	defer k.Close()

	if err := k.DeleteValue(key); err != nil {
		return fmt.Errorf("deleting %q: %w", key, err)
	}

	broadcastSettingChange()
	return nil
}

func (w *windowsUserEnv) AppendPath(dir string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, userEnvKey,
		registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening user environment key: %w", err)
	}
	defer k.Close()

	current, _ := readRawPathValue(k)

	// Check if already present (case-insensitive).
	norm := strings.ToLower(filepath.Clean(dir))
	for _, entry := range splitPath(current) {
		if strings.ToLower(filepath.Clean(entry)) == norm {
			return nil
		}
	}

	// Append.
	var newPath string
	if current == "" {
		newPath = dir
	} else {
		newPath = current + ";" + dir
	}

	if err := k.SetExpandStringValue("Path", newPath); err != nil {
		return fmt.Errorf("writing user PATH: %w", err)
	}

	// Update current process PATH.
	os.Setenv("PATH", dir+";"+os.Getenv("PATH"))

	broadcastSettingChange()
	return nil
}

func (w *windowsUserEnv) RemovePath(dir string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, userEnvKey,
		registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("opening user environment key: %w", err)
	}
	defer k.Close()

	current, _ := readRawPathValue(k)
	if current == "" {
		return nil
	}

	norm := strings.ToLower(filepath.Clean(dir))
	entries := splitPath(current)
	var filtered []string
	for _, e := range entries {
		if strings.ToLower(filepath.Clean(e)) != norm {
			filtered = append(filtered, e)
		}
	}

	newPath := strings.Join(filtered, ";")
	if err := k.SetExpandStringValue("Path", newPath); err != nil {
		return fmt.Errorf("writing user PATH: %w", err)
	}

	broadcastSettingChange()
	return nil
}

func (w *windowsUserEnv) ListPath() ([]PathEntry, error) {
	var entries []PathEntry

	// Read user PATH.
	if uk, err := registry.OpenKey(registry.CURRENT_USER, userEnvKey, registry.QUERY_VALUE); err == nil {
		defer uk.Close()
		if val, err := getRegistryString(registry.CURRENT_USER, userEnvKey, "Path"); err == nil {
			for _, dir := range splitPath(val) {
				_, statErr := os.Stat(dir)
				entries = append(entries, PathEntry{
					Dir:    dir,
					Source: SourceUser,
					Exists: statErr == nil,
				})
			}
		}
	}

	// Read system PATH.
	if val, err := getRegistryString(registry.LOCAL_MACHINE, systemEnvKey, "Path"); err == nil {
		for _, dir := range splitPath(val) {
			_, statErr := os.Stat(dir)
			entries = append(entries, PathEntry{
				Dir:    dir,
				Source: SourceSystem,
				Exists: statErr == nil,
			})
		}
	}

	return entries, nil
}

// getRegistryString reads a string value from the registry, expanding any
// embedded environment variable references for REG_EXPAND_SZ values.
func getRegistryString(root registry.Key, path, name string) (string, error) {
	k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	val, _, err := k.GetStringValue(name)
	return val, err
}

// readRawPathValue reads the user PATH registry value without expanding
// environment variable references, preserving patterns like %USERPROFILE%.
func readRawPathValue(k registry.Key) (string, error) {
	// First call with nil to get the required buffer size.
	n, valtype, err := k.GetValue("Path", nil)
	if err != nil {
		return "", err
	}

	if valtype != registry.SZ && valtype != registry.EXPAND_SZ {
		return "", fmt.Errorf("unexpected PATH value type: %d", valtype)
	}

	buf := make([]byte, n)
	n, _, err = k.GetValue("Path", buf)
	if err != nil {
		return "", err
	}

	if n == 0 {
		return "", nil
	}

	// The registry stores strings as null-terminated UTF-16LE.
	// Convert to Go string without environment variable expansion.
	u16 := unsafe.Slice((*uint16)(unsafe.Pointer(&buf[0])), n/2)
	return syscall.UTF16ToString(u16), nil
}

// splitPath splits a Windows PATH string by semicolons, filtering empty entries.
func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	var parts []string
	for _, p := range strings.Split(path, ";") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// broadcastSettingChange notifies running applications that environment
// variables have changed by sending WM_SETTINGCHANGE to all top-level windows.
func broadcastSettingChange() {
	env, _ := syscall.UTF16PtrFromString("Environment")

	const (
		hwndBroadcast   = 0xFFFF
		wmSettingChange  = 0x001A
		smtoAbortIfHung  = 0x0002
	)

	user32 := syscall.NewLazyDLL("user32.dll")
	sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")
	sendMessageTimeout.Call(
		uintptr(hwndBroadcast),
		uintptr(wmSettingChange),
		0,
		uintptr(unsafe.Pointer(env)),
		uintptr(smtoAbortIfHung),
		5000, // 5-second timeout
		0,
	)
}
