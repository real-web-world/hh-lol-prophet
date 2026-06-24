//go:build !windows

package lcu

func GetLolClientApiInfoAdapt() (port int, token string, err error) {
	return 1981, "Ms7GRH9S0o4YgjPf0bG8pg", nil
}
