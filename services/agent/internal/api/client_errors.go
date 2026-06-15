package api

import (
	"strings"
)

// publicSignError maps provider failures to client-safe messages. Internal details
// (module paths, PKCS#11 errors, wrapped causes) stay in server logs only.
func publicSignError(err error) string {
	if err == nil {
		return "signing failed"
	}

	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "smartcard not detected"),
		strings.Contains(lower, "no smartcard token detected"):
		return "smartcard not detected; insert your eID card and retry"
	case strings.Contains(lower, "pin is required"):
		return "PIN is required for signing"
	case strings.Contains(lower, "pin prompt"):
		return "PIN entry is not configured"
	case strings.Contains(lower, "pin_locked"), strings.Contains(lower, "pin locked"):
		return "smartcard PIN is locked"
	case strings.Contains(lower, "pin_incorrect"), strings.Contains(lower, "pin incorrect"):
		return "incorrect smartcard PIN"
	case strings.Contains(lower, "user not logged in"):
		return "smartcard authentication required"
	case strings.Contains(lower, "token login failed"):
		return "smartcard login failed"
	default:
		return "signing failed"
	}
}
