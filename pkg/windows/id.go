package windows

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
)

func GetMac() string {
	interfaces, err := net.Interfaces()
	if err != nil || len(interfaces) == 0 {
		return ""
	}
	return interfaces[0].HardwareAddr.String()
}

func GetCpuID() string {
	cmd := exec.Command("wmic", "cpu", "get", "ProcessorID")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	str := string(out)
	reg := regexp.MustCompile("\\s+")
	str = reg.ReplaceAllString(str, "")
	return str[11:]
}
