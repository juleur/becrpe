package utils

import "strings"

func PrettifyIP(ip string) string {
	if strings.Contains(ip, ".") {
		return strings.Split(ip, ":")[0] //ipv4 without port
	}
	leftBracket := strings.Index(ip, "[")
	rightBracket := strings.Index(ip, "]")
	return ip[leftBracket+1 : rightBracket] //ipv6 without port
}
