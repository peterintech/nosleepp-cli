//go:build windows

package process

import (
	"context"
	"syscall"
	"time"
	"unsafe"

	"github.com/peterintech/nosleepp/internal/agent"
)

const (
	th32csSnapProcess              = 0x00000002
	processQueryLimitedInformation = 0x1000
	maxPath                        = 260
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelp32Snap = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW      = kernel32.NewProc("Process32FirstW")
	procProcess32NextW       = kernel32.NewProc("Process32NextW")
	procOpenProcess          = kernel32.NewProc("OpenProcess")
	procGetProcessTimes      = kernel32.NewProc("GetProcessTimes")
)

type windowsScanner struct{}

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [maxPath]uint16
}

func NewScanner() Scanner {
	return windowsScanner{}
}

func (windowsScanner) Scan(ctx context.Context) ([]agent.Process, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	handle, _, err := procCreateToolhelp32Snap.Call(th32csSnapProcess, 0)
	if handle == uintptr(syscall.InvalidHandle) {
		if err != syscall.Errno(0) {
			return nil, err
		}
		return nil, syscall.EINVAL
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	ret, _, err := procProcess32FirstW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		if err != syscall.Errno(0) {
			return nil, err
		}
		return nil, syscall.EINVAL
	}

	processes := make([]agent.Process, 0)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		processes = append(processes, agent.Process{
			PID:       int(entry.ProcessID),
			ParentPID: int(entry.ParentProcessID),
			Name:      syscall.UTF16ToString(entry.ExeFile[:]),
			CPUTime:   processCPUTime(entry.ProcessID),
		})

		ret, _, err = procProcess32NextW.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			if err != syscall.Errno(0) && err != syscall.ERROR_NO_MORE_FILES {
				return nil, err
			}
			break
		}
	}

	return processes, nil
}

func processCPUTime(pid uint32) time.Duration {
	handle, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if handle == 0 {
		return 0
	}
	defer syscall.CloseHandle(syscall.Handle(handle))

	var creationTime syscall.Filetime
	var exitTime syscall.Filetime
	var kernelTime syscall.Filetime
	var userTime syscall.Filetime
	ret, _, _ := procGetProcessTimes.Call(
		handle,
		uintptr(unsafe.Pointer(&creationTime)),
		uintptr(unsafe.Pointer(&exitTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)
	if ret == 0 {
		return 0
	}

	return filetimeDuration(kernelTime) + filetimeDuration(userTime)
}

func filetimeDuration(filetime syscall.Filetime) time.Duration {
	ticks := (uint64(filetime.HighDateTime) << 32) | uint64(filetime.LowDateTime)
	return time.Duration(ticks * 100)
}
