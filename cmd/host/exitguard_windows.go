//go:build windows

package main

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// killJob is held until process exit on purpose. Closing it early would
// terminate every process assigned to the job.
var (
	killJob       windows.Handle
	jobOnce       sync.Once
	jobErr        error
	consoleOnce   sync.Once
	consoleErr    error
	consoleCtrlCB uintptr
)

var procSetConsoleCtrlHandler = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleCtrlHandler")

// prepareHostExit installs the close-button handler and the kill-on-close job.
// Failures are logged and do not block the demo.
func prepareHostExit() {
	if err := installConsoleCloseHandler(); err != nil {
		fmt.Fprintf(os.Stderr, "host: console close: %s\n", err)
	}
	if err := createKillOnCloseJob(); err != nil {
		fmt.Fprintf(os.Stderr, "host: process job: %s\n", err)
	}
}

// installConsoleCloseHandler handles CTRL_CLOSE_EVENT before Go's runtime
// handler. That handler calls block() and never returns, so on current Windows
// the black console and the listening process can both stay up. ExitProcess
// ends the listener and drops the console immediately.
func installConsoleCloseHandler() error {
	consoleOnce.Do(func() {
		consoleCtrlCB = windows.NewCallback(func(ctrl uintptr) uintptr {
			if consoleCloseShouldExit(uint32(ctrl)) {
				windows.ExitProcess(0)
			}
			return 0
		})
		r, _, err := procSetConsoleCtrlHandler.Call(consoleCtrlCB, 1)
		if r == 0 {
			if err != nil {
				consoleErr = err
				return
			}
			consoleErr = fmt.Errorf("SetConsoleCtrlHandler failed")
		}
	})
	return consoleErr
}

func createKillOnCloseJob() error {
	jobOnce.Do(func() {
		h, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			jobErr = err
			return
		}
		info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: hostJobLimitFlags(),
			},
		}
		if _, err := windows.SetInformationJobObject(
			h,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&info)),
			uint32(unsafe.Sizeof(info)),
		); err != nil {
			windows.CloseHandle(h)
			jobErr = err
			return
		}
		killJob = h
	})
	return jobErr
}

// trackHelperProcess puts a console helper (rundll32 fallback) into the job
// so it dies when the host exits or is killed. The browser opened via
// ShellExecute is not assigned.
func trackHelperProcess(p *os.Process) {
	if p == nil || killJob == 0 {
		return
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(p.Pid))
	if err != nil {
		return
	}
	defer windows.CloseHandle(h)
	_ = windows.AssignProcessToJobObject(killJob, h)
}
