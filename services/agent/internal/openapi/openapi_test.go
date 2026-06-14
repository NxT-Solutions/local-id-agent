package openapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAgentJSON(t *testing.T) {
	data, err := AgentJSON()
	require.NoError(t, err)
	require.Contains(t, string(data), `"openapi"`)
	require.Contains(t, string(data), `"SignChallengeRequest"`)
	require.Contains(t, string(data), `"/health"`)
}

func TestServeAgentJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	err := ServeAgentJSON(rec)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}
