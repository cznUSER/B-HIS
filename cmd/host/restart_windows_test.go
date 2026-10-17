//go:build windows

package main

import (
	"encoding/binary"
	"math/bits"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestListeningPIDsIPv4PortStateAndDedup(t *testing.T) {
	table := make([]byte, 4+3*tcp4OwnerRowSize)
	binary.LittleEndian.PutUint32(table[:4], 3)
	putTCP4OwnerRow(table[4:4+tcp4OwnerRowSize], 8080, mibTCPStateListen, 2001)
	putTCP4OwnerRow(table[4+tcp4OwnerRowSize:4+2*tcp4OwnerRowSize], 8080, mibTCPStateListen, 2001)
	putTCP4OwnerRow(table[4+2*tcp4OwnerRowSize:], 8080, 5, 2002)
	got, err := listeningPIDs(table, afInet, 8080)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != 2001 {
		t.Fatalf("IPv4 listeners = %v, want [2001]", got)
	}
}

func TestListeningPIDsIPv6PortStateAndDedup(t *testing.T) {
	table := make([]byte, 4+3*tcp6OwnerRowSize)
	binary.LittleEndian.PutUint32(table[:4], 3)
	putTCP6OwnerRow(table[4:4+tcp6OwnerRowSize], 9090, mibTCPStateListen, 3001)
	putTCP6OwnerRow(table[4+tcp6OwnerRowSize:4+2*tcp6OwnerRowSize], 9090, mibTCPStateListen, 3002)
	putTCP6OwnerRow(table[4+2*tcp6OwnerRowSize:], 9090, 4, 3003)
	got, err := listeningPIDs(table, afInet6, 9090)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !containsPID(got, 3001) || !containsPID(got, 3002) {
		t.Fatalf("IPv6 listeners = %v, want PIDs 3001 and 3002", got)
	}
}

func TestListeningPIDsRejectsTruncatedTable(t *testing.T) {
	table := make([]byte, 4)
	binary.LittleEndian.PutUint32(table, 1)
	if _, err := listeningPIDs(table, afInet, 8080); err == nil {
		t.Fatal("truncated table accepted")
	}
}

func TestReadExecutableMainPackageRejectsCorruptBuildInfo(t *testing.T) {
	path := t.TempDir() + `\not-go.exe`
	if err := os.WriteFile(path, []byte("not a Go executable"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readExecutableMainPackage(path); err == nil {
		t.Fatal("corrupt build info accepted")
	}
}

func TestQueryListeningPIDsFindsIsolatedListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := net.LookupPort("tcp", portText)
	if err != nil {
		t.Fatal(err)
	}
	pids, err := queryListeningPIDs(uint16(port))
	if err != nil {
		t.Fatal(err)
	}
	if !containsPID(pids, uint32(os.Getpid())) {
		t.Fatalf("listener table PIDs %v do not include test process %d", pids, os.Getpid())
	}
}

func TestUnknownWindowsListenerRemainsAlive(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("still alive"))
	}))
	defer ts.Close()

	err := restartExistingListener(strings.TrimPrefix(ts.URL, "http://"), currentRestartOps())
	if err == nil || !strings.Contains(err.Error(), "port ") || !strings.Contains(err.Error(), "occupied") {
		t.Fatalf("expected occupied-port error, got %v", err)
	}
	response, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("listener stopped after rejected restart: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("external listener returned HTTP %d", response.StatusCode)
	}
}

func putTCP4OwnerRow(row []byte, port uint16, state, pid uint32) {
	binary.LittleEndian.PutUint32(row[0:4], state)
	binary.LittleEndian.PutUint32(row[8:12], uint32(bits.ReverseBytes16(port)))
	binary.LittleEndian.PutUint32(row[20:24], pid)
}

func putTCP6OwnerRow(row []byte, port uint16, state, pid uint32) {
	binary.LittleEndian.PutUint32(row[20:24], uint32(bits.ReverseBytes16(port)))
	binary.LittleEndian.PutUint32(row[48:52], state)
	binary.LittleEndian.PutUint32(row[52:56], pid)
}

func containsPID(pids []uint32, want uint32) bool {
	for _, pid := range pids {
		if pid == want {
			return true
		}
	}
	return false
}
