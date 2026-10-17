//go:build !windows

package main

import (
	"strings"
	"testing"
)

func TestCurrentRestartOpsUnavailableFailsClosed(t *testing.T) {
	err := restartExistingListener("127.0.0.1:31987", currentRestartOps())
	if err == nil || !strings.Contains(err.Error(), "port 31987 is occupied") {
		t.Fatalf("expected explicit unavailable/occupied-port error, got %v", err)
	}
}
