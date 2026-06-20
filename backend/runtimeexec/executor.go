package runtimeexec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ErrExecutionDisabled is returned by Apply when execution is not enabled. It is
// a sentinel so handlers can answer 412 Precondition Failed for a disabled
// toggle while still mapping a genuine runtime failure to 500.
var ErrExecutionDisabled = errors.New("VPN execution is disabled; set VPN_EXECUTION_ENABLED=true to write config files and run provisioning commands")

type ApplyPlan struct {
	Files    map[string]string `json:"files"`
	Commands []string          `json:"commands"`
}

type Options struct {
	RootDir string
	// ExecutionEnabled is the single safety toggle (VPN_EXECUTION_ENABLED). When
	// false, Apply writes nothing and runs nothing, returning ErrExecutionDisabled.
	ExecutionEnabled   bool
	Runner             CommandRunner
	Timeout            time.Duration
	AllowAbsolutePaths bool
}

type CommandRunner func(context.Context, string) CommandResult

type CommandResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

type ApplyResult struct {
	Status   string          `json:"status"`
	RootDir  string          `json:"root_dir"`
	Files    []string        `json:"files"`
	Commands []CommandResult `json:"commands"`
}

func Apply(ctx context.Context, opts Options, plan ApplyPlan) (ApplyResult, error) {
	if opts.RootDir == "" {
		return ApplyResult{}, fmt.Errorf("runtime root dir is required")
	}
	if !opts.ExecutionEnabled {
		return ApplyResult{}, ErrExecutionDisabled
	}
	root, err := filepath.Abs(opts.RootDir)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return ApplyResult{}, err
	}
	result := ApplyResult{Status: "applied", RootDir: root, Files: []string{}, Commands: []CommandResult{}}
	for name, content := range plan.Files {
		path, err := safeJoin(root, name, opts.AllowAbsolutePaths)
		if err != nil {
			return ApplyResult{}, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return ApplyResult{}, err
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return ApplyResult{}, err
		}
		result.Files = append(result.Files, path)
	}
	runner := opts.Runner
	if runner == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		runner = shellRunner(timeout)
	}
	for _, command := range plan.Commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		cmdResult := runner(ctx, command)
		result.Commands = append(result.Commands, cmdResult)
		if cmdResult.ExitCode != 0 {
			return result, fmt.Errorf("command failed: %s: %s", command, cmdResult.Error)
		}
	}
	return result, nil
}

func safeJoin(root, name string, allowAbsolute bool) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(name))
	if allowAbsolute && filepath.IsAbs(clean) {
		if clean == "/" {
			return "", fmt.Errorf("unsafe file path: %s", name)
		}
		return clean, nil
	}
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." || filepath.IsAbs(clean) {
		return "", fmt.Errorf("unsafe file path: %s", name)
	}
	path := filepath.Join(root, clean)
	if !strings.HasPrefix(path, root+string(os.PathSeparator)) && path != root {
		return "", fmt.Errorf("unsafe file path: %s", name)
	}
	return path, nil
}

func shellRunner(timeout time.Duration) CommandRunner {
	return func(ctx context.Context, command string) CommandResult {
		cmdCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		cmd := exec.CommandContext(cmdCtx, "bash", "-lc", command)
		out, err := cmd.CombinedOutput()
		res := CommandResult{Command: command, Output: string(out)}
		if err != nil {
			res.Error = err.Error()
			if exitErr, ok := err.(*exec.ExitError); ok {
				res.ExitCode = exitErr.ExitCode()
			} else {
				res.ExitCode = 1
			}
		}
		return res
	}
}
