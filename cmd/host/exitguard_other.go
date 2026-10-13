//go:build !windows

package main

import "os"

// prepareHostExit is a no-op off Windows. The console-close hang and
// rundll32 console child are Windows-only.
func prepareHostExit() {}

func trackHelperProcess(*os.Process) {}
