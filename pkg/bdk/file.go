package bdk

import "os"

func IsFile(filename string) bool {
	fd, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !fd.Mode().IsDir()
}
