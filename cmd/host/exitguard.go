// 2026-10-13 宿主退出时不留监听进程或黑控制台。
package main

// Windows control events. CTRL_CLOSE is the console window's close button.
const (
	ctrlCEvent        = 0
	ctrlBreakEvent    = 1
	ctrlCloseEvent    = 2
	ctrlLogoffEvent   = 5
	ctrlShutdownEvent = 6
)

// Job object limit flags. Silent breakaway is intentionally absent: a console
// child that leaves the job would keep the black window after the host exits.
const (
	jobLimitKillOnJobClose    = 0x00002000
	jobLimitBreakawayOK       = 0x00000800
	jobLimitSilentBreakawayOK = 0x00001000
)

// consoleCloseShouldExit reports events where the host must leave immediately.
// Ctrl+C and Ctrl+Break stay on the graceful SIGINT path.
func consoleCloseShouldExit(ctrl uint32) bool {
	switch ctrl {
	case ctrlCloseEvent, ctrlLogoffEvent, ctrlShutdownEvent:
		return true
	default:
		return false
	}
}

// hostJobLimitFlags kills every process still in the job when the parent
// exits or is terminated (the last job handle closes with the process).
func hostJobLimitFlags() uint32 {
	return jobLimitKillOnJobClose | jobLimitBreakawayOK
}
