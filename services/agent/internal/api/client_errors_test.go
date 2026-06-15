package api

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPublicSignError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil",
			err:  nil,
			want: "signing failed",
		},
		{
			name: "smartcard missing",
			err:  errors.New("smartcard not detected"),
			want: "smartcard not detected; insert your eID card and retry",
		},
		{
			name: "module path leak",
			err:  fmt.Errorf(`resolve PKCS#11 module path: configured module path "/secret/path.so" is not available`),
			want: "signing failed",
		},
		{
			name: "pin required",
			err:  errors.New("PIN is required; set LOCALID_PKCS11_PIN"),
			want: "PIN is required for signing",
		},
		{
			name: "pin locked",
			err:  errors.New("token login failed: CKR_PIN_LOCKED"),
			want: "smartcard PIN is locked",
		},
		{
			name: "no smartcard token detected",
			err:  errors.New("no smartcard token detected"),
			want: "smartcard not detected; insert your eID card and retry",
		},
		{
			name: "pin prompt unsupported",
			err:  errors.New("PIN prompt gui is not supported"),
			want: "PIN entry is not configured",
		},
		{
			name: "pin incorrect",
			err:  errors.New("token login failed: CKR_PIN_INCORRECT"),
			want: "incorrect smartcard PIN",
		},
		{
			name: "user not logged in",
			err:  errors.New("user not logged in to token"),
			want: "smartcard authentication required",
		},
		{
			name: "token login failed",
			err:  errors.New("token login failed: CKR_GENERAL_ERROR"),
			want: "smartcard login failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, publicSignError(tt.err))
		})
	}
}
