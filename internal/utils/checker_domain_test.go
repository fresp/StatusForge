package utils

import (
	"testing"
)

func TestExtractDomain_HTTP(t *testing.T) {
	domain, err := extractDomain("https://example.com/path", "http")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if domain != "example.com" {
		t.Fatalf("expected example.com, got %s", domain)
	}
}

func TestExtractDomain_SSLHostPort(t *testing.T) {
	domain, err := extractDomain("example.com:443", "ssl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if domain != "example.com" {
		t.Fatalf("expected example.com, got %s", domain)
	}
}

func TestExtractDomain_RejectsIP(t *testing.T) {
	_, err := extractDomain("https://127.0.0.1", "http")
	if err == nil {
		t.Fatalf("expected error for IP host")
	}
}

func TestParseExpiryFromWhois(t *testing.T) {
	raw := "Domain Name: EXAMPLE.COM\nRegistry Expiry Date: 2030-01-01T00:00:00Z\n"
	expiry, err := parseExpiryFromWhois(raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if expiry.Year() != 2030 {
		t.Fatalf("expected year 2030, got %d", expiry.Year())
	}
}
