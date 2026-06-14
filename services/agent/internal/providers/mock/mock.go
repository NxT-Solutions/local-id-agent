package mock

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/rqc-icu/localid-agent/services/agent/internal/security"
)

// Deterministic dev key and certificate for reproducible mock signing.
var privateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDWsCtkhApwZBnn
R1NiTxSYBi1RC0EU8SZIuhrMkJ5vwtGRew4Vc56LrhxBkbZ/Rd+oUT9gv/i6RJGr
4KZ6vvjrSpIKFUdCRCkQyp6JtzPUDXjs5TxsSSZJiudQZ1A2iO9Co4c6zrQowXA3
V2iiosvDL16RtEbALnARTVTZb8KqWJDlk+rTUkVlSvTWClaIyo6ME7mYyXSjsnjY
vd9RpkT93/X/wyGJmZ+6/IOyX/9k/SFXwqsS6t073hb9PYZHwpipsyziztLKLpYh
AHm+fL/ScNiea5C7RaTnIYMr/XLXRsRSTDMwRGF+e34Anl3v84mQfybhbbVdKTdp
VcKhLzNfAgMBAAECggEABFDfVnJ3GUJR6P3clcXcNqAtKgttYAPnDqA7KLChlsKW
XvPX2SONZOZ3p/PLdOyNLf+QJBxH3krBmPB3uFD9hRFnNS+vUow3PSCtpjwaHSG2
NCD5oX2o+OKDevDQwn/nO3I5JjwAkC8vV9V4g4h9Syu5HXm/0F4+n4Jr+cEO60i6
FeqMHQbpaauzPYng7Ke3AZijIFjI04CXeRcKCEKST1uuOz4udz/Tw4hqaXuZPjrV
5/CzOUHc0nSbg50Fk7cPKvjRayid6/UptssZmkuunEyaNxXruiVf2rZz0tBopyDm
e35IpxGczGDxHTZFFY6HpHaY8gK++mDBfpLvsDl+YQKBgQD0PTTx06q0XEycDbuh
Ozpxbm9z4F5qtraayCmAFGBV2VvqeSgR9TQg0u/UMi/ofjKj9SgqwpZJ833kvWK5
lTJZLeC34lKR9Nz6eH3xbsvcEWM5VUYfQcIZy73ZRdpZou89HqhJ+rNVNmmmMthr
UPXKo4DZUwOVpJlD5lN6XU6YPwKBgQDhBq5wSfjlWTh2RVI2gsauyS/vpTLyjZhK
KVn1elzZS/ZHxqWcLK0xTs2kvISqm8XlrLYPn7AtA+VmYMhmiv7U5mcGvKpJ2qVS
N/HkC28Jpc8Na+afBtpNNAG5x8eu6TtU/nIK80GaIKz/tD6OGcV0geIlaSrhCGCQ
pHF46Guc4QKBgQDovQadxtktY6LxNjjs7FbFzrnQDyeJwzEBD+pUDZa7WBQ5vFhN
vH5/JOK7L1Pt1YwGJ1CuZpz2PVxjQ4E3XZAL4Sb5s2aBhXeqCnXhQnZq7/rAoJzg
njYLhNYVnHX04sndUUCGtqp7cg18/YizwwNlpSbccTnCOIaAaJR1z0Jc2wKBgQCA
FSR/J6uzJ8iakTSNcHyUilFtr3NDWlMfi0/4CNEGolUpX6luLoPaOAeXR/KfhZDr
/RWX6QYHaVtOnbITo/QOzKJB1Gt9JCXLmapmahemvykOc6dOR0FEhmChFVTUe07Z
Pwc8sl9Z0lWGKlsc2RBqE2/caXMNqY4FZoRsFKcEIQKBgAJHYnsR0C9wkdemP8ve
3WVXQh3+jkf8qWAy9dpl4eRwJi4gd6qLix8MuRCfceYvffkkakFPcrZtmfeL6UHs
9IXxIvxwLrK27P8kz677V1dGSaVucA5kcYnIba4nmJmx3Z53NknX18wSfvuaZnDy
0YiYZq5pwGCTC4NA0jgSdbdP
-----END PRIVATE KEY-----`

var certificatePEM = `-----BEGIN CERTIFICATE-----
MIIDFzCCAf+gAwIBAgIULLOLRJ7XLXMOZNOHAqBL5mdbcFgwDQYJKoZIhvcNAQEL
BQAwGzEZMBcGA1UEAwwQTG9jYWxJRCBNb2NrIERldjAeFw0yNjA2MTQyMjA0NDla
Fw0zNjA2MTEyMjA0NDlaMBsxGTAXBgNVBAMMEExvY2FsSUQgTW9jayBEZXYwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDWsCtkhApwZBnnR1NiTxSYBi1R
C0EU8SZIuhrMkJ5vwtGRew4Vc56LrhxBkbZ/Rd+oUT9gv/i6RJGr4KZ6vvjrSpIK
FUdCRCkQyp6JtzPUDXjs5TxsSSZJiudQZ1A2iO9Co4c6zrQowXA3V2iiosvDL16R
tEbALnARTVTZb8KqWJDlk+rTUkVlSvTWClaIyo6ME7mYyXSjsnjYvd9RpkT93/X/
wyGJmZ+6/IOyX/9k/SFXwqsS6t073hb9PYZHwpipsyziztLKLpYhAHm+fL/ScNie
a5C7RaTnIYMr/XLXRsRSTDMwRGF+e34Anl3v84mQfybhbbVdKTdpVcKhLzNfAgMB
AAGjUzBRMB0GA1UdDgQWBBTnIkiwYrLDqY9DMLTkFdkGXF2KtzAfBgNVHSMEGDAW
gBTnIkiwYrLDqY9DMLTkFdkGXF2KtzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3
DQEBCwUAA4IBAQC9yutGGZG2pCzQgItqJ4mxUSzvzTEUyo46Y8i9txG9+/TNUFGZ
yQdGGptn5QyfmDPIspw+OSMNEZvm5OyzEjE0PHohUdeYThYL3/guY0wykIk8Pk7x
El1FJAOam3i5Zn01Z42hsN3pynPE5/Lv9sfKlaF+UTKDBXLkQAt3NMuNQNwKcp/0
dg2B+wVFBKbXoHN98x8xWMOYFRMgiQ7eLJ6N9Nj0pwPSRne5eqorhhn9rRJZ9ihb
2nQJyJigKO1iiIRcapRbfb7h1Ds2XBwLtpNGLDlR1Ws6dOjTiDk+GQgZ85yNemMu
bOb3sVc9qLRpKvM3EJxFDBR2rdauY0Yftutk
-----END CERTIFICATE-----`

type Provider struct {
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	certificate []byte
}

func New() (*Provider, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("decode private key pem")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		privateKeyLegacy, legacyErr := x509.ParsePKCS1PrivateKey(block.Bytes)
		if legacyErr != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		privateKey = privateKeyLegacy
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not rsa")
	}

	certBlock, _ := pem.Decode([]byte(certificatePEM))
	if certBlock == nil {
		return nil, fmt.Errorf("decode certificate pem")
	}

	return &Provider{
		privateKey:  rsaKey,
		publicKey:   &rsaKey.PublicKey,
		certificate: certBlock.Bytes,
	}, nil
}

func (p *Provider) Name() string {
	return "mock"
}

func (p *Provider) Status(ctx context.Context) (*protocol.Status, error) {
	return &protocol.Status{
		Provider:    "mock",
		Ready:       true,
		CardPresent: true,
	}, nil
}

func (p *Provider) SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error) {
	timestamp := time.Now().UTC()

	payload, err := security.BuildCanonicalPayload(req, timestamp)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256(payload)
	signature, err := rsa.SignPKCS1v15(rand.Reader, p.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, fmt.Errorf("sign payload: %w", err)
	}

	return &protocol.SignChallengeResponse{
		Provider:    "mock",
		Algorithm:   "RS256",
		Challenge:   req.Challenge,
		Signature:   encodeBase64URL(signature),
		Certificate: base64.StdEncoding.EncodeToString(p.certificate),
		SignedAt:    timestamp.Format(time.RFC3339),
	}, nil
}

func (p *Provider) PublicKey() *rsa.PublicKey {
	return p.publicKey
}

func encodeBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
