package bdk

import (
	"strconv"
	"strings"
)

func CompareVersion(v1, v2 string) int {
	v1Val := getVersionVal(v1)
	v2Val := getVersionVal(v2)
	if v1Val > v2Val {
		return 1
	} else if v1Val == v2Val {
		return 0
	}
	return -1
}
func getVersionVal(ver string) int {
	strArr := strings.Split(ver, ".")
	if len(strArr) != 3 {
		return 0
	}
	val := 0
	v1, _ := strconv.Atoi(strArr[0])
	v2, _ := strconv.Atoi(strArr[1])
	v3, _ := strconv.Atoi(strArr[2])
	val += v1 * 1_0000_0000
	val += v2 * 1_0000
	val += v3 * 1
	return val
}
