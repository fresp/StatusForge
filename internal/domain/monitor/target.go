package monitor

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func ExtractDomain(target string, monitorType string) (string, error) {
	if monitorType == "http" {
		u, err := url.Parse(target)
		if err != nil {
			return "", err
		}
		host := u.Hostname()
		if host == "" {
			return "", fmt.Errorf("target has no hostname")
		}
		if net.ParseIP(host) != nil {
			return "", fmt.Errorf("domain_expiry does not support IP targets")
		}
		return host, nil
	}

	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("empty host")
	}
	if net.ParseIP(host) != nil {
		return "", fmt.Errorf("domain_expiry does not support IP targets")
	}

	return host, nil
}
