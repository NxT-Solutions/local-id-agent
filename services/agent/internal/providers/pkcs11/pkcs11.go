package pkcs11

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	p11 "github.com/miekg/pkcs11"
	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/rqc-icu/localid-agent/services/agent/internal/security"
)

const (
	providerName = "pkcs11"
	pinEnvVar    = "LOCALID_PKCS11_PIN"
)

var (
	moduleFactoryFn = defaultModuleFactory
	moduleResolver  = resolveModulePath
	nowFn           = func() time.Time { return time.Now().UTC() }
	readPINEnv      = func() string { return os.Getenv(pinEnvVar) }
	newPKCS11Ctx    = p11.New
)

var defaultAutoModulePaths = map[string][]string{
	"linux": {
		"/usr/lib/libbeidpkcs11.so",
		"/usr/lib/x86_64-linux-gnu/libbeidpkcs11.so",
		"/usr/lib64/libbeidpkcs11.so",
		"/usr/lib/opensc-pkcs11.so",
		"/usr/lib/x86_64-linux-gnu/opensc-pkcs11.so",
		"/usr/lib64/opensc-pkcs11.so",
	},
	"darwin": {
		"/opt/homebrew/lib/libbeidpkcs11.dylib",
		"/usr/local/lib/libbeidpkcs11.dylib",
		"/opt/homebrew/lib/opensc-pkcs11.so",
		"/usr/local/lib/opensc-pkcs11.so",
	},
}

type Provider struct {
	cfg config.PKCS11Config

	moduleFactory moduleFactory
	resolvePath   func(string) (string, error)
	now           func() time.Time
	readPIN       func() string
}

func New(cfg config.PKCS11Config) (*Provider, error) {
	return &Provider{
		cfg:           cfg,
		moduleFactory: moduleFactoryFn,
		resolvePath:   moduleResolver,
		now:           nowFn,
		readPIN:       readPINEnv,
	}, nil
}

func (p *Provider) Name() string {
	return providerName
}

func (p *Provider) Status(ctx context.Context) (*protocol.Status, error) {
	_ = ctx

	status := &protocol.Status{
		Provider: providerName,
	}

	modulePath, err := p.resolvePath(p.cfg.ModulePath)
	if err != nil {
		return status, nil
	}

	module, initialized, err := p.openModule(modulePath)
	if err != nil {
		return status, nil
	}
	defer closeModule(module, initialized)

	_, cardPresent, err := p.selectTokenSlot(module)
	if err != nil {
		return status, nil
	}

	status.Ready = cardPresent
	status.CardPresent = cardPresent
	return status, nil
}

func (p *Provider) SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	modulePath, err := p.resolvePath(p.cfg.ModulePath)
	if err != nil {
		return nil, fmt.Errorf("resolve PKCS#11 module path: %w", err)
	}

	module, initialized, err := p.openModule(modulePath)
	if err != nil {
		return nil, fmt.Errorf("open PKCS#11 module: %w", err)
	}
	defer closeModule(module, initialized)

	slot, cardPresent, err := p.selectTokenSlot(module)
	if err != nil {
		return nil, fmt.Errorf("select token slot: %w", err)
	}
	if !cardPresent {
		return nil, fmt.Errorf("smartcard not detected")
	}

	session, err := module.OpenSession(slot, p11.CKF_SERIAL_SESSION)
	if err != nil {
		return nil, fmt.Errorf("open PKCS#11 session: %w", err)
	}
	defer func() {
		_ = module.CloseSession(session)
	}()

	certificate, privateKey, err := p.loadSigningMaterial(module, session)
	if err != nil {
		return nil, err
	}

	timestamp := p.now().UTC()
	payload, err := security.BuildCanonicalPayload(req, timestamp)
	if err != nil {
		return nil, err
	}

	signature, err := p.signPayload(module, session, privateKey, payload)
	if requiresLogin(err) {
		signature, err = p.signWithPIN(module, session, privateKey, payload)
	}
	if err != nil {
		return nil, fmt.Errorf("sign payload: %w", err)
	}

	return &protocol.SignChallengeResponse{
		Provider:    providerName,
		Algorithm:   "RS256",
		Challenge:   req.Challenge,
		Signature:   base64.RawURLEncoding.EncodeToString(signature),
		Certificate: base64.StdEncoding.EncodeToString(certificate),
		SignedAt:    timestamp.Format(time.RFC3339),
	}, nil
}

type moduleFactory func(string) (module, error)

type module interface {
	Initialize(...p11.InitializeOption) error
	Finalize() error
	Destroy()
	GetSlotList(bool) ([]uint, error)
	GetTokenInfo(uint) (p11.TokenInfo, error)
	OpenSession(uint, uint) (p11.SessionHandle, error)
	CloseSession(p11.SessionHandle) error
	Login(p11.SessionHandle, uint, string) error
	Logout(p11.SessionHandle) error
	FindObjectsInit(p11.SessionHandle, []*p11.Attribute) error
	FindObjects(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error)
	FindObjectsFinal(p11.SessionHandle) error
	GetAttributeValue(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error)
	SignInit(p11.SessionHandle, []*p11.Mechanism, p11.ObjectHandle) error
	Sign(p11.SessionHandle, []byte) ([]byte, error)
}

func defaultModuleFactory(modulePath string) (module, error) {
	ctx := newPKCS11Ctx(modulePath)
	if ctx == nil {
		return nil, fmt.Errorf("create PKCS#11 context")
	}
	return ctx, nil
}

func resolveModulePath(configValue string) (string, error) {
	trimmed := strings.TrimSpace(configValue)
	if trimmed != "" && !strings.EqualFold(trimmed, "auto") {
		if _, err := os.Stat(trimmed); err != nil {
			return "", fmt.Errorf("configured module path %q is not available: %w", trimmed, err)
		}
		return trimmed, nil
	}

	for _, candidate := range defaultAutoModulePaths[runtime.GOOS] {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("no PKCS#11 module found for %s", runtime.GOOS)
}

func (p *Provider) openModule(modulePath string) (module, bool, error) {
	module, err := p.moduleFactory(modulePath)
	if err != nil {
		return nil, false, err
	}

	initialized := false
	if err := module.Initialize(); err != nil {
		if !isPKCS11Error(err, p11.CKR_CRYPTOKI_ALREADY_INITIALIZED) {
			module.Destroy()
			return nil, false, err
		}
	} else {
		initialized = true
	}

	return module, initialized, nil
}

func closeModule(module module, initialized bool) {
	if initialized {
		_ = module.Finalize()
	}
	module.Destroy()
}

func (p *Provider) selectTokenSlot(module module) (uint, bool, error) {
	slots, err := module.GetSlotList(false)
	if err != nil {
		return 0, false, err
	}

	targetLabel := strings.TrimSpace(p.cfg.TokenLabel)
	for _, slot := range slots {
		info, err := module.GetTokenInfo(slot)
		if err != nil {
			if isTokenUnavailableError(err) {
				continue
			}
			return 0, false, err
		}

		if info.Flags&p11.CKF_TOKEN_PRESENT == 0 {
			continue
		}

		if targetLabel != "" && strings.TrimSpace(info.Label) != targetLabel {
			continue
		}

		return slot, true, nil
	}

	return 0, false, nil
}

func (p *Provider) loadSigningMaterial(module module, session p11.SessionHandle) ([]byte, p11.ObjectHandle, error) {
	certTemplate := []*p11.Attribute{
		p11.NewAttribute(p11.CKA_CLASS, p11.CKO_CERTIFICATE),
	}
	if label := strings.TrimSpace(p.cfg.CertificateLabel); label != "" {
		certTemplate = append(certTemplate, p11.NewAttribute(p11.CKA_LABEL, label))
	}

	certificateObject, err := findFirstObject(module, session, certTemplate)
	if err != nil {
		return nil, 0, fmt.Errorf("find certificate object: %w", err)
	}

	attrs, err := module.GetAttributeValue(session, certificateObject, []*p11.Attribute{
		p11.NewAttribute(p11.CKA_VALUE, nil),
		p11.NewAttribute(p11.CKA_ID, nil),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("read certificate attributes: %w", err)
	}

	certificate := getAttribute(attrs, p11.CKA_VALUE)
	if len(certificate) == 0 {
		return nil, 0, fmt.Errorf("certificate attribute CKA_VALUE is empty")
	}
	if _, err := x509.ParseCertificate(certificate); err != nil {
		return nil, 0, fmt.Errorf("parse certificate DER: %w", err)
	}

	keyTemplate := []*p11.Attribute{
		p11.NewAttribute(p11.CKA_CLASS, p11.CKO_PRIVATE_KEY),
	}
	certID := getAttribute(attrs, p11.CKA_ID)
	if len(certID) > 0 {
		keyTemplate = append(keyTemplate, p11.NewAttribute(p11.CKA_ID, certID))
	}

	privateKey, err := findFirstObject(module, session, keyTemplate)
	if err != nil {
		return nil, 0, fmt.Errorf("find private key object: %w", err)
	}

	return certificate, privateKey, nil
}

func findFirstObject(module module, session p11.SessionHandle, template []*p11.Attribute) (handle p11.ObjectHandle, err error) {
	if err = module.FindObjectsInit(session, template); err != nil {
		return 0, err
	}
	defer func() {
		finalErr := module.FindObjectsFinal(session)
		if err == nil && finalErr != nil {
			err = finalErr
		}
	}()

	objects, _, err := module.FindObjects(session, 1)
	if err != nil {
		return 0, err
	}
	if len(objects) == 0 {
		return 0, fmt.Errorf("no matching PKCS#11 object")
	}

	return objects[0], nil
}

func getAttribute(attrs []*p11.Attribute, attributeType uint) []byte {
	for _, attr := range attrs {
		if attr != nil && attr.Type == attributeType {
			return attr.Value
		}
	}
	return nil
}

func (p *Provider) signPayload(module module, session p11.SessionHandle, key p11.ObjectHandle, payload []byte) ([]byte, error) {
	if err := module.SignInit(session, []*p11.Mechanism{
		p11.NewMechanism(p11.CKM_SHA256_RSA_PKCS, nil),
	}, key); err != nil {
		return nil, err
	}

	return module.Sign(session, payload)
}

func (p *Provider) signWithPIN(module module, session p11.SessionHandle, key p11.ObjectHandle, payload []byte) ([]byte, error) {
	pin, err := p.resolvePIN()
	if err != nil {
		return nil, err
	}

	if err := module.Login(session, p11.CKU_USER, pin); err != nil && !isPKCS11Error(err, p11.CKR_USER_ALREADY_LOGGED_IN) {
		return nil, fmt.Errorf("token login failed: %w", err)
	}
	defer func() {
		_ = module.Logout(session)
	}()

	return p.signPayload(module, session, key, payload)
}

func (p *Provider) resolvePIN() (string, error) {
	if !strings.EqualFold(strings.TrimSpace(p.cfg.PINPrompt), "terminal") {
		return "", fmt.Errorf("PIN prompt %q is not supported", p.cfg.PINPrompt)
	}

	pin := p.readPIN()
	if pin == "" {
		return "", fmt.Errorf("PIN is required; set %s", pinEnvVar)
	}

	return pin, nil
}

func isTokenUnavailableError(err error) bool {
	return isPKCS11Error(err, p11.CKR_TOKEN_NOT_PRESENT) ||
		isPKCS11Error(err, p11.CKR_DEVICE_REMOVED) ||
		isPKCS11Error(err, p11.CKR_SLOT_ID_INVALID)
}

func requiresLogin(err error) bool {
	return isPKCS11Error(err, p11.CKR_USER_NOT_LOGGED_IN) ||
		isPKCS11Error(err, p11.CKR_PIN_INCORRECT) ||
		isPKCS11Error(err, p11.CKR_PIN_LOCKED) ||
		isPKCS11Error(err, p11.CKR_PIN_EXPIRED)
}

func isPKCS11Error(err error, code p11.Error) bool {
	var pkcs11Err p11.Error
	if !errors.As(err, &pkcs11Err) {
		return false
	}
	return pkcs11Err == code
}
