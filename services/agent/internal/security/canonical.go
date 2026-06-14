package security

import (
	"encoding/json"
	"fmt"
)

var canonicalMarshal = json.Marshal

func marshalCanonical(payload map[string]string) ([]byte, error) {
	// Keys must appear in alphabetical order: backend, challenge, origin, purpose, timestamp.
	ordered := struct {
		Backend   string `json:"backend"`
		Challenge string `json:"challenge"`
		Origin    string `json:"origin"`
		Purpose   string `json:"purpose"`
		Timestamp string `json:"timestamp"`
	}{
		Backend:   payload["backend"],
		Challenge: payload["challenge"],
		Origin:    payload["origin"],
		Purpose:   payload["purpose"],
		Timestamp: payload["timestamp"],
	}

	data, err := canonicalMarshal(ordered)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical payload: %w", err)
	}

	return data, nil
}
