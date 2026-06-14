package protocol

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()
	var body ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "bad_request", "bad input")

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, ErrorResponse{Error: "bad_request", Message: "bad input"}, decodeError(t, rec))
}

func TestWriteErrorHelpers(t *testing.T) {
	tests := []struct {
		name    string
		call    func(http.ResponseWriter)
		status  int
		errCode string
		message string
	}{
		{
			name:    "forbidden",
			call:    func(w http.ResponseWriter) { WriteForbidden(w, "nope") },
			status:  http.StatusForbidden,
			errCode: "forbidden",
			message: "nope",
		},
		{
			name:    "bad request",
			call:    func(w http.ResponseWriter) { WriteBadRequest(w, "bad") },
			status:  http.StatusBadRequest,
			errCode: "bad_request",
			message: "bad",
		},
		{
			name:    "internal error",
			call:    WriteInternalError,
			status:  http.StatusInternalServerError,
			errCode: "internal_error",
			message: "an internal error occurred",
		},
		{
			name:    "unsupported media type",
			call:    WriteUnsupportedMediaType,
			status:  http.StatusUnsupportedMediaType,
			errCode: "unsupported_media_type",
			message: "Content-Type must be application/json",
		},
		{
			name:    "payload too large",
			call:    WritePayloadTooLarge,
			status:  http.StatusRequestEntityTooLarge,
			errCode: "payload_too_large",
			message: "request body exceeds maximum allowed size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.call(rec)

			assert.Equal(t, tt.status, rec.Code)
			body := decodeError(t, rec)
			assert.Equal(t, tt.errCode, body.Error)
			assert.Equal(t, tt.message, body.Message)
		})
	}
}
