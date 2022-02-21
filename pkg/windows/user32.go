package windows

import (
	syscall "golang.org/x/sys/windows"
)

var (
	user32 = syscall.NewLazySystemDLL("user32.dll")

	SetWindowPos     = user32.NewProc("SetWindowPos")
	GetSystemMetrics = user32.NewProc("GetSystemMetrics")
)
