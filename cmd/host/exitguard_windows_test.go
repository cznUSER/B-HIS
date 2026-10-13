//go:build windows

package main

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestWin32ExitConstants(t *testing.T) {
	if windows.CTRL_C_EVENT != ctrlCEvent ||
		windows.CTRL_BREAK_EVENT != ctrlBreakEvent ||
		windows.CTRL_CLOSE_EVENT != ctrlCloseEvent ||
		windows.CTRL_LOGOFF_EVENT != ctrlLogoffEvent ||
		windows.CTRL_SHUTDOWN_EVENT != ctrlShutdownEvent {
		t.Fatal("console control constants drifted from golang.org/x/sys/windows")
	}
	if windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE != jobLimitKillOnJobClose ||
		windows.JOB_OBJECT_LIMIT_BREAKAWAY_OK != jobLimitBreakawayOK ||
		windows.JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK != jobLimitSilentBreakawayOK {
		t.Fatal("job limit constants drifted")
	}
	if windows.CREATE_NO_WINDOW != createNoWindow || windows.CREATE_NEW_CONSOLE != createNewConsole {
		t.Fatal("process creation constants drifted")
	}
	if hostJobLimitFlags()&windows.JOB_OBJECT_LIMIT_SILENT_BREAKAWAY_OK != 0 {
		t.Fatal("silent breakaway")
	}
}

func TestCreateKillOnCloseJob(t *testing.T) {
	if err := createKillOnCloseJob(); err != nil {
		t.Fatal(err)
	}
	if killJob == 0 {
		t.Fatal("job handle was not kept open")
	}
	if err := createKillOnCloseJob(); err != nil {
		t.Fatal(err)
	}
}
