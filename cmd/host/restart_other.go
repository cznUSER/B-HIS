//go:build !windows

package main

func currentRestartOps() restartOps { return unavailableRestartOps{} }
