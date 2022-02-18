package bdk

import (
	"math/rand"
	"os"
	"time"
)

func IsFile(filename string) bool {
	fd, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !fd.IsDir()
}
func RandomAlphaNum(lengthParam ...int) []byte {
	length := 16
	if len(lengthParam) > 0 {
		length = lengthParam[0]
	}
	bytes := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return result
}
func InArrayInt(a int, arr []int) bool {
	for _, v := range arr {
		if v == a {
			return true
		}
	}
	return false
}
