package openapi

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v3"
)

//go:embed spec/agent.openapi.yaml
var specFS embed.FS

var (
	readSpecFile = func() ([]byte, error) {
		return specFS.ReadFile("spec/agent.openapi.yaml")
	}
	yamlUnmarshal = yaml.Unmarshal
	jsonMarshal   = json.Marshal
)

// AgentJSON returns the embedded agent OpenAPI document as JSON bytes.
func AgentJSON() ([]byte, error) {
	data, err := readSpecFile()
	if err != nil {
		return nil, fmt.Errorf("read openapi spec: %w", err)
	}

	var parsed any
	if err := yamlUnmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("parse openapi yaml: %w", err)
	}

	jsonData, err := jsonMarshal(parsed)
	if err != nil {
		return nil, fmt.Errorf("marshal openapi json: %w", err)
	}

	return jsonData, nil
}

// ServeAgentJSON writes the OpenAPI JSON document to the response writer.
func ServeAgentJSON(w http.ResponseWriter) error {
	data, err := AgentJSON()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}

// ServeAgentYAML writes the embedded YAML document.
func ServeAgentYAML(w http.ResponseWriter) error {
	data, err := readSpecFile()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}
