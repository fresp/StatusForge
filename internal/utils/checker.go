package utils

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	monitordomain "github.com/fresp/StatusForge/internal/domain/monitor"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type SSLCheckResult struct {
	DaysRemaining      int
	TriggeredThreshold int
	Warning            bool
}

func CheckHTTP(target string, timeout time.Duration, ignoreTLSError ...bool) (int, error) {
	client := &http.Client{Timeout: timeout}
	if len(ignoreTLSError) > 0 && ignoreTLSError[0] {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	resp, err := client.Get(target)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func CheckTCP(target string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func CheckDNS(target string, timeout time.Duration) error {
	resolver := &net.Resolver{}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := resolver.LookupHost(ctx, target)
	return err
}

func CheckPing(target string, timeout time.Duration) error {
	conn, err := icmp.ListenPacket("udp4", "")
	if err != nil {
		c, err2 := net.DialTimeout("tcp", target+":80", timeout)
		if err2 != nil {
			return err2
		}
		c.Close()
		return nil
	}
	defer conn.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1, Data: []byte("ping")},
	}
	b, _ := msg.Marshal(nil)
	conn.SetDeadline(time.Now().Add(timeout))

	dst, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return err
	}
	if _, err := conn.WriteTo(b, dst); err != nil {
		return err
	}

	reply := make([]byte, 1500)
	if _, _, err := conn.ReadFrom(reply); err != nil {
		return err
	}
	return nil
}

func CheckSSL(target string, timeout time.Duration, thresholds []int) (SSLCheckResult, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	host, port, err := net.SplitHostPort(target)
	if err != nil {
		host = target
		port = "443"
	}
	address := net.JoinHostPort(host, port)

	dialer := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return SSLCheckResult{}, err
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return SSLCheckResult{}, fmt.Errorf("no peer certificate presented")
	}

	leaf := state.PeerCertificates[0]
	if err := leaf.VerifyHostname(host); err != nil {
		return SSLCheckResult{}, err
	}

	daysRemaining := int(time.Until(leaf.NotAfter).Hours() / 24)
	if daysRemaining < 0 {
		return SSLCheckResult{}, fmt.Errorf("certificate expired %d days ago", -daysRemaining)
	}

	triggered := pickTriggeredThreshold(daysRemaining, thresholds)

	return SSLCheckResult{
		DaysRemaining:      daysRemaining,
		TriggeredThreshold: triggered,
		Warning:            triggered > 0,
	}, nil
}

func CheckHTTPSSLCertificate(target string, timeout time.Duration, thresholds []int) (SSLCheckResult, error) {
	u, err := url.Parse(target)
	if err != nil {
		return SSLCheckResult{}, err
	}

	host := u.Hostname()
	if host == "" {
		return SSLCheckResult{}, fmt.Errorf("target has no hostname")
	}

	port := u.Port()
	if port == "" {
		port = "443"
	}

	return CheckSSL(net.JoinHostPort(host, port), timeout, thresholds)
}

type DomainCheckResult struct {
	DaysRemaining      int
	TriggeredThreshold int
	Warning            bool
}

func CheckDomain(target string, monitorType string, thresholds []int) (DomainCheckResult, error) {
	domain, err := extractDomain(target, monitorType)
	if err != nil {
		return DomainCheckResult{}, err
	}

	expiresAt, err := lookupDomainExpiry(domain)
	if err != nil {
		return DomainCheckResult{}, err
	}

	daysRemaining := int(time.Until(expiresAt).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	triggered := pickTriggeredThreshold(daysRemaining, thresholds)

	return DomainCheckResult{
		DaysRemaining:      daysRemaining,
		TriggeredThreshold: triggered,
		Warning:            triggered > 0,
	}, nil
}

func extractDomain(target string, monitorType string) (string, error) {
	return monitordomain.ExtractDomain(target, monitorType)
}

func lookupDomainExpiry(domain string) (time.Time, error) {
	if expiry, err := lookupRDAPExpiry(domain); err == nil {
		return expiry, nil
	}

	return lookupWhoisExpiry(domain)
}

func lookupRDAPExpiry(domain string) (time.Time, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://rdap.org/domain/" + domain)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return time.Time{}, fmt.Errorf("rdap returned status %d", resp.StatusCode)
	}

	var payload struct {
		Events []struct {
			EventAction string `json:"eventAction"`
			EventDate   string `json:"eventDate"`
		} `json:"events"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return time.Time{}, err
	}

	for _, event := range payload.Events {
		action := strings.ToLower(strings.TrimSpace(event.EventAction))
		if action != "expiration" && action != "expiry" && action != "expires" {
			continue
		}
		expiry, err := time.Parse(time.RFC3339, event.EventDate)
		if err == nil {
			return expiry, nil
		}
	}

	return time.Time{}, fmt.Errorf("no domain expiry event in rdap response")
}

func lookupWhoisExpiry(domain string) (time.Time, error) {
	tld := domain
	if idx := strings.LastIndex(domain, "."); idx >= 0 && idx < len(domain)-1 {
		tld = domain[idx+1:]
	}

	ianaResp, err := queryWhoisServer("whois.iana.org:43", tld)
	if err != nil {
		return time.Time{}, err
	}

	whoisServer := ""
	scanner := bufio.NewScanner(strings.NewReader(ianaResp))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToLower(line), "whois:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				whoisServer = strings.TrimSpace(parts[1])
				break
			}
		}
	}
	if whoisServer == "" {
		return time.Time{}, fmt.Errorf("no whois server found for tld %s", tld)
	}

	resp, err := queryWhoisServer(net.JoinHostPort(whoisServer, "43"), domain)
	if err != nil {
		return time.Time{}, err
	}

	expiry, err := parseExpiryFromWhois(resp)
	if err != nil {
		return time.Time{}, err
	}

	return expiry, nil
}

func queryWhoisServer(address string, query string) (string, error) {
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(conn, "%s\r\n", query); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(conn); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func parseExpiryFromWhois(response string) (time.Time, error) {
	patterns := []string{
		`(?im)^(?:Registry Expiry Date|Registrar Registration Expiration Date|Expiration Date|Expiry date|expires|paid-till|renewal date)\s*:\s*(.+)$`,
	}

	timeFormats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006.01.02",
		"02-Jan-2006",
		"02.01.2006 15:04:05",
		"2006/01/02",
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(response, -1)
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			candidate := strings.TrimSpace(m[1])
			candidate = strings.Trim(candidate, ".")
			for _, format := range timeFormats {
				if parsed, err := time.Parse(format, candidate); err == nil {
					return parsed, nil
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse domain expiry from whois response")
}

func pickTriggeredThreshold(daysRemaining int, thresholds []int) int {
	if len(thresholds) == 0 {
		return 0
	}

	valid := make([]int, 0, len(thresholds))
	for _, threshold := range thresholds {
		if threshold > 0 {
			valid = append(valid, threshold)
		}
	}
	if len(valid) == 0 {
		return 0
	}

	sort.Ints(valid)
	triggered := 0
	for _, threshold := range valid {
		if daysRemaining <= threshold {
			triggered = threshold
			break
		}
	}

	return triggered
}
