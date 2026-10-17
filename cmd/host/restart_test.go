package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestKnownBHisMainPathIsExactAllowlist(t *testing.T) {
	for _, path := range []string{
		"github.com/cznUSER/B-HIS/cmd/api",
		"github.com/cznUSER/B-HIS/cmd/host",
	} {
		if !knownBHisMainPath(path) {
			t.Errorf("expected %q to be accepted", path)
		}
	}
	for _, path := range []string{
		"github.com/cznUSER/B-HIS/cmd/other",
		"github.com/other/B-HIS/cmd/api",
		"github.com/cznUSER/B-HIS/cmd/api/",
		"",
	} {
		if knownBHisMainPath(path) {
			t.Errorf("unexpectedly accepted %q", path)
		}
	}
}

func TestValidateRestartOwnerRejectsUnknownOrCurrentProcess(t *testing.T) {
	base := processIdentity{PID: 451, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}
	for name, identity := range map[string]processIdentity{
		"unknown build info": {PID: 451, ImagePath: base.ImagePath, MainPath: "github.com/other/app"},
		"missing image path": {PID: 451, MainPath: base.MainPath},
		"zero pid":           {ImagePath: base.ImagePath, MainPath: base.MainPath},
		"current process":    {PID: 451, ImagePath: base.ImagePath, MainPath: base.MainPath},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateRestartOwner(identity, 451); err == nil {
				t.Fatal("expected owner to be rejected")
			}
		})
	}
	if err := validateRestartOwner(base, 452); err != nil {
		t.Fatalf("valid project owner rejected: %v", err)
	}
}

func TestHealthMatchesCurrentContract(t *testing.T) {
	valid := healthInfo{
		Status: "ok", Version: "0.2.7", ContractVersion: "0.2",
		ImplementedVersion: "0.2.7", Driver: "sqlserver",
	}
	if !healthMatchesCurrentContract(valid) {
		t.Fatal("valid B-HIS health response rejected")
	}
	mutations := map[string]func(*healthInfo){
		"status":              func(h *healthInfo) { h.Status = "degraded" },
		"version":             func(h *healthInfo) { h.Version = "" },
		"contract version":    func(h *healthInfo) { h.ContractVersion = "" },
		"implemented version": func(h *healthInfo) { h.ImplementedVersion = "" },
		"driver":              func(h *healthInfo) { h.Driver = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			got := valid
			mutate(&got)
			if healthMatchesCurrentContract(got) {
				t.Fatal("mismatched health response accepted")
			}
		})
	}
}

func TestConfiguredRestartEndpoint(t *testing.T) {
	cases := map[string]struct {
		addr string
		want string
	}{
		"ipv4":             {"127.0.0.1:8080", "http://127.0.0.1:8080/health"},
		"unspecified host": {":8080", "http://127.0.0.1:8080/health"},
		"wildcard ipv4":    {"0.0.0.0:8080", "http://127.0.0.1:8080/health"},
		"wildcard ipv6":    {"[::]:8080", "http://[::1]:8080/health"},
		"ipv6":             {"[::1]:9090", "http://[::1]:9090/health"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := configuredRestartEndpoint(tc.addr)
			if err != nil {
				t.Fatal(err)
			}
			if got.healthURL != tc.want {
				t.Fatalf("health URL %q, want %q", got.healthURL, tc.want)
			}
		})
	}
	for _, addr := range []string{"", "localhost", "127.0.0.1:0", "127.0.0.1:65536", "not-an-address"} {
		t.Run("invalid/"+addr, func(t *testing.T) {
			if _, err := configuredRestartEndpoint(addr); err == nil {
				t.Fatal("expected invalid endpoint")
			}
		})
	}
}

func TestProbeRestartHealthRequiresCurrentContractAndNoRedirect(t *testing.T) {
	valid := `{"status":"ok","version":"0.2.7","contractVersion":"0.2","implementedVersion":"0.2.7","driver":"sqlserver"}`
	for name, handler := range map[string]http.HandlerFunc{
		"valid": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(valid)) },
		"wrong status": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(valid))
		},
		"missing fields": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"status":"ok"}`)) },
		"redirect":       func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/other", http.StatusFound) },
		"malformed":      func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{")) },
	} {
		t.Run(name, func(t *testing.T) {
			ts := httptest.NewServer(handler)
			defer ts.Close()
			got, err := probeRestartHealth(ts.URL)
			if name == "valid" {
				if err != nil || !healthMatchesCurrentContract(got) {
					t.Fatalf("valid health rejected: %#v, %v", got, err)
				}
				return
			}
			if err == nil && healthMatchesCurrentContract(got) {
				t.Fatal("invalid health endpoint accepted")
			}
		})
	}
}

func TestRestartExistingListenerOnlyStopsVerifiedSingleOwner(t *testing.T) {
	t.Run("no listener starts normally", func(t *testing.T) {
		ops := &fakeRestartOps{}
		if err := restartExistingListener("127.0.0.1:8080", ops); err != nil {
			t.Fatal(err)
		}
		if ops.probeCalls != 0 || ops.waitReleaseCalls != 0 {
			t.Fatalf("unexpected calls: probe=%d release=%d", ops.probeCalls, ops.waitReleaseCalls)
		}
	})

	t.Run("verified owner stops and waits", func(t *testing.T) {
		initial := &fakeRestartProcess{identity: processIdentity{PID: 8101, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		refreshed := &fakeRestartProcess{identity: initial.identity}
		ops := validFakeRestartOps(initial)
		ops.ownerSnapshots = [][]restartProcess{{initial}, {refreshed}}
		if err := restartExistingListener("127.0.0.1:8080", ops); err != nil {
			t.Fatal(err)
		}
		if initial.stopCalls != 0 || refreshed.stopCalls != 1 || refreshed.waitCalls != 1 || ops.waitReleaseCalls != 1 {
			t.Fatalf("initial stop=%d refreshed stop=%d exit-wait=%d release-wait=%d", initial.stopCalls, refreshed.stopCalls, refreshed.waitCalls, ops.waitReleaseCalls)
		}
		if initial.closeCalls != 1 || refreshed.closeCalls != 1 {
			t.Fatalf("initial/refreshed handles closed %d/%d times", initial.closeCalls, refreshed.closeCalls)
		}
	})

	t.Run("listener disappearing during health check continues startup", func(t *testing.T) {
		initial := &fakeRestartProcess{identity: processIdentity{PID: 8111, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		ops := validFakeRestartOps(initial)
		ops.ownerSnapshots = [][]restartProcess{{initial}, {}}
		if err := restartExistingListener("127.0.0.1:8080", ops); err != nil {
			t.Fatalf("expected normal startup after listener disappeared, got %v", err)
		}
		if initial.stopCalls != 0 || ops.waitReleaseCalls != 0 {
			t.Fatalf("disappeared listener stop=%d release-wait=%d", initial.stopCalls, ops.waitReleaseCalls)
		}
	})

	t.Run("replacement listener remains alive", func(t *testing.T) {
		initial := &fakeRestartProcess{identity: processIdentity{PID: 8112, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		replacement := &fakeRestartProcess{identity: processIdentity{PID: 8113, ImagePath: `C:\temp\host.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/host"}}
		ops := validFakeRestartOps(initial)
		ops.ownerSnapshots = [][]restartProcess{{initial}, {replacement}}
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil || !strings.Contains(err.Error(), "port 8080 is occupied") {
			t.Fatalf("expected occupied-port error for replacement listener, got %v", err)
		}
		if initial.stopCalls != 0 || replacement.stopCalls != 0 {
			t.Fatal("listener replacement must not stop either process")
		}
		if initial.closeCalls != 1 || replacement.closeCalls != 1 {
			t.Fatalf("initial/refreshed handles closed %d/%d times", initial.closeCalls, replacement.closeCalls)
		}
	})

	t.Run("multiple refreshed owners remain alive", func(t *testing.T) {
		initial := &fakeRestartProcess{identity: processIdentity{PID: 8114, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		first := &fakeRestartProcess{identity: processIdentity{PID: 8115, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		second := &fakeRestartProcess{identity: processIdentity{PID: 8116, ImagePath: `C:\temp\host.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/host"}}
		ops := validFakeRestartOps(initial)
		ops.ownerSnapshots = [][]restartProcess{{initial}, {first, second}}
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil || !strings.Contains(err.Error(), "multiple") {
			t.Fatalf("expected multiple-owner error, got %v", err)
		}
		if initial.stopCalls != 0 || first.stopCalls != 0 || second.stopCalls != 0 {
			t.Fatal("multiple refreshed owners must never be stopped")
		}
		if initial.closeCalls != 1 || first.closeCalls != 1 || second.closeCalls != 1 {
			t.Fatalf("initial/refresh handles closed %d/%d/%d times", initial.closeCalls, first.closeCalls, second.closeCalls)
		}
	})

	t.Run("same pid with changed image identity remains alive", func(t *testing.T) {
		initial := &fakeRestartProcess{identity: processIdentity{PID: 8117, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		changed := &fakeRestartProcess{identity: processIdentity{PID: initial.identity.PID, ImagePath: `C:\temp\replacement.exe`, MainPath: initial.identity.MainPath}}
		ops := validFakeRestartOps(initial)
		ops.ownerSnapshots = [][]restartProcess{{initial}, {changed}}
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil {
			t.Fatal("changed owner identity was accepted")
		}
		if initial.stopCalls != 0 || changed.stopCalls != 0 {
			t.Fatal("changed identity must not be stopped")
		}
	})

	t.Run("same pid and image with changed main package remains alive", func(t *testing.T) {
		initial := &fakeRestartProcess{identity: processIdentity{PID: 8119, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		changed := &fakeRestartProcess{identity: processIdentity{PID: initial.identity.PID, ImagePath: initial.identity.ImagePath, MainPath: "github.com/cznUSER/B-HIS/cmd/host"}}
		ops := validFakeRestartOps(initial)
		ops.ownerSnapshots = [][]restartProcess{{initial}, {changed}}
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil {
			t.Fatal("changed main package was accepted")
		}
		if initial.stopCalls != 0 || changed.stopCalls != 0 {
			t.Fatal("changed main package must not be stopped")
		}
	})

	t.Run("refreshed owner lookup failure remains fail closed", func(t *testing.T) {
		initial := &fakeRestartProcess{identity: processIdentity{PID: 8118, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		partial := &fakeRestartProcess{identity: processIdentity{PID: 8120, ImagePath: `C:\temp\host.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/host"}}
		ops := validFakeRestartOps(initial)
		ops.ownerSnapshots = [][]restartProcess{{initial}, {partial}}
		ops.ownerErrors = []error{nil, errors.New("owner table unavailable")}
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil {
			t.Fatal("refreshed owner lookup failure accepted")
		}
		if initial.stopCalls != 0 {
			t.Fatal("owner lookup failure must not stop the initial process")
		}
		if initial.closeCalls != 1 || partial.closeCalls != 1 {
			t.Fatalf("initial/partial handles closed %d/%d times", initial.closeCalls, partial.closeCalls)
		}
	})

	t.Run("unknown owner remains alive", func(t *testing.T) {
		p := &fakeRestartProcess{identity: processIdentity{PID: 8102, ImagePath: `C:\temp\other.exe`, MainPath: "example.com/other"}}
		ops := validFakeRestartOps(p)
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil || !strings.Contains(err.Error(), "port 8080 is occupied") {
			t.Fatalf("expected clear occupied-port error, got %v", err)
		}
		if p.stopCalls != 0 {
			t.Fatalf("unknown process was stopped %d times", p.stopCalls)
		}
	})

	t.Run("multiple owners remain alive", func(t *testing.T) {
		first := &fakeRestartProcess{identity: processIdentity{PID: 8103, ImagePath: "a.exe", MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		second := &fakeRestartProcess{identity: processIdentity{PID: 8104, ImagePath: "b.exe", MainPath: "github.com/cznUSER/B-HIS/cmd/host"}}
		ops := validFakeRestartOps(first, second)
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil {
			t.Fatal("multiple owners accepted")
		}
		if first.stopCalls != 0 || second.stopCalls != 0 {
			t.Fatal("multiple owners must never be stopped")
		}
	})

	t.Run("health mismatch leaves project process alive", func(t *testing.T) {
		p := &fakeRestartProcess{identity: processIdentity{PID: 8105, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}}
		ops := validFakeRestartOps(p)
		ops.health.ContractVersion = ""
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil {
			t.Fatal("health mismatch accepted")
		}
		if p.stopCalls != 0 {
			t.Fatal("health mismatch process was stopped")
		}
	})

	t.Run("wait failure is reported after verified stop", func(t *testing.T) {
		p := &fakeRestartProcess{identity: processIdentity{PID: 8106, ImagePath: `C:\temp\api.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/api"}, waitErr: errors.New("still running")}
		ops := validFakeRestartOps(p)
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil || !strings.Contains(err.Error(), "port 8080 is occupied") {
			t.Fatalf("expected occupied-port timeout error, got %v", err)
		}
		if p.stopCalls != 1 || ops.waitReleaseCalls != 0 {
			t.Fatalf("stop=%d release-wait=%d", p.stopCalls, ops.waitReleaseCalls)
		}
	})

	t.Run("port release timeout is reported", func(t *testing.T) {
		p := &fakeRestartProcess{identity: processIdentity{PID: 8107, ImagePath: `C:\temp\host.exe`, MainPath: "github.com/cznUSER/B-HIS/cmd/host"}}
		ops := validFakeRestartOps(p)
		ops.releaseErr = errors.New("still listening")
		if err := restartExistingListener("127.0.0.1:8080", ops); err == nil || !strings.Contains(err.Error(), "port 8080 is occupied") {
			t.Fatalf("expected occupied-port timeout error, got %v", err)
		}
		if p.stopCalls != 1 || ops.waitReleaseCalls != 1 {
			t.Fatalf("stop=%d release-wait=%d", p.stopCalls, ops.waitReleaseCalls)
		}
	})
}

type fakeRestartProcess struct {
	identity   processIdentity
	stopErr    error
	waitErr    error
	stopCalls  int
	waitCalls  int
	closeCalls int
}

func (p *fakeRestartProcess) Identity() processIdentity    { return p.identity }
func (p *fakeRestartProcess) Stop() error                  { p.stopCalls++; return p.stopErr }
func (p *fakeRestartProcess) WaitExit(time.Duration) error { p.waitCalls++; return p.waitErr }
func (p *fakeRestartProcess) Close()                       { p.closeCalls++ }

type fakeRestartOps struct {
	owners           []restartProcess
	ownerSnapshots   [][]restartProcess
	ownerErrors      []error
	listenerCalls    int
	ownersErr        error
	health           healthInfo
	healthErr        error
	releaseErr       error
	probeCalls       int
	waitReleaseCalls int
}

func (o *fakeRestartOps) ListenerOwners(uint16) ([]restartProcess, error) {
	call := o.listenerCalls
	o.listenerCalls++
	if call < len(o.ownerSnapshots) {
		var err error
		if call < len(o.ownerErrors) {
			err = o.ownerErrors[call]
		}
		return o.ownerSnapshots[call], err
	}
	return o.owners, o.ownersErr
}
func (o *fakeRestartOps) ProbeHealth(string) (healthInfo, error) {
	o.probeCalls++
	return o.health, o.healthErr
}
func (o *fakeRestartOps) WaitPortReleased(uint16, time.Duration) error {
	o.waitReleaseCalls++
	return o.releaseErr
}

func validFakeRestartOps(owners ...restartProcess) *fakeRestartOps {
	return &fakeRestartOps{
		owners: owners,
		health: healthInfo{
			Status: "ok", Version: "0.2.7", ContractVersion: "0.2",
			ImplementedVersion: "0.2.7", Driver: "sqlserver",
		},
	}
}

func TestUnavailableRestartOpsFailClosed(t *testing.T) {
	err := restartExistingListener("127.0.0.1:31987", unavailableRestartOps{})
	if err == nil || !strings.Contains(err.Error(), "port 31987 is occupied") {
		t.Fatalf("expected explicit unavailable/occupied-port error, got %v", err)
	}
}

func TestRestartHealthJSONAllowsOlderBHisVersion(t *testing.T) {
	body, err := json.Marshal(healthInfo{
		Status: "ok", Version: "0.1.0", ContractVersion: "0.1",
		ImplementedVersion: "0.1.0", Driver: "sqlserver",
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	defer ts.Close()
	got, err := probeRestartHealth(ts.URL)
	if err != nil || !healthMatchesCurrentContract(got) {
		t.Fatalf("older B-HIS health response rejected: %#v, %v", got, err)
	}
}
