package alexa

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// certURLCacheTTL bounds how long a fetched signing certificate is reused
	// (Amazon recommends at least 6h; shorter would hammer S3 on every call).
	certURLCacheTTL = 6 * time.Hour
	// maxTimestampSkew is the widest acceptable age for the request timestamp
	// (Amazon recommends 150s to resist replay attacks).
	maxTimestampSkew = 150 * time.Second
	// maxCertBytes caps the signing certificate download.
	maxCertBytes = 64 << 10

	signatureHeader = "Signature"
	certURLHeader   = "SignatureCertChainUrl"
)

// Verifier validates that a request genuinely comes from Alexa (PRD §18):
// the signing certificate URL is a fixed Amazon location, the certificate
// chains to a trusted root, the RSA signature over the raw body matches,
// the timestamp is fresh, and the skill ID is the configured one.
type Verifier struct {
	// SkillID is the application ID of this skill; requests from any other
	// skill are rejected. Empty disables the skill ID check (local dev only).
	SkillID string
	// Client fetches signing certificates. Defaults to a 10s-timeout client.
	Client *http.Client
	// Now is injectable for timestamp tests.
	Now func() time.Time
	// Roots is the trust store for certificate verification; nil means the
	// system pool.
	Roots *x509.CertPool

	certCache sync.Map // url -> cachedCerts
}

type cachedCerts struct {
	certs []*x509.Certificate
	at    time.Time
}

// NewVerifier builds a verifier for the given skill ID.
func NewVerifier(skillID string) *Verifier {
	return &Verifier{
		SkillID: skillID,
		Client:  &http.Client{Timeout: 10 * time.Second},
		Now:     time.Now,
	}
}

// Verify checks the signature, certificate chain and timestamp of a raw
// request body plus its HTTP headers.
func (v *Verifier) Verify(ctx context.Context, raw []byte, h http.Header) error {
	certURL := h.Get(certURLHeader)
	if certURL == "" {
		return fmt.Errorf("%w: missing %s header", ErrUnauthorized, certURLHeader)
	}
	sigB64 := h.Get(signatureHeader)
	if sigB64 == "" {
		return fmt.Errorf("%w: missing %s header", ErrUnauthorized, signatureHeader)
	}
	if err := validateCertURL(certURL); err != nil {
		return err
	}

	certs, err := v.fetchCerts(ctx, certURL)
	if err != nil {
		return err
	}

	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		return fmt.Errorf("%w: invalid request JSON", ErrUnauthorized)
	}
	if err := v.checkTimestamp(req.Request.Timestamp); err != nil {
		return err
	}

	sig, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Errorf("%w: invalid signature encoding", ErrUnauthorized)
	}
	if err := verifySignature(certs[0].PublicKey, sig, raw); err != nil {
		return err
	}
	return verifyChain(v.Roots, certs)
}

// CheckSkillID verifies the request comes from this skill's application.
// An unset SkillID disables the check (documented dev mode).
func (v *Verifier) CheckSkillID(req *Request) error {
	if v.SkillID == "" {
		return nil
	}
	got := req.Context.System.Application.ApplicationID
	if got == "" {
		got = req.Session.Application.ApplicationID
	}
	if got != v.SkillID {
		return fmt.Errorf("%w: skill ID %q does not match %q", ErrUnauthorized, got, v.SkillID)
	}
	return nil
}

// validateCertURL enforces Amazon's rules for the signing certificate
// location: https, the dedicated S3 bucket, and the /echo.api/ path.
func validateCertURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: invalid cert URL", ErrUnauthorized)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("%w: cert URL must use https", ErrUnauthorized)
	}
	if !strings.EqualFold(u.Host, "s3.amazonaws.com") {
		return fmt.Errorf("%w: cert URL host %q is not s3.amazonaws.com", ErrUnauthorized, u.Host)
	}
	if u.User != nil || !strings.HasPrefix(u.Path, "/echo.api/") {
		return fmt.Errorf("%w: cert URL must not carry credentials or leave /echo.api/", ErrUnauthorized)
	}
	return nil
}

func (v *Verifier) checkTimestamp(ts string) error {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return fmt.Errorf("%w: invalid request timestamp", ErrUnauthorized)
	}
	skew := v.Now().Sub(t)
	if skew < -maxTimestampSkew || skew > maxTimestampSkew {
		return fmt.Errorf("%w: request timestamp outside %s window", ErrUnauthorized, maxTimestampSkew)
	}
	return nil
}

func verifySignature(pub any, sig, raw []byte) error {
	hashed := sha256.Sum256(raw)
	switch k := pub.(type) {
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(k, crypto.SHA256, hashed[:], sig); err != nil {
			return fmt.Errorf("%w: signature mismatch", ErrUnauthorized)
		}
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(k, hashed[:], sig) {
			return fmt.Errorf("%w: signature mismatch", ErrUnauthorized)
		}
	default:
		return fmt.Errorf("%w: unsupported certificate key type %T", ErrUnauthorized, pub)
	}
	return nil
}

// verifyChain ensures the fetched certificate chains to a trusted root.
// Amazon returns the chain as a PEM bundle; the first block is the leaf.
func verifyChain(roots *x509.CertPool, certs []*x509.Certificate) error {
	if roots == nil {
		var err error
		roots, err = x509.SystemCertPool()
		if err != nil {
			return fmt.Errorf("%w: system cert pool unavailable: %v", ErrUnauthorized, err)
		}
	}
	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}
	if _, err := certs[0].Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return fmt.Errorf("%w: certificate chain invalid: %v", ErrUnauthorized, err)
	}
	return nil
}

func (v *Verifier) fetchCerts(ctx context.Context, certURL string) ([]*x509.Certificate, error) {
	if cached, ok := v.certCache.Load(certURL); ok {
		cc := cached.(cachedCerts)
		if time.Since(cc.at) < certURLCacheTTL {
			return cc.certs, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, certURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: bad cert URL", ErrUnauthorized)
	}
	resp, err := v.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot fetch signing certificate: %v", ErrUnauthorized, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: certificate fetch status %d", ErrUnauthorized, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCertBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read signing certificate: %v", ErrUnauthorized, err)
	}

	certs, err := parseCertChain(body)
	if err != nil {
		return nil, err
	}
	v.certCache.Store(certURL, cachedCerts{certs: certs, at: v.Now()})
	return certs, nil
}

func parseCertChain(body []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	rest := body
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot parse signing certificate: %v", ErrUnauthorized, err)
		}
		certs = append(certs, c)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("%w: certificate bundle contains no certificates", ErrUnauthorized)
	}
	return certs, nil
}
