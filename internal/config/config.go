package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Remote struct {
	Name   string
	URL    string
	Branch string
}

type Enforcement struct {
	Mode string
}

type Shell struct {
	ProxyGit bool
}

type Config struct {
	PublicBranch  string
	PrivateBranch string
	PublicRemote  Remote
	PrivateRemote Remote
	Enforcement   Enforcement
	Shell         Shell
}

type State struct {
	Context string `json:"context"`
}

func Dir() string        { return ".dualgit" }
func ConfigPath() string { return filepath.Join(Dir(), "config.toml") }
func StatePath() string  { return filepath.Join(Dir(), "state.json") }

func EnsureDir() error { return os.MkdirAll(Dir(), 0o755) }

func Default() Config {
	return Config{
		PublicBranch:  "main",
		PrivateBranch: "private-main",
		PublicRemote:  Remote{Name: "origin", Branch: "main"},
		PrivateRemote: Remote{Name: "private", Branch: "main"},
		Enforcement:   Enforcement{Mode: "strict"},
		Shell:         Shell{ProxyGit: true},
	}
}

func DefaultState() State { return State{Context: "public"} }

func Load() (Config, error) {
	raw, err := os.ReadFile(ConfigPath())
	if err != nil {
		return Config{}, err
	}
	cfg := Default()
	s := bufio.NewScanner(strings.NewReader(string(raw)))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, "\"")
		switch k {
		case "public_branch":
			cfg.PublicBranch = v
		case "private_branch":
			cfg.PrivateBranch = v
		case "public_remote.name":
			cfg.PublicRemote.Name = v
		case "public_remote.url":
			cfg.PublicRemote.URL = v
		case "public_remote.branch":
			cfg.PublicRemote.Branch = v
		case "private_remote.name":
			cfg.PrivateRemote.Name = v
		case "private_remote.url":
			cfg.PrivateRemote.URL = v
		case "private_remote.branch":
			cfg.PrivateRemote.Branch = v
		case "enforcement.mode":
			cfg.Enforcement.Mode = v
		case "shell.proxy_git":
			cfg.Shell.ProxyGit = (v == "true")
		}
	}
	if err := s.Err(); err != nil {
		return Config{}, err
	}
	if cfg.PublicBranch == "" || cfg.PrivateBranch == "" || cfg.PublicRemote.Name == "" || cfg.PrivateRemote.Name == "" {
		return Config{}, errors.New("invalid dualgit config")
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	content := fmt.Sprintf(
		"public_branch = %q\nprivate_branch = %q\npublic_remote.name = %q\npublic_remote.url = %q\npublic_remote.branch = %q\nprivate_remote.name = %q\nprivate_remote.url = %q\nprivate_remote.branch = %q\nenforcement.mode = %q\nshell.proxy_git = %t\n",
		cfg.PublicBranch,
		cfg.PrivateBranch,
		cfg.PublicRemote.Name,
		cfg.PublicRemote.URL,
		cfg.PublicRemote.Branch,
		cfg.PrivateRemote.Name,
		cfg.PrivateRemote.URL,
		cfg.PrivateRemote.Branch,
		cfg.Enforcement.Mode,
		cfg.Shell.ProxyGit,
	)
	return os.WriteFile(ConfigPath(), []byte(content), 0o644)
}

func LoadState() (State, error) {
	raw, err := os.ReadFile(StatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultState(), nil
		}
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return State{}, err
	}
	if s.Context == "" {
		s.Context = "public"
	}
	return s, nil
}

func SaveState(s State) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	if s.Context == "" {
		s.Context = "public"
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath(), raw, 0o644)
}
