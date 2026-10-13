//go:build !windows

package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

func openBrowserOS(rawURL string) error {
	args := planBrowser(runtime.GOOS, rawURL).Args
	if len(args) == 0 {
		return fmt.Errorf("no browser launcher")
	}
	return exec.Command(args[0], args[1:]...).Start()
}
