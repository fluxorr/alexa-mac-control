package alexa

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"net/http"
	"testing"
	"time"
)

// testChain builds a root CA, an intermediate, and a leaf certificate whose
// private key signs request bodies, mirroring Amazon's chain layout.
type testChain struct {
	leaf    *x509.Certificate
	leafKey *rsa.PrivateKey
	certs   []*x509.Certificate
	roots   *x509.CertPool
}

func newTestChain(t *testing.T) *testChain {
	t.Helper()

	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rootTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test Root CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	rootCert, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}

	interKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	interTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Test Intermediate CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	interDER, err := x509.CreateCertificate(rand.Reader, interTmpl, rootCert, &interKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	interCert, err := x509.ParseCertificate(interDER)
	if err != nil {
		t.Fatal(err)
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "echo-api.amazon.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, interCert, &leafKey.PublicKey, interKey)
	if err != nil {
		t.Fatal(err)
	}
	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(rootCert)

	return &testChain{leaf: leafCert, leafKey: leafKey, certs: []*x509.Certificate{leafCert, interCert, rootCert}, roots: roots}
}

// signedRequest builds a request body signed by the chain's leaf key, with
// the given timestamp and skill ID.
func (c *testChain) signedRequest(t *testing.T, ts, skillID string) ([]byte, http.Header) {
	t.Helper()
	raw := []byte(`{"version":"1.0","session":{"application":{"applicationId":"` + skillID + `"},"user":{"userId":"u"}},"context":{"System":{"application":{"applicationId":"` + skillID + `"}}},"request":{"type":"IntentRequest","requestId":"r1","timestamp":"` + ts + `","intent":{"name":"MacStatusIntent","slots":{}}}}`)
	sig, err := rsa.SignPKCS1v15(rand.Reader, c.leafKey, crypto.SHA256, hash(raw))
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set(signatureHeader, base64.StdEncoding.EncodeToString(sig))
	h.Set(certURLHeader, "https://s3.amazonaws.com/echo.api/test.pem")
	return raw, h
}

func hash(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

func (c *testChain) verifier(now time.Time) *Verifier {
	v := NewVerifier("amzn1.ask.skill.test")
	v.Now = func() time.Time { return now }
	v.Roots = c.roots
	v.certCache.Store("https://s3.amazonaws.com/echo.api/test.pem",
		cachedCerts{certs: c.certs, at: now})
	return v
}

func TestVerifyValidRequest(t *testing.T) {
	c := newTestChain(t)
	now := time.Now()
	raw, h := c.signedRequest(t, now.UTC().Format(time.RFC3339), "amzn1.ask.skill.test")

	if err := c.verifier(now).Verify(context.Background(), raw, h); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyTamperedBody(t *testing.T) {
	c := newTestChain(t)
	now := time.Now()
	raw, h := c.signedRequest(t, now.UTC().Format(time.RFC3339), "amzn1.ask.skill.test")
	raw[len(raw)/2] ^= 0xff

	if err := c.verifier(now).Verify(context.Background(), raw, h); err == nil {
		t.Error("Verify() with tampered body: want error, got nil")
	}
}

func TestVerifyWrongSignature(t *testing.T) {
	c := newTestChain(t)
	now := time.Now()
	raw, h := c.signedRequest(t, now.UTC().Format(time.RFC3339), "amzn1.ask.skill.test")
	h.Set(signatureHeader, base64.StdEncoding.EncodeToString(make([]byte, 256)))

	if err := c.verifier(now).Verify(context.Background(), raw, h); err == nil {
		t.Error("Verify() with bogus signature: want error, got nil")
	}
}

func TestVerifyStaleTimestamp(t *testing.T) {
	c := newTestChain(t)
	now := time.Now()
	raw, h := c.signedRequest(t, now.Add(-10*time.Minute).UTC().Format(time.RFC3339), "amzn1.ask.skill.test")

	if err := c.verifier(now).Verify(context.Background(), raw, h); err == nil {
		t.Error("Verify() with stale timestamp: want error, got nil")
	}
}

func TestVerifyFutureTimestamp(t *testing.T) {
	c := newTestChain(t)
	now := time.Now()
	raw, h := c.signedRequest(t, now.Add(10*time.Minute).UTC().Format(time.RFC3339), "amzn1.ask.skill.test")

	if err := c.verifier(now).Verify(context.Background(), raw, h); err == nil {
		t.Error("Verify() with future timestamp: want error, got nil")
	}
}

func TestVerifyMissingHeaders(t *testing.T) {
	c := newTestChain(t)
	now := time.Now()
	raw, _ := c.signedRequest(t, now.UTC().Format(time.RFC3339), "amzn1.ask.skill.test")

	for name, h := range map[string]http.Header{
		"no signature": {},
		"no cert url":  {signatureHeader: []string{"abc"}},
	} {
		if err := c.verifier(now).Verify(context.Background(), raw, h); err == nil {
			t.Errorf("Verify() with %s: want error, got nil", name)
		}
	}
}

func TestValidateCertURL(t *testing.T) {
	for _, tt := range []struct {
		url     string
		wantErr bool
	}{
		{"https://s3.amazonaws.com/echo.api/test.pem", false},
		{"http://s3.amazonaws.com/echo.api/test.pem", true},
		{"https://evil.com/echo.api/test.pem", true},
		{"https://s3.amazonaws.com/other/test.pem", true},
		{"https://user:pass@s3.amazonaws.com/echo.api/test.pem", true},
		{"", true},
	} {
		err := validateCertURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateCertURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestCheckSkillID(t *testing.T) {
	req := &Request{Context: Context{}}
	req.Context.System.Application.ApplicationID = "amzn1.ask.skill.right"

	v := NewVerifier("amzn1.ask.skill.right")
	if err := v.CheckSkillID(req); err != nil {
		t.Errorf("CheckSkillID with matching ID: error = %v", err)
	}

	v = NewVerifier("amzn1.ask.skill.wrong")
	if err := v.CheckSkillID(req); err == nil {
		t.Error("CheckSkillID with mismatched ID: want error, got nil")
	}

	v = NewVerifier("")
	if err := v.CheckSkillID(req); err != nil {
		t.Errorf("CheckSkillID with empty config: error = %v, want nil (dev mode)", err)
	}
}

func TestParseCertChainRejectsGarbage(t *testing.T) {
	if _, err := parseCertChain([]byte("not a pem")); err == nil {
		t.Error("parseCertChain(garbage): want error, got nil")
	}
}

func TestVerifyECDSAKey(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("hello")
	if err := verifySignature(&key.PublicKey, nil, raw); err == nil {
		t.Error("verifySignature with nil ECDSA signature: want error")
	}
}
