package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type SSLCheckResult struct {
	DaysRemaining      int
	TriggeredThreshold int
	Warning            bool
}

func CheckHTTP(target string, timeout time.Duration) (int, error) {
	client := &http.Client{Timeout: timeout}
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
