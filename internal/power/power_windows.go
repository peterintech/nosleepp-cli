//go:build windows

package power

import (
	"fmt"
	"syscall"
)

const (
	esSystemRequired = 0x00000001
	esContinuous     = 0x80000000
)

var (
	powerKernel32               = syscall.NewLazyDLL("kernel32.dll")
	procSetThreadExecutionState = powerKernel32.NewProc("SetThreadExecutionState")
)

type windowsManager struct{}

func NewManager() Manager {
	return windowsManager{}
}

func (windowsManager) Acquire() error {
	return setThreadExecutionState(esContinuous | esSystemRequired)
}

func (windowsManager) Release() error {
	return setThreadExecutionState(esContinuous)
}

func setThreadExecutionState(state uintptr) error {
	ret, _, err := procSetThreadExecutionState.Call(state)
	if ret == 0 {
		if err != syscall.Errno(0) {
			return fmt.Errorf("SetThreadExecutionState failed: %w", err)
		}
		return fmt.Errorf("SetThreadExecutionState failed")
	}
	return nil
}
