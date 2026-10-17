package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	hostMainPackage       = "github.com/cznUSER/B-HIS/cmd/host"
	apiMainPackage        = "github.com/cznUSER/B-HIS/cmd/api"
	restartHealthTimeout  = 3 * time.Second
	restartProcessTimeout = 5 * time.Second
	restartPortTimeout    = 5 * time.Second
)

type processIdentity struct {
	PID       uint32
	ImagePath string
	MainPath  string
}

type healthInfo struct {
	Status             string `json:"status"`
	Version            string `json:"version"`
	ContractVersion    string `json:"contractVersion"`
	ImplementedVersion string `json:"implementedVersion"`
	Driver             string `json:"driver"`
}

type restartEndpoint struct {
	port      uint16
	healthURL string
}

type restartProcess interface {
	Identity() processIdentity
	Stop() error
	WaitExit(time.Duration) error
	Close()
}

type restartOps interface {
	ListenerOwners(uint16) ([]restartProcess, error)
	ProbeHealth(string) (healthInfo, error)
	WaitPortReleased(uint16, time.Duration) error
}

type unavailableRestartOps struct{}

func (unavailableRestartOps) ListenerOwners(uint16) ([]restartProcess, error) {
	return nil, fmt.Errorf("listener owner detection is unavailable on this platform")
}

func (unavailableRestartOps) ProbeHealth(string) (healthInfo, error) {
	return healthInfo{}, fmt.Errorf("listener owner detection is unavailable on this platform")
}

func (unavailableRestartOps) WaitPortReleased(uint16, time.Duration) error {
	return fmt.Errorf("listener owner detection is unavailable on this platform")
}

func knownBHisMainPath(path string) bool {
	return path == hostMainPackage || path == apiMainPackage
}

func currentHostPID() uint32 { return uint32(os.Getpid()) }

func validateRestartOwner(identity processIdentity, currentPID uint32) error {
	if identity.PID == 0 || identity.PID == currentPID {
		return fmt.Errorf("invalid or current process id")
	}
	if strings.TrimSpace(identity.ImagePath) == "" {
		return fmt.Errorf("process image path is unavailable")
	}
	if !knownBHisMainPath(identity.MainPath) {
		return fmt.Errorf("Go main package %q is not a B-HIS host or API", identity.MainPath)
	}
	return nil
}

func healthMatchesCurrentContract(info healthInfo) bool {
	return info.Status == "ok" &&
		strings.TrimSpace(info.Version) != "" &&
		strings.TrimSpace(info.ContractVersion) != "" &&
		strings.TrimSpace(info.ImplementedVersion) != "" &&
		strings.TrimSpace(info.Driver) != ""
}

func configuredRestartEndpoint(addr string) (restartEndpoint, error) {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return restartEndpoint{}, fmt.Errorf("invalid Host.ApiAddr %q: %w", addr, err)
	}
	portNumber, err := strconv.Atoi(portText)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return restartEndpoint{}, fmt.Errorf("invalid Host.ApiAddr port %q", portText)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	switch strings.Trim(host, "[]") {
	case "0.0.0.0":
		host = "127.0.0.1"
	case "::":
		host = "::1"
	}
	if _, err := url.Parse("http://" + net.JoinHostPort(host, strconv.Itoa(portNumber)) + "/health"); err != nil {
		return restartEndpoint{}, fmt.Errorf("invalid Host.ApiAddr host %q: %w", host, err)
	}
	return restartEndpoint{
		port:      uint16(portNumber),
		healthURL: "http://" + net.JoinHostPort(host, strconv.Itoa(portNumber)) + "/health",
	}, nil
}

func probeRestartHealth(rawURL string) (healthInfo, error) {
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           (&net.Dialer{Timeout: restartHealthTimeout}).DialContext,
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: restartHealthTimeout,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		Timeout:   restartHealthTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return healthInfo{}, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return healthInfo{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil {
		return healthInfo{}, err
	}
	if len(body) > 4096 {
		return healthInfo{}, fmt.Errorf("health response is too large")
	}
	if response.StatusCode != http.StatusOK {
		return healthInfo{}, fmt.Errorf("health endpoint returned HTTP %d", response.StatusCode)
	}
	var info healthInfo
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&info); err != nil {
		return healthInfo{}, fmt.Errorf("invalid health response: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return healthInfo{}, fmt.Errorf("health response has trailing data")
	}
	if !healthMatchesCurrentContract(info) {
		return healthInfo{}, fmt.Errorf("health response does not match the current B-HIS contract")
	}
	return info, nil
}

func restartExistingListener(addr string, ops restartOps) error {
	endpoint, err := configuredRestartEndpoint(addr)
	if err != nil {
		return err
	}
	if ops == nil {
		return fmt.Errorf("port %d is occupied or cannot be checked: restart operations are unavailable", endpoint.port)
	}
	owners, err := ops.ListenerOwners(endpoint.port)
	if err != nil {
		closeRestartProcesses(owners)
		return fmt.Errorf("port %d is occupied or cannot be checked: listener owner lookup failed: %w", endpoint.port, err)
	}
	defer closeRestartProcesses(owners)
	if len(owners) == 0 {
		return nil
	}
	if len(owners) != 1 {
		return fmt.Errorf("port %d is occupied by multiple processes; refusing to stop any process", endpoint.port)
	}
	owner := owners[0]
	if owner == nil {
		return fmt.Errorf("port %d is occupied but its process identity is unavailable; refusing to stop any process", endpoint.port)
	}
	identity := owner.Identity()
	if err := validateRestartOwner(identity, currentHostPID()); err != nil {
		return fmt.Errorf("port %d is occupied by an unverified process; refusing to stop it: %w", endpoint.port, err)
	}
	health, err := ops.ProbeHealth(endpoint.healthURL)
	if err != nil {
		return fmt.Errorf("port %d is occupied; B-HIS health verification failed, refusing to stop process %d: %w", endpoint.port, identity.PID, err)
	}
	if !healthMatchesCurrentContract(health) {
		return fmt.Errorf("port %d is occupied; B-HIS health verification did not match the current contract, refusing to stop process %d", endpoint.port, identity.PID)
	}
	refreshedOwners, err := ops.ListenerOwners(endpoint.port)
	if err != nil {
		closeRestartProcesses(refreshedOwners)
		return fmt.Errorf("port %d is occupied or cannot be checked: listener owner recheck failed: %w", endpoint.port, err)
	}
	defer closeRestartProcesses(refreshedOwners)
	if len(refreshedOwners) == 0 {
		return nil
	}
	if len(refreshedOwners) != 1 {
		return fmt.Errorf("port %d is occupied by multiple processes after health verification; refusing to stop any process", endpoint.port)
	}
	currentOwner := refreshedOwners[0]
	if currentOwner == nil {
		return fmt.Errorf("port %d is occupied but its current process identity is unavailable; refusing to stop any process", endpoint.port)
	}
	currentIdentity := currentOwner.Identity()
	if currentIdentity != identity {
		return fmt.Errorf("port %d is occupied by a different or changed process after health verification; refusing to stop it", endpoint.port)
	}
	if err := validateRestartOwner(currentIdentity, currentHostPID()); err != nil {
		return fmt.Errorf("port %d is occupied by an unverified process after health verification; refusing to stop it: %w", endpoint.port, err)
	}
	if err := currentOwner.Stop(); err != nil {
		return fmt.Errorf("port %d is occupied; could not stop verified B-HIS process %d: %w", endpoint.port, identity.PID, err)
	}
	if err := currentOwner.WaitExit(restartProcessTimeout); err != nil {
		return fmt.Errorf("port %d is occupied; verified B-HIS process %d did not exit: %w", endpoint.port, identity.PID, err)
	}
	if err := ops.WaitPortReleased(endpoint.port, restartPortTimeout); err != nil {
		return fmt.Errorf("port %d is occupied; listener did not release the port: %w", endpoint.port, err)
	}
	return nil
}

type listenerSnapshot struct {
	endpoint restartEndpoint
	owners   []restartProcess
}

func restartHostListeners(apiAddr, webAddr string, ops restartOps) error {
	endpoints := make([]restartEndpoint, 0, 2)
	for _, addr := range []string{apiAddr, webAddr} {
		endpoint, err := configuredRestartEndpoint(addr)
		if err != nil {
			return err
		}
		for _, existing := range endpoints {
			if existing.port == endpoint.port {
				return fmt.Errorf("API and web listener cannot share port %d", endpoint.port)
			}
		}
		endpoints = append(endpoints, endpoint)
	}
	if ops == nil {
		return fmt.Errorf("ports %d and %d are occupied or cannot be checked: restart operations are unavailable", endpoints[0].port, endpoints[1].port)
	}

	initial, err := inspectListeners(endpoints, ops)
	if err != nil {
		return err
	}
	defer closeSnapshots(initial)

	var identity processIdentity
	occupied := make([]int, 0, len(endpoints))
	for i, snapshot := range initial {
		if len(snapshot.owners) == 0 {
			continue
		}
		if len(snapshot.owners) != 1 || snapshot.owners[0] == nil {
			return fmt.Errorf("port %d has multiple or unknown listener owners; refusing to stop any process", snapshot.endpoint.port)
		}
		current := snapshot.owners[0].Identity()
		if err := validateRestartOwner(current, currentHostPID()); err != nil {
			return fmt.Errorf("port %d is occupied by an unverified process; refusing to stop it: %w", snapshot.endpoint.port, err)
		}
		if current.MainPath != hostMainPackage && current.MainPath != apiMainPackage {
			return fmt.Errorf("port %d is occupied by an unverified process; refusing to stop it", snapshot.endpoint.port)
		}
		if len(occupied) > 0 && current != identity {
			return fmt.Errorf("ports %d and %d are occupied by different processes; refusing to stop either", endpoints[occupied[0]].port, snapshot.endpoint.port)
		}
		identity = current
		occupied = append(occupied, i)
	}
	if len(occupied) == 0 {
		return nil
	}
	if identity.MainPath == apiMainPackage && (len(occupied) != 1 || occupied[0] != 0) {
		return fmt.Errorf("standalone B-HIS API may be restarted only when it exclusively owns the API port; refusing to stop it")
	}

	for _, i := range occupied {
		health, err := ops.ProbeHealth(initial[i].endpoint.healthURL)
		if err != nil {
			return fmt.Errorf("port %d is occupied; B-HIS health verification failed, refusing to stop process %d: %w", initial[i].endpoint.port, identity.PID, err)
		}
		if !healthMatchesCurrentContract(health) {
			return fmt.Errorf("port %d is occupied; B-HIS health verification did not match the current contract, refusing to stop process %d", initial[i].endpoint.port, identity.PID)
		}
	}

	current, err := inspectListeners(endpoints, ops)
	if err != nil {
		return err
	}
	defer closeSnapshots(current)
	for i := range initial {
		if len(current[i].owners) != len(initial[i].owners) {
			return fmt.Errorf("listener owners changed on port %d during verification; refusing to stop any process", endpoints[i].port)
		}
		if len(current[i].owners) == 1 && (current[i].owners[0] == nil || current[i].owners[0].Identity() != initial[i].owners[0].Identity()) {
			return fmt.Errorf("listener owner changed on port %d during verification; refusing to stop any process", endpoints[i].port)
		}
	}

	var process restartProcess
	for _, snapshot := range current {
		if len(snapshot.owners) == 1 {
			process = snapshot.owners[0]
			break
		}
	}
	if process == nil {
		return nil
	}
	if err := process.Stop(); err != nil {
		return fmt.Errorf("ports %d and %d are occupied; could not stop verified B-HIS process %d: %w", endpoints[0].port, endpoints[1].port, identity.PID, err)
	}
	if err := process.WaitExit(restartProcessTimeout); err != nil {
		return fmt.Errorf("verified B-HIS process %d did not exit: %w", identity.PID, err)
	}
	releaseEndpoints := endpoints
	if identity.MainPath == apiMainPackage {
		releaseEndpoints = endpoints[:1]
	}
	for _, endpoint := range releaseEndpoints {
		if err := ops.WaitPortReleased(endpoint.port, restartPortTimeout); err != nil {
			return fmt.Errorf("listener did not release port %d: %w", endpoint.port, err)
		}
	}
	return nil
}

func inspectListeners(endpoints []restartEndpoint, ops restartOps) ([]listenerSnapshot, error) {
	snapshots := make([]listenerSnapshot, 0, len(endpoints))
	for _, endpoint := range endpoints {
		owners, err := ops.ListenerOwners(endpoint.port)
		snapshots = append(snapshots, listenerSnapshot{endpoint: endpoint, owners: owners})
		if err != nil {
			closeSnapshots(snapshots)
			return nil, fmt.Errorf("port %d is occupied or cannot be checked: listener owner lookup failed: %w", endpoint.port, err)
		}
	}
	return snapshots, nil
}

func closeSnapshots(snapshots []listenerSnapshot) {
	for _, snapshot := range snapshots {
		closeRestartProcesses(snapshot.owners)
	}
}

func closeRestartProcesses(processes []restartProcess) {
	for _, process := range processes {
		if process != nil {
			process.Close()
		}
	}
}
