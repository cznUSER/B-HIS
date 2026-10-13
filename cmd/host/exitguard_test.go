package main

import "testing"

func TestConsoleCloseShouldExit(t *testing.T) {
	mustExit := []uint32{ctrlCloseEvent, ctrlLogoffEvent, ctrlShutdownEvent}
	for _, ctrl := range mustExit {
		if !consoleCloseShouldExit(ctrl) {
			t.Fatalf("ctrl %d should end the process", ctrl)
		}
	}
	// Ctrl+C / Ctrl+Break keep the graceful SIGINT shutdown.
	for _, ctrl := range []uint32{ctrlCEvent, ctrlBreakEvent, 3, 4, 99} {
		if consoleCloseShouldExit(ctrl) {
			t.Fatalf("ctrl %d should not force ExitProcess", ctrl)
		}
	}
}

func TestHostJobLimitFlags(t *testing.T) {
	flags := hostJobLimitFlags()
	if flags&jobLimitKillOnJobClose == 0 {
		t.Fatal("job must kill members when the host handle closes")
	}
	if flags&jobLimitSilentBreakawayOK != 0 {
		t.Fatal("silent breakaway would let a console child outlive the host")
	}
}
