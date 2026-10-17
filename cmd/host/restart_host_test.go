package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRestartHostListenersStopsOnlyOneVerifiedHostAndReleasesBothPorts(t *testing.T) {
	identity := processIdentity{PID: 9801, ImagePath: `C:\temp\b-his-host.exe`, MainPath: hostMainPackage}
	firstAPI := &fakeRestartProcess{identity: identity}
	firstWeb := &fakeRestartProcess{identity: identity}
	currentAPI := &fakeRestartProcess{identity: identity}
	currentWeb := &fakeRestartProcess{identity: identity}
	ops := validFakeRestartOps(firstAPI)
	ops.ownerSnapshots = [][]restartProcess{{firstAPI}, {firstWeb}, {currentAPI}, {currentWeb}}

	if err := restartHostListeners("127.0.0.1:9999", "127.0.0.1:8888", ops); err != nil {
		t.Fatal(err)
	}
	if ops.listenerCalls != 4 || ops.probeCalls != 2 || ops.waitReleaseCalls != 2 {
		t.Fatalf("owner checks=%d health checks=%d release waits=%d", ops.listenerCalls, ops.probeCalls, ops.waitReleaseCalls)
	}
	if currentAPI.stopCalls+currentWeb.stopCalls != 1 || currentAPI.waitCalls+currentWeb.waitCalls != 1 {
		t.Fatalf("the verified host should stop once: API stop/wait=%d/%d web=%d/%d", currentAPI.stopCalls, currentAPI.waitCalls, currentWeb.stopCalls, currentWeb.waitCalls)
	}
}

func TestRestartHostListenersWithOnlyOneOccupiedPortStillChecksBoth(t *testing.T) {
	identity := processIdentity{PID: 9805, ImagePath: `C:\temp\b-his-host.exe`, MainPath: hostMainPackage}
	initial := &fakeRestartProcess{identity: identity}
	current := &fakeRestartProcess{identity: identity}
	ops := validFakeRestartOps(initial)
	ops.ownerSnapshots = [][]restartProcess{{initial}, nil, {current}, nil}

	if err := restartHostListeners("127.0.0.1:9999", "127.0.0.1:8888", ops); err != nil {
		t.Fatal(err)
	}
	if ops.listenerCalls != 4 || ops.waitReleaseCalls != 2 {
		t.Fatalf("owner checks=%d release waits=%d", ops.listenerCalls, ops.waitReleaseCalls)
	}
	if current.stopCalls != 1 || current.waitCalls != 1 {
		t.Fatalf("verified host stop/wait=%d/%d", current.stopCalls, current.waitCalls)
	}
}

func TestRestartHostListenersCanTakeOverStandaloneAPIOnAPIOnly(t *testing.T) {
	identity := processIdentity{PID: 9808, ImagePath: `C:\temp\b-his-api.exe`, MainPath: apiMainPackage}
	initial := &fakeRestartProcess{identity: identity}
	current := &fakeRestartProcess{identity: identity}
	ops := validFakeRestartOps(initial)
	ops.ownerSnapshots = [][]restartProcess{{initial}, nil, {current}, nil}

	if err := restartHostListeners("127.0.0.1:9999", "127.0.0.1:8888", ops); err != nil {
		t.Fatal(err)
	}
	if ops.listenerCalls != 4 || ops.probeCalls != 1 || ops.waitReleaseCalls != 1 {
		t.Fatalf("owner checks=%d health checks=%d release waits=%d", ops.listenerCalls, ops.probeCalls, ops.waitReleaseCalls)
	}
	if initial.stopCalls != 0 || current.stopCalls != 1 || current.waitCalls != 1 {
		t.Fatalf("initial stop=%d current stop/wait=%d/%d", initial.stopCalls, current.stopCalls, current.waitCalls)
	}
}

func TestRestartHostListenersRejectsStandaloneAPIOutsideExclusiveAPIEndpoint(t *testing.T) {
	identity := processIdentity{PID: 9809, ImagePath: `C:\temp\b-his-api.exe`, MainPath: apiMainPackage}
	tests := []struct {
		name string
		api  restartProcess
		web  restartProcess
	}{
		{
			name: "api owns web only",
			web:  &fakeRestartProcess{identity: identity},
		},
		{
			name: "api owns both ports",
			api:  &fakeRestartProcess{identity: identity},
			web:  &fakeRestartProcess{identity: identity},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := validFakeRestartOps()
			ops.ownerSnapshots = [][]restartProcess{{tt.api}, {tt.web}}
			if tt.api == nil {
				ops.ownerSnapshots[0] = nil
			}
			if tt.web == nil {
				ops.ownerSnapshots[1] = nil
			}
			if err := restartHostListeners("127.0.0.1:9999", "127.0.0.1:8888", ops); err == nil {
				t.Fatal("expected standalone API owner to be rejected")
			}
			if ops.probeCalls != 0 || (tt.api != nil && tt.api.(*fakeRestartProcess).stopCalls != 0) || (tt.web != nil && tt.web.(*fakeRestartProcess).stopCalls != 0) {
				t.Fatal("rejected ownership layout must not be probed or stopped")
			}
		})
	}
}

func TestRestartHostListenersDoesNotStopHostWhenOtherPortIsUnrelated(t *testing.T) {
	host := &fakeRestartProcess{identity: processIdentity{PID: 9806, ImagePath: `C:\temp\b-his-host.exe`, MainPath: hostMainPackage}}
	other := &fakeRestartProcess{identity: processIdentity{PID: 9807, ImagePath: `C:\node.exe`, MainPath: "node"}}
	ops := validFakeRestartOps(host)
	ops.ownerSnapshots = [][]restartProcess{{host}, {other}}

	err := restartHostListeners("127.0.0.1:9999", "127.0.0.1:8888", ops)
	if err == nil || !strings.Contains(err.Error(), "refusing to stop") {
		t.Fatalf("expected refusal, got %v", err)
	}
	if host.stopCalls != 0 || other.stopCalls != 0 {
		t.Fatalf("host/external stop calls=%d/%d", host.stopCalls, other.stopCalls)
	}
}

func TestRestartHostListenersStandaloneAPIFailsClosedOnHealthOrOwnerRace(t *testing.T) {
	initialIdentity := processIdentity{PID: 9810, ImagePath: `C:\temp\b-his-api.exe`, MainPath: apiMainPackage}
	initial := &fakeRestartProcess{identity: initialIdentity}
	replacement := &fakeRestartProcess{identity: processIdentity{PID: 9811, ImagePath: `C:\temp\b-his-api.exe`, MainPath: apiMainPackage}}
	for _, tt := range []struct {
		name         string
		healthErr    error
		ownerRefresh restartProcess
		wantRecheck  bool
	}{
		{name: "health failure", healthErr: errRestartTestHealth},
		{name: "different owner after health", ownerRefresh: replacement, wantRecheck: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ops := validFakeRestartOps(initial)
			ops.healthErr = tt.healthErr
			ops.ownerSnapshots = [][]restartProcess{{initial}, nil}
			if tt.wantRecheck {
				ops.ownerSnapshots = append(ops.ownerSnapshots, []restartProcess{tt.ownerRefresh}, nil)
			}
			err := restartHostListeners("127.0.0.1:9999", "127.0.0.1:8888", ops)
			if err == nil {
				t.Fatal("expected fail-closed error")
			}
			if initial.stopCalls != 0 || replacement.stopCalls != 0 {
				t.Fatal("process changed during verification must not be stopped")
			}
			if tt.wantRecheck && ops.listenerCalls != 4 {
				t.Fatalf("owner recheck count %d, want 4", ops.listenerCalls)
			}
		})
	}
}

var errRestartTestHealth = errors.New("unhealthy endpoint")

func TestRestartHostListenersRefusesUnrelatedOrDifferentOwners(t *testing.T) {
	tests := []struct {
		name    string
		api     restartProcess
		web     restartProcess
		wantErr string
	}{
		{
			name:    "unrelated process",
			api:     &fakeRestartProcess{identity: processIdentity{PID: 9802, ImagePath: `C:\node.exe`, MainPath: "node"}},
			wantErr: "refusing to stop",
		},
		{
			name:    "different processes",
			api:     &fakeRestartProcess{identity: processIdentity{PID: 9803, ImagePath: `C:\temp\host.exe`, MainPath: hostMainPackage}},
			web:     &fakeRestartProcess{identity: processIdentity{PID: 9804, ImagePath: `C:\temp\host.exe`, MainPath: hostMainPackage}},
			wantErr: "different processes",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := validFakeRestartOps()
			ops.ownerSnapshots = [][]restartProcess{{tt.api}, {tt.web}}
			if tt.api == nil {
				ops.ownerSnapshots[0] = nil
			}
			if tt.web == nil {
				ops.ownerSnapshots[1] = nil
			}
			err := restartHostListeners("127.0.0.1:9999", "127.0.0.1:8888", ops)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected %q error, got %v", tt.wantErr, err)
			}
			if tt.api != nil && tt.api.(*fakeRestartProcess).stopCalls != 0 {
				t.Fatal("API owner must not be stopped")
			}
			if tt.web != nil && tt.web.(*fakeRestartProcess).stopCalls != 0 {
				t.Fatal("web owner must not be stopped")
			}
		})
	}
}
