//go:build !windows

package lcu

func GetLolClientApiInfoAdapt() (port int, token string, err error) {
	return 0, "", ErrLolProcessNotFound
}
