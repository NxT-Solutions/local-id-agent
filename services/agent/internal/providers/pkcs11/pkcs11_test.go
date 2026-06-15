package pkcs11

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"os"
	"runtime"
	"testing"
	"time"

	p11 "github.com/miekg/pkcs11"
	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeModule struct {
	initializeFn      func() error
	finalizeFn        func() error
	destroyFn         func()
	getSlotListFn     func(bool) ([]uint, error)
	getTokenInfoFn    func(uint) (p11.TokenInfo, error)
	openSessionFn     func(uint, uint) (p11.SessionHandle, error)
	closeSessionFn    func(p11.SessionHandle) error
	loginFn           func(p11.SessionHandle, uint, string) error
	logoutFn          func(p11.SessionHandle) error
	findObjectsInitFn func(p11.SessionHandle, []*p11.Attribute) error
	findObjectsFn     func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error)
	findObjectsEndFn  func(p11.SessionHandle) error
	getAttributesFn   func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error)
	signInitFn        func(p11.SessionHandle, []*p11.Mechanism, p11.ObjectHandle) error
	signFn            func(p11.SessionHandle, []byte) ([]byte, error)
}

func (m *fakeModule) Initialize(...p11.InitializeOption) error {
	if m.initializeFn != nil {
		return m.initializeFn()
	}
	return nil
}

func (m *fakeModule) Finalize() error {
	if m.finalizeFn != nil {
		return m.finalizeFn()
	}
	return nil
}

func (m *fakeModule) Destroy() {
	if m.destroyFn != nil {
		m.destroyFn()
	}
}

func (m *fakeModule) GetSlotList(tokenPresent bool) ([]uint, error) {
	if m.getSlotListFn != nil {
		return m.getSlotListFn(tokenPresent)
	}
	return nil, nil
}

func (m *fakeModule) GetTokenInfo(slotID uint) (p11.TokenInfo, error) {
	if m.getTokenInfoFn != nil {
		return m.getTokenInfoFn(slotID)
	}
	return p11.TokenInfo{}, nil
}

func (m *fakeModule) OpenSession(slotID uint, flags uint) (p11.SessionHandle, error) {
	if m.openSessionFn != nil {
		return m.openSessionFn(slotID, flags)
	}
	return 0, nil
}

func (m *fakeModule) CloseSession(sh p11.SessionHandle) error {
	if m.closeSessionFn != nil {
		return m.closeSessionFn(sh)
	}
	return nil
}

func (m *fakeModule) Login(sh p11.SessionHandle, userType uint, pin string) error {
	if m.loginFn != nil {
		return m.loginFn(sh, userType, pin)
	}
	return nil
}

func (m *fakeModule) Logout(sh p11.SessionHandle) error {
	if m.logoutFn != nil {
		return m.logoutFn(sh)
	}
	return nil
}

func (m *fakeModule) FindObjectsInit(sh p11.SessionHandle, template []*p11.Attribute) error {
	if m.findObjectsInitFn != nil {
		return m.findObjectsInitFn(sh, template)
	}
	return nil
}

func (m *fakeModule) FindObjects(sh p11.SessionHandle, max int) ([]p11.ObjectHandle, bool, error) {
	if m.findObjectsFn != nil {
		return m.findObjectsFn(sh, max)
	}
	return nil, false, nil
}

func (m *fakeModule) FindObjectsFinal(sh p11.SessionHandle) error {
	if m.findObjectsEndFn != nil {
		return m.findObjectsEndFn(sh)
	}
	return nil
}

func (m *fakeModule) GetAttributeValue(sh p11.SessionHandle, o p11.ObjectHandle, a []*p11.Attribute) ([]*p11.Attribute, error) {
	if m.getAttributesFn != nil {
		return m.getAttributesFn(sh, o, a)
	}
	return nil, nil
}

func (m *fakeModule) SignInit(sh p11.SessionHandle, mech []*p11.Mechanism, key p11.ObjectHandle) error {
	if m.signInitFn != nil {
		return m.signInitFn(sh, mech, key)
	}
	return nil
}

func (m *fakeModule) Sign(sh p11.SessionHandle, payload []byte) ([]byte, error) {
	if m.signFn != nil {
		return m.signFn(sh, payload)
	}
	return nil, nil
}

func TestProviderBasics(t *testing.T) {
	p, err := New(config.PKCS11Config{
		Enabled:          true,
		ModulePath:       "/tmp/pkcs11.so",
		TokenLabel:       "token",
		CertificateLabel: "cert",
		PINPrompt:        "terminal",
	})
	require.NoError(t, err)
	assert.Equal(t, "pkcs11", p.Name())
	assert.Equal(t, "/tmp/pkcs11.so", p.cfg.ModulePath)
	assert.NotNil(t, p.moduleFactory)
	assert.NotNil(t, p.resolvePath)
	assert.NotNil(t, p.now)
	assert.NotNil(t, p.readPIN)
}

func TestStatus(t *testing.T) {
	t.Run("returns unavailable when resolve fails", func(t *testing.T) {
		p := &Provider{
			cfg:         config.PKCS11Config{},
			resolvePath: func(string) (string, error) { return "", errors.New("missing") },
		}
		status, err := p.Status(context.Background())
		require.NoError(t, err)
		assert.False(t, status.Ready)
		assert.False(t, status.CardPresent)
		assert.Equal(t, "PKCS#11 module is not available", status.Message)
	})

	t.Run("returns unavailable when slot scan fails", func(t *testing.T) {
		p := &Provider{
			cfg:         config.PKCS11Config{},
			resolvePath: func(string) (string, error) { return "/module.so", nil },
			moduleFactory: func(string) (module, error) {
				return &fakeModule{
					getSlotListFn: func(bool) ([]uint, error) { return nil, errors.New("slot failure") },
				}, nil
			},
		}
		status, err := p.Status(context.Background())
		require.NoError(t, err)
		assert.False(t, status.Ready)
		assert.False(t, status.CardPresent)
		assert.Equal(t, "could not scan for smartcard token", status.Message)
	})

	t.Run("returns unavailable when module open fails", func(t *testing.T) {
		p := &Provider{
			cfg:         config.PKCS11Config{},
			resolvePath: func(string) (string, error) { return "/module.so", nil },
			moduleFactory: func(string) (module, error) {
				return nil, errors.New("factory failed")
			},
		}
		status, err := p.Status(context.Background())
		require.NoError(t, err)
		assert.False(t, status.Ready)
		assert.False(t, status.CardPresent)
		assert.Equal(t, "PKCS#11 module could not be opened", status.Message)
	})

	t.Run("ready when token initialized without present flag", func(t *testing.T) {
		p := &Provider{
			cfg:         config.PKCS11Config{},
			resolvePath: func(string) (string, error) { return "/module.so", nil },
			moduleFactory: func(string) (module, error) {
				return &fakeModule{
					getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
					getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
						return p11.TokenInfo{Flags: p11.CKF_TOKEN_INITIALIZED}, nil
					},
				}, nil
			},
		}
		status, err := p.Status(context.Background())
		require.NoError(t, err)
		assert.True(t, status.Ready)
		assert.True(t, status.CardPresent)
		assert.Empty(t, status.Message)
	})

	t.Run("ready when token present", func(t *testing.T) {
		p := &Provider{
			cfg:         config.PKCS11Config{},
			resolvePath: func(string) (string, error) { return "/module.so", nil },
			moduleFactory: func(string) (module, error) {
				return &fakeModule{
					getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
					getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
						return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT}, nil
					},
				}, nil
			},
		}
		status, err := p.Status(context.Background())
		require.NoError(t, err)
		assert.True(t, status.Ready)
		assert.True(t, status.CardPresent)
		assert.Empty(t, status.Message)
	})

	t.Run("not ready when no token detected", func(t *testing.T) {
		p := &Provider{
			cfg:         config.PKCS11Config{},
			resolvePath: func(string) (string, error) { return "/module.so", nil },
			moduleFactory: func(string) (module, error) {
				return &fakeModule{
					getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
					getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
						return p11.TokenInfo{Flags: 0}, nil
					},
				}, nil
			},
		}
		status, err := p.Status(context.Background())
		require.NoError(t, err)
		assert.False(t, status.Ready)
		assert.False(t, status.CardPresent)
		assert.Equal(t, "no smartcard token detected; insert your eID card and retry", status.Message)
	})
}

func TestSignChallenge(t *testing.T) {
	validCert := mustCreateCertificateDER(t)
	req := validRequest()
	fixed := time.Date(2026, time.June, 15, 1, 0, 0, 0, time.UTC)

	t.Run("handles context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		p := &Provider{}
		resp, err := p.SignChallenge(ctx, req)
		assert.Nil(t, resp)
		require.Error(t, err)
	})

	t.Run("fails when card absent", func(t *testing.T) {
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return nil, nil },
		}, fixed)
		resp, err := p.SignChallenge(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "smartcard not detected")
	})

	t.Run("fails when module path resolution fails", func(t *testing.T) {
		p, err := New(config.PKCS11Config{})
		require.NoError(t, err)
		p.resolvePath = func(string) (string, error) { return "", errors.New("no module") }
		resp, err := p.SignChallenge(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resolve PKCS#11 module path")
	})

	t.Run("fails when module initialization fails", func(t *testing.T) {
		p, err := New(config.PKCS11Config{})
		require.NoError(t, err)
		p.resolvePath = func(string) (string, error) { return "/module.so", nil }
		p.moduleFactory = func(string) (module, error) { return nil, errors.New("factory") }
		resp, err := p.SignChallenge(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "open PKCS#11 module")
	})

	t.Run("fails when slot selection errors", func(t *testing.T) {
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return nil, errors.New("slot") },
		}, fixed)
		resp, err := p.SignChallenge(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "select token slot")
	})

	t.Run("fails when opening session", func(t *testing.T) {
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT}, nil
			},
			openSessionFn: func(uint, uint) (p11.SessionHandle, error) { return 0, errors.New("session") },
		}, fixed)
		resp, err := p.SignChallenge(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "open PKCS#11 session")
	})

	t.Run("fails when loading materials", func(t *testing.T) {
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT}, nil
			},
			openSessionFn: func(uint, uint) (p11.SessionHandle, error) { return 1, nil },
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				return nil, false, nil
			},
		}, fixed)
		resp, err := p.SignChallenge(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find certificate object")
	})

	t.Run("fails when canonical payload invalid", func(t *testing.T) {
		findCalls := 0
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT}, nil
			},
			openSessionFn: func(uint, uint) (p11.SessionHandle, error) { return 1, nil },
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				findCalls++
				if findCalls == 1 {
					return []p11.ObjectHandle{10}, false, nil
				}
				return []p11.ObjectHandle{11}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: validCert},
				}, nil
			},
		}, fixed)
		invalid := req
		invalid.Challenge = ""
		resp, err := p.SignChallenge(context.Background(), invalid)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "challenge is required")
	})

	t.Run("fails with non-login sign error", func(t *testing.T) {
		findCalls := 0
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT}, nil
			},
			openSessionFn: func(uint, uint) (p11.SessionHandle, error) { return 1, nil },
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				findCalls++
				if findCalls == 1 {
					return []p11.ObjectHandle{10}, false, nil
				}
				return []p11.ObjectHandle{11}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: validCert},
				}, nil
			},
			signFn: func(p11.SessionHandle, []byte) ([]byte, error) {
				return nil, errors.New("sign failed")
			},
		}, fixed)
		resp, err := p.SignChallenge(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sign payload")
	})

	t.Run("successful signing", func(t *testing.T) {
		findCalls := 0
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{9}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT}, nil
			},
			openSessionFn: func(slot uint, flags uint) (p11.SessionHandle, error) {
				assert.Equal(t, uint(9), slot)
				assert.Equal(t, uint(p11.CKF_SERIAL_SESSION), flags)
				return 42, nil
			},
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				findCalls++
				if findCalls == 1 {
					return []p11.ObjectHandle{111}, false, nil
				}
				return []p11.ObjectHandle{222}, false, nil
			},
			getAttributesFn: func(_ p11.SessionHandle, o p11.ObjectHandle, _ []*p11.Attribute) ([]*p11.Attribute, error) {
				assert.Equal(t, p11.ObjectHandle(111), o)
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: validCert},
					{Type: p11.CKA_ID, Value: []byte("kid")},
				}, nil
			},
			signFn: func(_ p11.SessionHandle, payload []byte) ([]byte, error) {
				assert.NotEmpty(t, payload)
				return []byte("signature"), nil
			},
		}, fixed)

		resp, err := p.SignChallenge(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, "pkcs11", resp.Provider)
		assert.Equal(t, "RS256", resp.Algorithm)
		assert.Equal(t, req.Challenge, resp.Challenge)
		assert.NotEmpty(t, resp.Signature)
		assert.NotEmpty(t, resp.Certificate)
		assert.Equal(t, fixed.Format(time.RFC3339), resp.SignedAt)
	})

	t.Run("retries after login", func(t *testing.T) {
		findCalls := 0
		signCalls := 0
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT}, nil
			},
			openSessionFn: func(uint, uint) (p11.SessionHandle, error) { return 12, nil },
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				findCalls++
				if findCalls == 1 {
					return []p11.ObjectHandle{10}, false, nil
				}
				return []p11.ObjectHandle{11}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: validCert},
					{Type: p11.CKA_ID, Value: []byte("kid")},
				}, nil
			},
			signFn: func(p11.SessionHandle, []byte) ([]byte, error) {
				signCalls++
				if signCalls == 1 {
					return nil, p11.Error(p11.CKR_USER_NOT_LOGGED_IN)
				}
				return []byte("ok"), nil
			},
		}, fixed)
		p.readPIN = func() string { return "1234" }

		resp, err := p.SignChallenge(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 2, signCalls)
	})

	t.Run("fails when PIN prompt unsupported", func(t *testing.T) {
		findCalls := 0
		p := newSignProvider(t, &fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT}, nil
			},
			openSessionFn: func(uint, uint) (p11.SessionHandle, error) { return 1, nil },
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				findCalls++
				if findCalls == 1 {
					return []p11.ObjectHandle{10}, false, nil
				}
				return []p11.ObjectHandle{11}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: validCert},
				}, nil
			},
			signFn: func(p11.SessionHandle, []byte) ([]byte, error) {
				return nil, p11.Error(p11.CKR_USER_NOT_LOGGED_IN)
			},
		}, fixed)
		p.cfg.PINPrompt = "gui"
		resp, err := p.SignChallenge(context.Background(), req)
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})
}

func TestHelpers(t *testing.T) {
	t.Run("resolve pin", func(t *testing.T) {
		p := &Provider{
			cfg:     config.PKCS11Config{PINPrompt: "terminal"},
			readPIN: func() string { return "0000" },
		}
		pin, err := p.resolvePIN()
		require.NoError(t, err)
		assert.Equal(t, "0000", pin)
	})

	t.Run("resolve pin requires env value", func(t *testing.T) {
		p := &Provider{
			cfg:     config.PKCS11Config{PINPrompt: "terminal"},
			readPIN: func() string { return "" },
		}
		_, err := p.resolvePIN()
		require.Error(t, err)
		assert.Contains(t, err.Error(), pinEnvVar)
	})

	t.Run("module path configured", func(t *testing.T) {
		file := writeTempModuleFile(t)
		path, err := resolveModulePath(file)
		require.NoError(t, err)
		assert.Equal(t, file, path)
	})

	t.Run("module path configured missing", func(t *testing.T) {
		_, err := resolveModulePath("/definitely/missing/module.so")
		require.Error(t, err)
	})

	t.Run("module path auto", func(t *testing.T) {
		original := defaultAutoModulePaths[runtime.GOOS]
		t.Cleanup(func() { defaultAutoModulePaths[runtime.GOOS] = original })
		file := writeTempModuleFile(t)
		defaultAutoModulePaths[runtime.GOOS] = []string{file}
		path, err := resolveModulePath("auto")
		require.NoError(t, err)
		assert.Equal(t, file, path)
	})

	t.Run("module path missing", func(t *testing.T) {
		original := defaultAutoModulePaths[runtime.GOOS]
		t.Cleanup(func() { defaultAutoModulePaths[runtime.GOOS] = original })
		defaultAutoModulePaths[runtime.GOOS] = []string{"/definitely/missing/module.so"}
		_, err := resolveModulePath("auto")
		require.Error(t, err)
	})

	t.Run("module path blank resolves auto", func(t *testing.T) {
		original := defaultAutoModulePaths[runtime.GOOS]
		t.Cleanup(func() { defaultAutoModulePaths[runtime.GOOS] = original })
		file := writeTempModuleFile(t)
		defaultAutoModulePaths[runtime.GOOS] = []string{file}
		path, err := resolveModulePath("")
		require.NoError(t, err)
		assert.Equal(t, file, path)
	})

	t.Run("find first object returns final error", func(t *testing.T) {
		m := &fakeModule{
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				return []p11.ObjectHandle{1}, false, nil
			},
			findObjectsEndFn: func(p11.SessionHandle) error {
				return errors.New("finalize")
			},
		}
		_, err := findFirstObject(m, 1, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "finalize")
	})

	t.Run("find first object init error", func(t *testing.T) {
		m := &fakeModule{
			findObjectsInitFn: func(p11.SessionHandle, []*p11.Attribute) error {
				return errors.New("init")
			},
		}
		_, err := findFirstObject(m, 1, nil)
		require.Error(t, err)
	})

	t.Run("find first object find error", func(t *testing.T) {
		m := &fakeModule{
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				return nil, false, errors.New("find")
			},
		}
		_, err := findFirstObject(m, 1, nil)
		require.Error(t, err)
	})

	t.Run("find first object empty result", func(t *testing.T) {
		m := &fakeModule{}
		_, err := findFirstObject(m, 1, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no matching")
	})

	t.Run("pkcs11 error helpers", func(t *testing.T) {
		assert.True(t, isPKCS11Error(p11.Error(p11.CKR_TOKEN_NOT_PRESENT), p11.CKR_TOKEN_NOT_PRESENT))
		assert.False(t, isPKCS11Error(errors.New("x"), p11.CKR_TOKEN_NOT_PRESENT))
		assert.True(t, isTokenUnavailableError(p11.Error(p11.CKR_TOKEN_NOT_PRESENT)))
		assert.True(t, requiresLogin(p11.Error(p11.CKR_USER_NOT_LOGGED_IN)))
		assert.False(t, requiresLogin(errors.New("no")))
	})

	t.Run("select token slot filters labels", func(t *testing.T) {
		p := &Provider{cfg: config.PKCS11Config{TokenLabel: "chosen"}}
		slot, present, err := p.selectTokenSlot(&fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1, 2}, nil },
			getTokenInfoFn: func(slot uint) (p11.TokenInfo, error) {
				if slot == 1 {
					return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT, Label: "other"}, nil
				}
				return p11.TokenInfo{Flags: p11.CKF_TOKEN_PRESENT, Label: "chosen"}, nil
			},
		})
		require.NoError(t, err)
		assert.True(t, present)
		assert.Equal(t, uint(2), slot)
	})

	t.Run("select token slot ignores unavailable entries", func(t *testing.T) {
		p := &Provider{}
		slot, present, err := p.selectTokenSlot(&fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{}, p11.Error(p11.CKR_TOKEN_NOT_PRESENT)
			},
		})
		require.NoError(t, err)
		assert.False(t, present)
		assert.Equal(t, uint(0), slot)
	})

	t.Run("select token slot ignores not-present flags", func(t *testing.T) {
		p := &Provider{}
		slot, present, err := p.selectTokenSlot(&fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{Flags: 0}, nil
			},
		})
		require.NoError(t, err)
		assert.False(t, present)
		assert.Equal(t, uint(0), slot)
	})

	t.Run("select token slot returns non-availability errors", func(t *testing.T) {
		p := &Provider{}
		_, _, err := p.selectTokenSlot(&fakeModule{
			getSlotListFn: func(bool) ([]uint, error) { return []uint{1}, nil },
			getTokenInfoFn: func(uint) (p11.TokenInfo, error) {
				return p11.TokenInfo{}, errors.New("token info failed")
			},
		})
		require.Error(t, err)
	})

	t.Run("load signing material error paths", func(t *testing.T) {
		p := &Provider{}
		_, _, _, err := p.loadSigningMaterial(&fakeModule{
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				return []p11.ObjectHandle{1}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return nil, errors.New("attrs")
			},
		}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read certificate attributes")

		_, _, _, err = p.loadSigningMaterial(&fakeModule{
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				return []p11.ObjectHandle{1}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{}, nil
			},
		}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CKA_VALUE is empty")

		_, _, _, err = p.loadSigningMaterial(&fakeModule{
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				return []p11.ObjectHandle{1}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: []byte("invalid")},
				}, nil
			},
		}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse certificate DER")

		call := 0
		_, _, _, err = p.loadSigningMaterial(&fakeModule{
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				call++
				if call == 1 {
					return []p11.ObjectHandle{1}, false, nil
				}
				return nil, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: mustCreateCertificateDER(t)},
				}, nil
			},
		}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "find private key object")

		p.cfg.CertificateLabel = "sign-cert"
		call = 0
		_, _, _, err = p.loadSigningMaterial(&fakeModule{
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				call++
				if call == 1 {
					return []p11.ObjectHandle{1}, false, nil
				}
				return []p11.ObjectHandle{2}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: mustCreateCertificateDER(t)},
				}, nil
			},
		}, 1)
		require.NoError(t, err)
	})

	t.Run("get attribute handles nil entries", func(t *testing.T) {
		value := getAttribute([]*p11.Attribute{
			nil,
			{Type: p11.CKA_ID, Value: []byte("abc")},
		}, p11.CKA_ID)
		assert.Equal(t, []byte("abc"), value)
	})

	t.Run("sign payload falls back when SHA256_RSA_PKCS unsupported", func(t *testing.T) {
		p := &Provider{}
		initCalls := 0
		var signedPayload []byte

		sig, err := p.signRSAPayload(&fakeModule{
			signInitFn: func(_ p11.SessionHandle, mech []*p11.Mechanism, _ p11.ObjectHandle) error {
				initCalls++
				if initCalls == 1 {
					assert.Equal(t, uint(p11.CKM_SHA256_RSA_PKCS), mech[0].Mechanism)
					return p11.Error(p11.CKR_MECHANISM_INVALID)
				}
				assert.Equal(t, uint(p11.CKM_RSA_PKCS), mech[0].Mechanism)
				return nil
			},
			signFn: func(_ p11.SessionHandle, payload []byte) ([]byte, error) {
				signedPayload = payload
				return []byte("signature"), nil
			},
		}, 1, 1, []byte("payload"))
		require.NoError(t, err)
		assert.Equal(t, []byte("signature"), sig)
		assert.Equal(t, 2, initCalls)
		require.Len(t, signedPayload, len(sha256DigestInfoPrefix)+sha256.Size)
		assert.Equal(t, sha256DigestInfoPrefix, signedPayload[:len(sha256DigestInfoPrefix)])
	})

	t.Run("sign payload and sign with pin errors", func(t *testing.T) {
		p := &Provider{
			cfg:     config.PKCS11Config{PINPrompt: "terminal"},
			readPIN: func() string { return "1234" },
		}
		_, err := p.signRSAPayload(&fakeModule{
			signInitFn: func(p11.SessionHandle, []*p11.Mechanism, p11.ObjectHandle) error {
				return errors.New("init")
			},
		}, 1, 1, []byte("payload"))
		require.Error(t, err)

		_, err = p.signWithPIN(&fakeModule{
			loginFn: func(p11.SessionHandle, uint, string) error {
				return errors.New("login")
			},
		}, 1, 1, "RS256", []byte("payload"))
		require.Error(t, err)

		_, err = p.signWithPIN(&fakeModule{
			loginFn: func(p11.SessionHandle, uint, string) error {
				return p11.Error(p11.CKR_USER_ALREADY_LOGGED_IN)
			},
			signFn: func(p11.SessionHandle, []byte) ([]byte, error) {
				return []byte("ok"), nil
			},
		}, 1, 1, "RS256", []byte("payload"))
		require.NoError(t, err)
	})

	t.Run("open module handles init errors", func(t *testing.T) {
		p := &Provider{
			moduleFactory: func(string) (module, error) {
				return &fakeModule{}, nil
			},
		}
		_, initialized, err := p.openModule("/module.so")
		require.NoError(t, err)
		assert.True(t, initialized)

		p.moduleFactory = func(string) (module, error) {
			return &fakeModule{
				initializeFn: func() error { return p11.Error(p11.CKR_CRYPTOKI_ALREADY_INITIALIZED) },
			}, nil
		}
		_, initialized, err = p.openModule("/module.so")
		require.NoError(t, err)
		assert.False(t, initialized)

		destroyCalled := false
		p.moduleFactory = func(string) (module, error) {
			return &fakeModule{
				initializeFn: func() error { return errors.New("init fail") },
				destroyFn:    func() { destroyCalled = true },
			}, nil
		}
		_, _, err = p.openModule("/module.so")
		require.Error(t, err)
		assert.True(t, destroyCalled)

		p.moduleFactory = func(string) (module, error) {
			return nil, errors.New("factory fail")
		}
		_, _, err = p.openModule("/module.so")
		require.Error(t, err)
	})

	t.Run("signing algorithm for certificate types", func(t *testing.T) {
		rsaCert, err := x509.ParseCertificate(mustCreateCertificateDER(t))
		require.NoError(t, err)
		algorithm, err := signingAlgorithmForCert(rsaCert)
		require.NoError(t, err)
		assert.Equal(t, "RS256", algorithm)

		p256Cert, err := x509.ParseCertificate(mustCreateECDSACertificateDER(t, elliptic.P256()))
		require.NoError(t, err)
		algorithm, err = signingAlgorithmForCert(p256Cert)
		require.NoError(t, err)
		assert.Equal(t, "ES256", algorithm)

		p384Cert, err := x509.ParseCertificate(mustCreateECDSACertificateDER(t, elliptic.P384()))
		require.NoError(t, err)
		algorithm, err = signingAlgorithmForCert(p384Cert)
		require.NoError(t, err)
		assert.Equal(t, "ES384", algorithm)

		p521Cert, err := x509.ParseCertificate(mustCreateECDSACertificateDER(t, elliptic.P521()))
		require.NoError(t, err)
		_, err = signingAlgorithmForCert(p521Cert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported ECDSA curve")

		unsupportedCert := &x509.Certificate{PublicKey: struct{}{}}
		_, err = signingAlgorithmForCert(unsupportedCert)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported certificate public key type")
	})

	t.Run("sign payload supports ECDSA algorithms", func(t *testing.T) {
		p := &Provider{}
		var mechanism uint
		sig, err := p.signPayload(&fakeModule{
			signInitFn: func(_ p11.SessionHandle, mech []*p11.Mechanism, _ p11.ObjectHandle) error {
				mechanism = mech[0].Mechanism
				return nil
			},
			signFn: func(p11.SessionHandle, []byte) ([]byte, error) {
				return []byte("ec-signature"), nil
			},
		}, 1, 1, "ES256", []byte("payload"))
		require.NoError(t, err)
		assert.Equal(t, []byte("ec-signature"), sig)
		assert.Equal(t, uint(p11.CKM_ECDSA_SHA256), mechanism)

		sig, err = p.signPayload(&fakeModule{
			signInitFn: func(_ p11.SessionHandle, mech []*p11.Mechanism, _ p11.ObjectHandle) error {
				mechanism = mech[0].Mechanism
				return nil
			},
			signFn: func(p11.SessionHandle, []byte) ([]byte, error) {
				return []byte("ec384-signature"), nil
			},
		}, 1, 1, "ES384", []byte("payload"))
		require.NoError(t, err)
		assert.Equal(t, []byte("ec384-signature"), sig)
		assert.Equal(t, uint(p11.CKM_ECDSA_SHA384), mechanism)

		_, err = p.signPayload(&fakeModule{}, 1, 1, "HS256", []byte("payload"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported signing algorithm")
	})

	t.Run("sign payload digest info RSA init error", func(t *testing.T) {
		p := &Provider{}
		_, err := p.signPayloadDigestInfoRSA(&fakeModule{
			signInitFn: func(p11.SessionHandle, []*p11.Mechanism, p11.ObjectHandle) error {
				return errors.New("init failed")
			},
		}, 1, 1, []byte("payload"))
		require.Error(t, err)
	})

	t.Run("load signing material unsupported curve", func(t *testing.T) {
		p := &Provider{}
		call := 0
		_, _, _, err := p.loadSigningMaterial(&fakeModule{
			findObjectsFn: func(p11.SessionHandle, int) ([]p11.ObjectHandle, bool, error) {
				call++
				if call == 1 {
					return []p11.ObjectHandle{1}, false, nil
				}
				return []p11.ObjectHandle{2}, false, nil
			},
			getAttributesFn: func(p11.SessionHandle, p11.ObjectHandle, []*p11.Attribute) ([]*p11.Attribute, error) {
				return []*p11.Attribute{
					{Type: p11.CKA_VALUE, Value: mustCreateECDSACertificateDER(t, elliptic.P521())},
				}, nil
			},
		}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported ECDSA curve")
	})
}

func TestCloseModuleAndFactory(t *testing.T) {
	finalized := false
	destroyed := false
	closeModule(&fakeModule{
		finalizeFn: func() error { finalized = true; return nil },
		destroyFn:  func() { destroyed = true },
	}, true)
	assert.True(t, finalized)
	assert.True(t, destroyed)

	originalNew := newPKCS11Ctx
	t.Cleanup(func() { newPKCS11Ctx = originalNew })
	newPKCS11Ctx = func(string) *p11.Ctx { return nil }
	_, err := defaultModuleFactory("/definitely/missing/libpkcs11.so")
	require.Error(t, err)

	newPKCS11Ctx = originalNew
	created, err := defaultModuleFactory("/definitely/missing/libpkcs11.so")
	if err == nil && created != nil {
		created.Destroy()
	}

	newPKCS11Ctx = func(string) *p11.Ctx { return &p11.Ctx{} }
	created, err = defaultModuleFactory("/any/path")
	require.NoError(t, err)
	require.NotNil(t, created)
}

func TestReadPINEnv(t *testing.T) {
	t.Setenv(pinEnvVar, "7777")
	assert.Equal(t, "7777", readPINEnv())
}

func newSignProvider(t *testing.T, fake module, now time.Time) *Provider {
	t.Helper()
	p, err := New(config.PKCS11Config{PINPrompt: "terminal"})
	require.NoError(t, err)
	p.resolvePath = func(string) (string, error) { return "/module.so", nil }
	p.moduleFactory = func(string) (module, error) { return fake, nil }
	p.now = func() time.Time { return now }
	p.readPIN = func() string { return "1234" }
	return p
}

func validRequest() protocol.SignChallengeRequest {
	return protocol.SignChallengeRequest{
		Challenge: "YWJj",
		Backend:   "http://localhost:8000",
		Purpose:   "login",
		Origin:    "http://localhost:5173",
	}
}

func mustCreateCertificateDER(t *testing.T) []byte {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "LocalID Test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	return certDER
}

func mustCreateECDSACertificateDER(t *testing.T, curve elliptic.Curve) []byte {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "LocalID ECDSA Test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	return certDER
}

func writeTempModuleFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "pkcs11-module-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}
