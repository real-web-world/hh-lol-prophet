//go:build windows

package bootstrap

import (
	"os"

	"golang.org/x/sys/windows"
)

func initConsoleAdapt() {
	stdIn := windows.Handle(os.Stdin.Fd())
	var consoleMode uint32
	_ = windows.GetConsoleMode(stdIn, &consoleMode)
	consoleMode = consoleMode&^windows.ENABLE_QUICK_EDIT_MODE | windows.ENABLE_EXTENDED_FLAGS
	_ = windows.SetConsoleMode(stdIn, consoleMode)
}
