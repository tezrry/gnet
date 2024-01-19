package net

import (
	"fmt"
	"net"
	"regexp"
)

func GetIPv4(patterns ...string) (string, error) {
	addrList, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	var regx []*regexp.Regexp
	if len(patterns) > 0 {
		regx = make([]*regexp.Regexp, len(patterns))
		for i, pat := range patterns {
			r, err := regexp.Compile(pat)
			if err != nil {
				return "", err
			}

			regx[i] = r
		}
	}

	var ip net.IP
	for _, addr := range addrList {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			ip = ipNet.IP.To4()
			if ip == nil {
				continue
			}

			ipStr := ip.String()
			if regx == nil {
				return ipStr, nil
			}

			for _, r := range regx {
				if r.MatchString(ipStr) {
					return ipStr, nil
				}
			}
		}

		if ipNet, ok := addr.(*net.IPAddr); ok && !ipNet.IP.IsLoopback() {
			ip = ipNet.IP.To4()
			if ip == nil {
				continue
			}

			ipStr := ip.String()
			if regx == nil {
				return ipStr, nil
			}

			for _, r := range regx {
				if r.MatchString(ipStr) {
					return ipStr, nil
				}
			}
		}
	}

	return "", fmt.Errorf("cannot find an IP with the pattern(s): %v", patterns)
}
