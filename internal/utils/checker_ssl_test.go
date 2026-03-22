package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSSL_NoThresholdTriggered(t *testing.T) {
	host := startTLSServerWithCert(t, time.Now().Add(-time.Hour), time.Now().Add(90*24*time.Hour))

	result, err := CheckSSL(host, 5*time.Second, []int{30, 14, 7})
	require.NoError(t, err)

	assert.Greater(t, result.DaysRemaining, 30)
	assert.Equal(t, 0, result.TriggeredThreshold)
	assert.False(t, result.Warning)
}

func TestCheckSSL_ExpiredCertificateReturnsError(t *testing.T) {
	host := startTLSServerWithCert(t, time.Now().Add(-10*24*time.Hour), time.Now().Add(-24*time.Hour))

	_, err := CheckSSL(host, 5*time.Second, []int{30, 14, 7})
	require.Error(t, err)
}

func TestCheckSSL_TriggersConfiguredThreshold(t *testing.T) {
	host := startTLSServerWithCert(t, time.Now().Add(-time.Hour), time.Now().Add(10*24*time.Hour))

	result, err := CheckSSL(host, 5*time.Second, []int{30, 14, 7})
	require.NoError(t, err)

	assert.LessOrEqual(t, result.DaysRemaining, 14)
	assert.Equal(t, 14, result.TriggeredThreshold)
	assert.True(t, result.Warning)
}

func startTLSServerWithCert(t *testing.T, notBefore, notAfter time.Time) string {
	t.Helper()

	certPEM, keyPEM := generateSelfSignedCert(t, notBefore, notAfter)
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{tlsCert}})
	require.NoError(t, err)

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})}

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		_ = server.Close()
	})

	return listener.Addr().String()
}

func generateSelfSignedCert(t *testing.T, notBefore, notAfter time.Time) ([]byte, []byte) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return certPEM, keyPEM
}
