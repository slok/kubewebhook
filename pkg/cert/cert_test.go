package cert

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

func createTestCrt() error {
	var err error

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	publicCaKey := privateKey.Public()

	//[RFC5280]
	subjectCa := pkix.Name{
		CommonName:         "example",
		OrganizationalUnit: []string{"Example Org Unit"},
		Organization:       []string{"Example Org"},
		Country:            []string{"JP"},
	}

	crttmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subjectCa,
		NotAfter:              time.Date(2031, 12, 31, 0, 0, 0, 0, time.UTC),
		NotBefore:             time.Now(),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	//Self Sign Certificate
	certificate, err := x509.CreateCertificate(rand.Reader, crttmpl, crttmpl, publicCaKey, privateKey)
	if err != nil {
		return err
	}

	var f *os.File
	f, err = os.Create("./example.crt")
	if err != nil {
		return err
	}
	pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: certificate})
	err = f.Close()
	if err != nil {
		return err
	}

	f, err = os.Create("./example.key")
	if err != nil {
		return err
	}
	derPrivateKey := x509.MarshalPKCS1PrivateKey(privateKey)

	pem.Encode(f, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: derPrivateKey})
	err = f.Close()
	if err != nil {
		return err
	}

	return nil
}

func TestWatch(t *testing.T) {
	cb := context.Background()
	ctx, cancel := context.WithCancel(cb)
	certFile := "./example.crt"
	keyFile := "./example.key"

	defer func() {
		os.Remove(certFile)
		os.Remove(keyFile)
		cancel()
	}()

	if err := createTestCrt(); err != nil {
		t.Error("Failed to create certificate file for the first time.", err)
	}

	crt, err := New(certFile, keyFile)
	if err != nil {
		t.Error("Failed to read certificate file for the first time.", err)
		os.Exit(1)
	}

	go func() {
		if err := crt.Start(ctx); err != nil {
			t.Error("Unable to start watch for certificate file", err)
			os.Exit(1)
		}
	}()

	time.Sleep(time.Second * 5)
	if err := createTestCrt(); err != nil {
		t.Error("Test to create certificate file failed", err)
	}
	time.Sleep(time.Second * 1)
}
