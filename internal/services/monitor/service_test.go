package monitor

import (
	"testing"

	"github.com/fresp/Statora/internal/models"
)

func TestValidateAdvancedOptions_RejectsIgnoreTLSForNonHTTP(t *testing.T) {
	err := ValidateAdvancedOptions(models.MonitorTCP, "example.com:443", models.MonitorAdvancedOptions{IgnoreTLSError: true})
	if err == nil {
		t.Fatalf("expected validation error for ignore_tls_error on tcp monitor")
	}
}

func TestValidateAdvancedOptions_RejectsCertExpiryForNonSupportedMonitor(t *testing.T) {
	err := ValidateAdvancedOptions(models.MonitorPing, "8.8.8.8", models.MonitorAdvancedOptions{CertExpiry: true})
	if err == nil {
		t.Fatalf("expected validation error for cert_expiry on ping monitor")
	}
}

func TestValidateAdvancedOptions_RejectsDomainExpiryForIPTarget(t *testing.T) {
	err := ValidateAdvancedOptions(models.MonitorHTTP, "https://127.0.0.1", models.MonitorAdvancedOptions{DomainExpiry: true})
	if err == nil {
		t.Fatalf("expected validation error for domain_expiry with IP target")
	}
}

func TestValidateAdvancedOptions_RejectsIgnoreTLSAndCertExpiryTogether(t *testing.T) {
	err := ValidateAdvancedOptions(models.MonitorHTTP, "https://example.com", models.MonitorAdvancedOptions{IgnoreTLSError: true, CertExpiry: true})
	if err == nil {
		t.Fatalf("expected validation error for ignore_tls_error + cert_expiry")
	}
}

func TestValidateAdvancedOptions_RejectsHTTPNonHTTPSWhenTLSOptionsEnabled(t *testing.T) {
	err := ValidateAdvancedOptions(models.MonitorHTTP, "http://example.com", models.MonitorAdvancedOptions{IgnoreTLSError: true})
	if err == nil {
		t.Fatalf("expected validation error for non-https http target with tls options")
	}
}

func TestValidateAdvancedOptions_AllowsHTTPHTTPSDomainAndCert(t *testing.T) {
	err := ValidateAdvancedOptions(models.MonitorHTTP, "https://example.com", models.MonitorAdvancedOptions{DomainExpiry: true, CertExpiry: true})
	if err != nil {
		t.Fatalf("expected no validation error, got: %v", err)
	}
}

func TestValidateAdvancedOptions_AllowsSSLDomain(t *testing.T) {
	err := ValidateAdvancedOptions(models.MonitorSSL, "example.com:443", models.MonitorAdvancedOptions{DomainExpiry: true})
	if err != nil {
		t.Fatalf("expected no validation error, got: %v", err)
	}
}
