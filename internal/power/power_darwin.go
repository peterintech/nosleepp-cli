//go:build darwin

package power

import (
	"errors"
	"os"
	"os/exec"
	"sync"
)

type darwinManager struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

func NewManager() Manager {
	return &darwinManager{}
}

func (m *darwinManager) Acquire() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil {
		return nil
	}

	cmd := exec.Command("caffeinate", "-i")
	if err := cmd.Start(); err != nil {
		return err
	}
	m.cmd = cmd
	return nil
}

func (m *darwinManager) Release() error {
	m.mu.Lock()
	cmd := m.cmd
	m.cmd = nil
	m.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	killErr := cmd.Process.Kill()
	waitErr := cmd.Wait()
	if killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
		return killErr
	}
	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return nil
		}
		return waitErr
	}
	return nil
}
