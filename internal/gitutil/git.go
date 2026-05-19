package gitutil

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Runner struct{}

func (r Runner) Run(args ...string) error {
	return r.RunEnv(nil, args...)
}

func (r Runner) RunEnv(env map[string]string, args ...string) error {
	cmd := exec.Command("git", args...)
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return nil
}

func (r Runner) Output(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (r Runner) OutputLines(args ...string) ([]string, error) {
	out, err := r.Output(args...)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	parts := strings.Split(out, "\n")
	lines := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		lines = append(lines, v)
	}
	return lines, nil
}

func (r Runner) IsRepo() bool {
	_, err := r.Output("rev-parse", "--is-inside-work-tree")
	return err == nil
}

func (r Runner) EnsureRepo() error {
	if r.IsRepo() {
		return nil
	}
	return r.Run("init")
}

func (r Runner) CurrentBranch() (string, error) {
	out, err := r.Output("branch", "--show-current")
	if err != nil {
		return "", err
	}
	return out, nil
}

func (r Runner) HasHead() bool {
	_, err := r.Output("rev-parse", "--verify", "HEAD")
	return err == nil
}

func (r Runner) IsWorkTreeClean() (bool, error) {
	out, err := r.Output("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

func (r Runner) Checkout(branch string) error {
	return r.Run("checkout", branch)
}

func (r Runner) BranchExists(branch string) bool {
	_, err := r.Output("show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

func (r Runner) EnsureBranch(branch, from string) error {
	if r.BranchExists(branch) {
		return nil
	}
	if from == "" {
		return r.Run("branch", branch)
	}
	return r.Run("branch", branch, from)
}

func (r Runner) EnsureRemote(name, url string) error {
	if strings.TrimSpace(url) == "" {
		return errors.New("remote URL cannot be empty")
	}
	_, err := r.Output("remote", "get-url", name)
	if err != nil {
		return r.Run("remote", "add", name, url)
	}
	return r.Run("remote", "set-url", name, url)
}
