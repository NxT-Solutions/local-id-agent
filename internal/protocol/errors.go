package protocol

import (
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrNotImplemented     = errors.New("not implemented")
	ErrForbidden          = errors.New("forbidden")
	ErrBadRequest         = errors.New("bad request")
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrPayloadTooLarge    = errors.New("payload too large")
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func WriteError(w http.ResponseWriter, status int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}

func WriteForbidden(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusForbidden, "forbidden", message)
}

func WriteBadRequest(w http.ResponseWriter, message string) {
	WriteError(w, http.StatusBadRequest, "bad_request", message)
}

func WriteInternalError(w http.ResponseWriter) {
	WriteError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
}

func WriteUnsupportedMediaType(w http.ResponseWriter) {
	WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
}

func WritePayloadTooLarge(w http.ResponseWriter) {
	WriteError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "request body exceeds maximum allowed size")
}
