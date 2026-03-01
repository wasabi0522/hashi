package exec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
)

// IsExitError reports whether err wraps an *exec.ExitError.
func IsExitError(err error) bool {
	var exitErr *osexec.ExitError
	return errors.As(err, &exitErr)
}

// IsExitCode reports whether err wraps an *exec.ExitError with the given exit code.
func IsExitCode(err error, code int) bool {
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == code
	}
	return false
}

//go:generate moq -out exec_mock.go . Executor

// Executor abstracts command execution for testing.
type Executor interface {
	LookPath(name string) error
	Output(name string, args ...string) (string, error)
	Run(name string, args ...string) error
	RunInteractive(name string, args ...string) error
	RunShell(command, dir string) error
	RunShellContext(ctx context.Context, command, dir string) error
}

var _ Executor = (*DefaultExecutor)(nil)

// DefaultExecutor implements Executor using os/exec.
type DefaultExecutor struct{}

func NewDefaultExecutor() *DefaultExecutor {
	return &DefaultExecutor{}
}

func (e *DefaultExecutor) LookPath(name string) error {
	_, err := osexec.LookPath(name)
	if err != nil {
		return fmt.Errorf("command not found: %s", name)
	}
	return nil
}

func wrapExecError(err error, stderr string) error {
	errMsg := strings.TrimSpace(stderr)
	if errMsg != "" {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return err
}

func (e *DefaultExecutor) Output(name string, args ...string) (string, error) {
	cmd := osexec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", wrapExecError(err, stderr.String())
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

func (e *DefaultExecutor) Run(name string, args ...string) error {
	cmd := osexec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return wrapExecError(err, stderr.String())
	}
	return nil
}

func (e *DefaultExecutor) RunInteractive(name string, args ...string) error {
	cmd := osexec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (e *DefaultExecutor) RunShell(command, dir string) error {
	return e.RunShellContext(context.Background(), command, dir)
}

func (e *DefaultExecutor) RunShellContext(ctx context.Context, command, dir string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	cmd := osexec.CommandContext(ctx, shell, "-c", command)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
