package openapi

import (
	"errors"
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

func TestServeAgentYAML(t *testing.T) {
	rec := httptest.NewRecorder()
	err := ServeAgentYAML(rec)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/yaml", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "openapi:")
}

func TestAgentJSONErrors(t *testing.T) {
	t.Run("read failure", func(t *testing.T) {
		restoreRead := readSpecFile
		t.Cleanup(func() { readSpecFile = restoreRead })
		readSpecFile = func() ([]byte, error) { return nil, errors.New("boom") }

		_, err := AgentJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "read openapi spec")
	})

	t.Run("yaml parse failure", func(t *testing.T) {
		restoreRead := readSpecFile
		t.Cleanup(func() { readSpecFile = restoreRead })
		readSpecFile = func() ([]byte, error) { return []byte("{"), nil }

		_, err := AgentJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse openapi yaml")
	})

	t.Run("json marshal failure", func(t *testing.T) {
		restoreMarshal := jsonMarshal
		t.Cleanup(func() { jsonMarshal = restoreMarshal })
		jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("marshal fail") }

		_, err := AgentJSON()
		require.Error(t, err)
		require.Contains(t, err.Error(), "marshal openapi json")
	})
}

type errWriter struct {
	header http.Header
}

func (w *errWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}
func (w *errWriter) WriteHeader(statusCode int)  {}
func (w *errWriter) Write(p []byte) (int, error) { return 0, errors.New("write failed") }

func TestServeWritersReturnErrors(t *testing.T) {
	w := &errWriter{}
	require.Error(t, ServeAgentJSON(w))
	require.Error(t, ServeAgentYAML(w))
}

func TestServeReadFailures(t *testing.T) {
	restoreRead := readSpecFile
	t.Cleanup(func() { readSpecFile = restoreRead })
	readSpecFile = func() ([]byte, error) { return nil, errors.New("read fail") }

	rec := httptest.NewRecorder()
	require.Error(t, ServeAgentJSON(rec))
	require.Error(t, ServeAgentYAML(rec))
}
