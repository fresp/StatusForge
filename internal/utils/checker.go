package utils

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

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
