package runtimeexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Gates struct {
	RuntimeExecution bool
	FirewallApply    bool
	HostVerification bool
}

type ApplyPlan struct {
	Files    map[string]string `json:"files"`
	Commands []string          `json:"commands"`
}

type Options struct {
	RootDir         string
	Gates           Gates
	ExecutorEnabled bool
	Runner          CommandRunner
	Timeout         time.Duration
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
	if !opts.Gates.RuntimeExecution {
		return ApplyResult{}, fmt.Errorf("VPN_RUNTIME_EXECUTION_ENABLED must be true before runtime commands can run")
	}
	if !opts.Gates.FirewallApply {
		return ApplyResult{}, fmt.Errorf("VPN_FIREWALL_APPLY_ENABLED must be true before firewall rules can be applied")
	}
	if !opts.Gates.HostVerification {
		return ApplyResult{}, fmt.Errorf("VPN_HOST_VERIFICATION_PASSED must be true after host-side verification")
	}
	if !opts.ExecutorEnabled {
		return ApplyResult{}, fmt.Errorf("VPN_COMMAND_EXECUTOR_ENABLED must be true before commands are executed")
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
