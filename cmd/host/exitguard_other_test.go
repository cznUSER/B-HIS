//go:build !windows

package main

import "testing"

func TestPrepareHostExitOffWindows(t *testing.T) {
	prepareHostExit()
	trackHelperProcess(nil)
}
