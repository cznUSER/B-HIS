//go:build windows

package main

import (
	"debug/buildinfo"
	"encoding/binary"
	"fmt"
	"math/bits"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	afInet                  = 2
	afInet6                 = 23
	tcpTableOwnerPIDAll     = 5
	mibTCPStateListen       = 2
	processStillActive      = 259
	getExtendedTCPTableSize = 122
	tcp4OwnerRowSize        = 24
	tcp6OwnerRowSize        = 56
)

var procGetExtendedTCPTable = windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetExtendedTcpTable")

type windowsRestartOps struct{}

type windowsRestartProcess struct {
	identity processIdentity
	handle   windows.Handle
	closeOne sync.Once
}

func currentRestartOps() restartOps { return windowsRestartOps{} }

func (windowsRestartOps) ListenerOwners(port uint16) ([]restartProcess, error) {
	pids, err := queryListeningPIDs(port)
	if err != nil {
		return nil, err
	}
	owners := make([]restartProcess, 0, len(pids))
	for _, pid := range pids {
		owner, err := openWindowsRestartProcess(pid)
		if err != nil {
			closeRestartProcesses(owners)
			return nil, fmt.Errorf("cannot inspect listener PID %d: %w", pid, err)
		}
		owners = append(owners, owner)
	}
	return owners, nil
}

func (windowsRestartOps) ProbeHealth(rawURL string) (healthInfo, error) {
	return probeRestartHealth(rawURL)
}

func (windowsRestartOps) WaitPortReleased(port uint16, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		pids, err := queryListeningPIDs(port)
		if err != nil {
			return err
		}
		if len(pids) == 0 {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("listener PIDs remain: %v", pids)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func openWindowsRestartProcess(pid uint32) (*windowsRestartProcess, error) {
	access := uint32(windows.PROCESS_QUERY_LIMITED_INFORMATION | windows.SYNCHRONIZE | windows.PROCESS_TERMINATE)
	handle, err := windows.OpenProcess(access, false, pid)
	if err != nil {
		return nil, err
	}
	closeOnError := true
	defer func() {
		if closeOnError {
			_ = windows.CloseHandle(handle)
		}
	}()

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return nil, err
	}
	if exitCode != processStillActive {
		return nil, fmt.Errorf("process has already exited")
	}
	imageBuffer := make([]uint16, 32768)
	imageLength := uint32(len(imageBuffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &imageBuffer[0], &imageLength); err != nil {
		return nil, err
	}
	imagePath := windows.UTF16ToString(imageBuffer[:imageLength])
	mainPath, err := readExecutableMainPackage(imagePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read Go build info from listener image: %w", err)
	}
	closeOnError = false
	return &windowsRestartProcess{
		identity: processIdentity{PID: pid, ImagePath: imagePath, MainPath: mainPath},
		handle:   handle,
	}, nil
}

func readExecutableMainPackage(imagePath string) (string, error) {
	info, err := buildinfo.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	if info.Path == "" {
		return "", fmt.Errorf("Go main package path is empty")
	}
	return info.Path, nil
}

func (p *windowsRestartProcess) Identity() processIdentity { return p.identity }

func (p *windowsRestartProcess) Stop() error {
	var exitCode uint32
	if err := windows.GetExitCodeProcess(p.handle, &exitCode); err != nil {
		return err
	}
	if exitCode != processStillActive {
		return nil
	}
	return windows.TerminateProcess(p.handle, 1)
}

func (p *windowsRestartProcess) WaitExit(timeout time.Duration) error {
	ms := uint32(timeout / time.Millisecond)
	if timeout%time.Millisecond != 0 {
		ms++
	}
	event, err := windows.WaitForSingleObject(p.handle, ms)
	if err != nil {
		return err
	}
	switch event {
	case uint32(windows.WAIT_OBJECT_0):
		return nil
	case uint32(windows.WAIT_TIMEOUT):
		return fmt.Errorf("process exit wait timed out after %s", timeout)
	default:
		return fmt.Errorf("unexpected process wait result %d", event)
	}
}

func (p *windowsRestartProcess) Close() {
	p.closeOne.Do(func() { _ = windows.CloseHandle(p.handle) })
}

func queryListeningPIDs(port uint16) ([]uint32, error) {
	unique := make(map[uint32]struct{})
	for _, family := range []uint32{afInet, afInet6} {
		rows, err := readTCPTable(family)
		if err != nil {
			return nil, fmt.Errorf("TCP owner table query failed for address family %d: %w", family, err)
		}
		pids, err := listeningPIDs(rows, family, port)
		if err != nil {
			return nil, fmt.Errorf("invalid TCP owner table for address family %d: %w", family, err)
		}
		for _, pid := range pids {
			unique[pid] = struct{}{}
		}
	}
	result := make([]uint32, 0, len(unique))
	for pid := range unique {
		result = append(result, pid)
	}
	return result, nil
}

func readTCPTable(family uint32) ([]byte, error) {
	if err := procGetExtendedTCPTable.Find(); err != nil {
		return nil, err
	}
	var size uint32
	status, _, _ := procGetExtendedTCPTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		uintptr(family),
		uintptr(tcpTableOwnerPIDAll),
		0,
	)
	if status != getExtendedTCPTableSize && status != 0 {
		return nil, syscall.Errno(status)
	}
	for attempt := 0; attempt < 3; attempt++ {
		if size < 4 {
			size = 4
		}
		buffer := make([]byte, size)
		actualSize := size
		status, _, _ = procGetExtendedTCPTable.Call(
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(unsafe.Pointer(&actualSize)),
			0,
			uintptr(family),
			uintptr(tcpTableOwnerPIDAll),
			0,
		)
		if status == 0 {
			if actualSize > uint32(len(buffer)) {
				return nil, fmt.Errorf("TCP owner table size changed unexpectedly")
			}
			return buffer[:actualSize], nil
		}
		if status != getExtendedTCPTableSize {
			return nil, syscall.Errno(status)
		}
		size = actualSize
	}
	return nil, fmt.Errorf("TCP owner table kept changing during lookup")
}

func listeningPIDs(table []byte, family uint32, port uint16) ([]uint32, error) {
	rowSize := tcp4OwnerRowSize
	portOffset, stateOffset, pidOffset := 8, 0, 20
	if family == afInet6 {
		rowSize, portOffset, stateOffset, pidOffset = tcp6OwnerRowSize, 20, 48, 52
	}
	if len(table) < 4 {
		return nil, fmt.Errorf("table header is truncated")
	}
	count := uint64(binary.LittleEndian.Uint32(table[:4]))
	if count > uint64((len(table)-4)/rowSize) {
		return nil, fmt.Errorf("table row count exceeds buffer")
	}
	seen := make(map[uint32]struct{})
	for i := uint64(0); i < count; i++ {
		row := table[4+int(i)*rowSize : 4+int(i+1)*rowSize]
		state := binary.LittleEndian.Uint32(row[stateOffset : stateOffset+4])
		localPort := bits.ReverseBytes16(uint16(binary.LittleEndian.Uint32(row[portOffset : portOffset+4])))
		pid := binary.LittleEndian.Uint32(row[pidOffset : pidOffset+4])
		if state == mibTCPStateListen && localPort == port && pid != 0 {
			seen[pid] = struct{}{}
		}
	}
	result := make([]uint32, 0, len(seen))
	for pid := range seen {
		result = append(result, pid)
	}
	return result, nil
}
