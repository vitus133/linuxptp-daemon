package phcsync

import (
	"bufio"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// gracePeriod is how long to wait for a subprocess to exit on SIGTERM before
// escalating to SIGKILL.
const gracePeriod = 2 * time.Second

// Runner abstracts subprocess execution so ptp4l and phc_ctl invocations can
// be stubbed in unit tests.
type Runner interface {
	// RunAndCapture runs name to completion and returns combined stdout+stderr.
	RunAndCapture(name string, args ...string) (string, error)
	// StartRunning starts a long-lived process and returns a handle for
	// streaming its output and terminating it.
	StartRunning(name string, args ...string) (*RunningProcess, error)
}

// RunningProcess wraps a started subprocess and its merged output stream.
type RunningProcess struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

// StartRunning starts name via the OS with stdout and stderr merged onto a
// single readable stream (so ptp4l log capture matches the daemon behaviour).
func StartRunning(name string, args ...string) (*RunningProcess, error) {
	cmd := exec.Command(name, args...)
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return nil, err
	}
	pw.Close()
	return &RunningProcess{cmd: cmd, stdout: pr}, nil
}

// Scanner returns a line scanner over the process's merged output.
func (p *RunningProcess) Scanner() *bufio.Scanner {
	return bufio.NewScanner(p.stdout)
}

// Terminate sends SIGTERM to the running process.
func (p *RunningProcess) Terminate() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Signal(syscall.SIGTERM)
}

// Stop terminates the process and waits for it to exit, escalating to SIGKILL
// if it does not stop within gracePeriod. It returns the Wait error.
func (p *RunningProcess) Stop() error {
	if err := p.Terminate(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	select {
	case err := <-done:
		p.stdout.Close()
		return err
	case <-time.After(gracePeriod):
		_ = p.cmd.Process.Kill()
		err := <-done
		p.stdout.Close()
		return err
	}
}

// OSRunner executes processes through the real OS environment.
type OSRunner struct{}

// RunAndCapture implements Runner.
func (OSRunner) RunAndCapture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// StartRunning implements Runner.
func (OSRunner) StartRunning(name string, args ...string) (*RunningProcess, error) {
	return StartRunning(name, args...)
}

// LoggingRunner wraps a Runner and streams every command and each output line
// to Log as it is produced, while still returning the full captured output.
type LoggingRunner struct {
	Inner Runner
	Log   func(format string, args ...any)
}

// RunAndCapture implements Runner.
func (l LoggingRunner) RunAndCapture(name string, args ...string) (string, error) {
	if l.Log != nil {
		l.Log("command: %s %s", name, strings.Join(args, " "))
	}
	cmd := exec.Command(name, args...)
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", err
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return "", err
	}
	pw.Close()

	var out strings.Builder
	sc := bufio.NewScanner(pr)
	for sc.Scan() {
		line := sc.Text()
		if l.Log != nil {
			l.Log("%s: %s", name, line)
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	pr.Close()
	err = cmd.Wait()
	if err != nil {
		return out.String(), err
	}
	return out.String(), nil
}

// StartRunning implements Runner.
func (l LoggingRunner) StartRunning(name string, args ...string) (*RunningProcess, error) {
	if l.Log != nil {
		l.Log("command: %s %s", name, strings.Join(args, " "))
	}
	return l.Inner.StartRunning(name, args...)
}
