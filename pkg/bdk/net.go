package bdk

import (
	"bytes"
	"net"
)

func GetMac() uint64 {
	interfaces, err := net.Interfaces()
	if err != nil {
		return uint64(0)
	}
	for _, i := range interfaces {
		if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
			if i.HardwareAddr[0]&2 == 2 {
				continue
			}
			var mac uint64
			for j, b := range i.HardwareAddr {
				if j >= 8 {
					break
				}
				mac <<= 8
				mac += uint64(b)
			}
			return mac
		}
	}
	return uint64(0)
}
