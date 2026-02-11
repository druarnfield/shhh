package config

import (
	"os"
	"path/filepath"
)

func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "shhh")
	}
	return filepath.Join(home, ".config", "shhh")
}

func ConfigFilePath() string {
	exe, err := os.Executable()
	if err == nil {
		adjacent := filepath.Join(filepath.Dir(exe), "shhh.toml")
		if _, err := os.Stat(adjacent); err == nil {
			return adjacent
		}
	}
	return filepath.Join(ConfigDir(), "shhh.toml")
}

func StateFilePath() string {
	return filepath.Join(ConfigDir(), "state.json")
}

func LogFilePath() string {
	return filepath.Join(ConfigDir(), "shhh.log")
}

func CABundlePath() string {
	return filepath.Join(ConfigDir(), "ca-bundle.pem")
}
