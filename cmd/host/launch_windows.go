//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procShellExecuteW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")

func openBrowserOS(rawURL string) error {
	plan := planBrowser("windows", rawURL)
	if plan.UseShellExecute {
		if err := shellOpen(rawURL); err == nil {
			return nil
		}
	}
	if len(plan.Fallback) == 0 {
		return fmt.Errorf("no browser launcher")
	}
	cmd := exec.Command(plan.Fallback[0], plan.Fallback[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: plan.CreationFlags,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	trackHelperProcess(cmd.Process)
	return nil
}

// shellOpen uses ShellExecuteW so the browser is not a console child of the host.
// rundll32 is a console-subsystem binary; if it stays attached, the black window
// remains after b-his-host.exe exits.
func shellOpen(rawURL string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(rawURL)
	if err != nil {
		return err
	}
	r, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		0,
		uintptr(windows.SW_SHOWNORMAL),
	)
	if r > 32 {
		return nil
	}
	if callErr != nil && callErr != syscall.Errno(0) {
		return callErr
	}
	return fmt.Errorf("ShellExecute failed (%d)", r)
}
