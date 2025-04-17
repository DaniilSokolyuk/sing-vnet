package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Shell struct {
	*exec.Cmd
}

func (s *Shell) SetDir(path string) *Shell {
	s.Dir = path
	return s
}

func (s *Shell) Attach() *Shell {
	s.Stdin = os.Stdin
	s.Stdout = os.Stderr
	s.Stderr = os.Stderr
	return s
}

func (s *Shell) SetEnv(env []string) *Shell {
	s.Env = append(os.Environ(), env...)
	return s
}

func (s *Shell) Wait() error {
	return s.buildError(s.Cmd.Wait())
}

func (s *Shell) Stop() error {
	if err := s.Cmd.Process.Signal(os.Interrupt); err != nil {
		return s.buildError(err)
	}

	done := make(chan error)
	go func() {
		done <- s.Wait()
	}()

	select {
	case err := <-done:
		return s.buildError(err)
	case <-time.After(3 * time.Second):
		if err := s.Cmd.Process.Kill(); err != nil {
			return s.buildError(fmt.Errorf("failed to kill process after timeout: %w", err))
		}
		return s.buildError(fmt.Errorf("process killed after 3s timeout"))
	}
}

func (s *Shell) Run() error {
	return s.buildError(s.Cmd.Run())
}

func (s *Shell) Read() (string, error) {
	output, err := s.CombinedOutput()
	return string(output), s.buildError(err)
}

func (s *Shell) ReadOutput() (string, error) {
	output, err := s.Output()
	return strings.TrimSpace(string(output)), s.buildError(err)
}

func (s *Shell) buildError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("command %s %s failed: %w", s.Path, s.Args, err)
}
