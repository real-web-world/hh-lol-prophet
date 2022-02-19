//go:build windows
// +build windows

package process

import (
	"errors"
	"fmt"
	"reflect"
	"syscall"
	"unsafe"

	"github.com/real-web-world/hh-lol-prophet/services/logger"
	"golang.org/x/sys/windows"
)

// Windows API functions
var (
	modKernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
	procCreateToolhelp32Snapshot = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = modKernel32.NewProc("Process32FirstW")
	procProcess32Next            = modKernel32.NewProc("Process32NextW")
	queryFullProcessImageName    = modKernel32.NewProc("QueryFullProcessImageNameW")
	errNotFoundProcess           = errors.New("未找到进程")
)

// Some constants from the Windows API
const (
	ERROR_NO_MORE_FILES = 0x12
	MAX_PATH            = 260
)

// PROCESSENTRY32 is the Windows API structure that contains a process's
// information.
type PROCESSENTRY32 struct {
	Size              uint32
	CntUsage          uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	CntThreads        uint32
	ParentProcessID   uint32
	PriorityClassBase int32
	Flags             uint32
	ExeFile           [MAX_PATH]uint16
}

// Process is an implementation of Process for Windows.
type Process struct {
	pid  int
	ppid int
	exe  string
}

func (p *Process) Pid() int {
	return p.pid
}

func (p *Process) PPid() int {
	return p.ppid
}

func (p *Process) Executable() string {
	return p.exe
}

func newWindowsProcess(e *PROCESSENTRY32) *Process {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return &Process{
		pid:  int(e.ProcessID),
		ppid: int(e.ParentProcessID),
		exe:  syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

func findProcess(pid int) (*Process, error) {
	ps, err := Processes()
	if err != nil {
		return nil, err
	}

	for _, p := range ps {
		if p.Pid() == pid {
			return p, nil
		}
	}

	return nil, nil
}

func Processes() ([]*Process, error) {
	handle, _, _ := procCreateToolhelp32Snapshot.Call(
		0x00000002,
		0)
	if handle < 0 {
		return nil, syscall.GetLastError()
	}
	defer procCloseHandle.Call(handle)

	var entry PROCESSENTRY32
	entry.Size = uint32(unsafe.Sizeof(entry))
	ret, _, _ := procProcess32First.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, fmt.Errorf("Error retrieving process info.")
	}

	results := make([]*Process, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		ret, _, _ := procProcess32Next.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return results, nil
}

func GetProcessFullPath(targetName string) (string, error) {
	var pid int
	processList, err := Processes()
	if err != nil {
		return "", err
	}
	for _, processInfo := range processList {
		if processInfo.Executable() == targetName {
			pid = processInfo.Pid()
			break
		}
	}
	if pid == 0 {
		return "", errNotFoundProcess
	}
	hProcess, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		logger.Debug("无法获取到进程handle: ", err)
		return "", errNotFoundProcess
	}
	buf := [MAX_PATH]uint16{}
	size := MAX_PATH
	ret, _, lastErr := queryFullProcessImageName.Call(uintptr(hProcess), 0,
		uintptr(unsafe.Pointer(&buf)), uintptr(unsafe.Pointer(&size)))
	if ret != 1 {
		errMsg := "none"
		if lastErr != nil {
			errMsg = lastErr.Error()
		}
		return "", errors.New("获取进程全路径失败:" + errMsg)
	}
	return syscall.UTF16ToString(buf[:]), nil
}
func GetProcessCommand(targetName string) (string, error) {
	var pid int
	processList, err := Processes()
	if err != nil {
		return "", err
	}
	for _, processInfo := range processList {
		if processInfo.Executable() == targetName {
			pid = processInfo.Pid()
			break
		}
	}
	if pid == 0 {
		return "", errNotFoundProcess
	}
	return GetCmdline(uint32(pid))
}
func GetCmdline(pid uint32) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, pid)
	if err != nil {
		if e, ok := err.(windows.Errno); ok && e == windows.ERROR_ACCESS_DENIED {
			return "", nil // 没权限,忽略这个进程
		}
		return "", err
	}
	defer func() {
		_ = windows.CloseHandle(h)
	}()
	var pbi struct {
		ExitStatus                   uint32
		PebBaseAddress               uintptr
		AffinityMask                 uintptr
		BasePriority                 int32
		UniqueProcessID              uintptr
		InheritedFromUniqueProcessID uintptr
	}
	pbiLen := uint32(unsafe.Sizeof(pbi))
	err = windows.NtQueryInformationProcess(h, windows.ProcessBasicInformation, unsafe.Pointer(&pbi), pbiLen, &pbiLen)
	if err != nil {
		return "", err
	}
	var addr uint64
	d := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&addr)),
		Len:  8, Cap: 8}))
	err = windows.ReadProcessMemory(h, pbi.PebBaseAddress+32,
		&d[0], uintptr(len(d)), nil)
	if err != nil {
		return "", err
	}
	var commandLine windows.NTUnicodeString
	Len := unsafe.Sizeof(commandLine)
	d = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&commandLine)),
		Len:  int(Len), Cap: int(Len)}))
	err = windows.ReadProcessMemory(h, uintptr(addr+112),
		&d[0], Len, nil)
	if err != nil {
		return "", err
	}
	cmdData := make([]uint16, commandLine.Length/2)
	d = *(*[]byte)(unsafe.Pointer(&cmdData))
	err = windows.ReadProcessMemory(h, uintptr(unsafe.Pointer(commandLine.Buffer)),
		&d[0], uintptr(commandLine.Length), nil)
	if err != nil {
		return "", err
	}
	return windows.UTF16ToString(cmdData), nil
}
