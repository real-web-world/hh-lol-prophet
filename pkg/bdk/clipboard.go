package bdk

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

func WriteClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// windows clip.exe
		cmd = exec.Command("clip")
	case "linux":
		// X11 xclip
		cmd = exec.Command("xclip", "-selection", "c")
	default:
		return fmt.Errorf("不支持系统: %s", runtime.GOOS)
	}
	stdin := bytes.NewBufferString(text)
	cmd.Stdin = stdin
	return cmd.Run()
}
