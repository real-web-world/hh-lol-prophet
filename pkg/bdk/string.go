package bdk

import "unsafe"

func Str2Bytes(str string) []byte {
	strData := (*[2]uintptr)(unsafe.Pointer(&str))
	byteSlice := [3]uintptr{strData[0], strData[1], strData[1]}
	return *(*[]byte)(unsafe.Pointer(&byteSlice))
}
func Bytes2Str(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}
