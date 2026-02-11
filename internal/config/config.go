package config

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Org        OrgConfig        `toml:"org"`
	Proxy      ProxyConfig      `toml:"proxy"`
	Certs      CertsConfig      `toml:"certs"`
	Git        GitConfig        `toml:"git"`
	GitLab     GitLabConfig     `toml:"gitlab"`
	Registries RegistriesConfig `toml:"registries"`
	Scoop      ScoopConfig      `toml:"scoop"`
	Tools      ToolsConfig      `toml:"tools"`
	Python     PythonConfig     `toml:"python"`
	Golang     GolangConfig     `toml:"golang"`
	Node       NodeConfig       `toml:"node"`
}

type OrgConfig struct {
	Name string `toml:"name"`
}

type ProxyConfig struct {
	HTTP    string `toml:"http"`
	HTTPS   string `toml:"https"`
	NoProxy string `toml:"no_proxy"`
}

type CertsConfig struct {
	Source string   `toml:"source"`
	Extra  []string `toml:"extra"`
}

type GitConfig struct {
	DefaultBranch string   `toml:"default_branch"`
	SSHHosts      []string `toml:"ssh_hosts"`
}

type GitLabConfig struct {
	Host    string `toml:"host"`
	SSHPort int    `toml:"ssh_port"`
}

type RegistriesConfig struct {
	PyPIMirror  string `toml:"pypi_mirror"`
	NPMRegistry string `toml:"npm_registry"`
	GoProxy     string `toml:"go_proxy"`
}

type ScoopConfig struct {
	Buckets []string `toml:"buckets"`
}

type ToolsConfig struct {
	Core     []string `toml:"core"`
	Data     []string `toml:"data"`
	Optional []string `toml:"optional"`
}

type PythonConfig struct {
	Version string `toml:"version"`
}

type GolangConfig struct {
	Version string `toml:"version"`
}

type NodeConfig struct {
	Version string `toml:"version"`
}

func Defaults() *Config {
	return &Config{
		Certs:  CertsConfig{Source: "system"},
		Git:    GitConfig{DefaultBranch: "main"},
		GitLab: GitLabConfig{SSHPort: 22},
		Python: PythonConfig{Version: "3.12"},
		Golang: GolangConfig{Version: "1.23"},
		Node:   NodeConfig{Version: "22"},
	}
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := Defaults()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}
