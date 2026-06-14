package protocol

type SignChallengeRequest struct {
	Challenge string `json:"challenge"`
	Backend   string `json:"backend"`
	Purpose   string `json:"purpose"`
	Origin    string `json:"origin"`
}

type SignChallengeResponse struct {
	Provider    string `json:"provider"`
	Algorithm   string `json:"algorithm"`
	Challenge   string `json:"challenge"`
	Signature   string `json:"signature"`
	Certificate string `json:"certificate,omitempty"`
	SignedAt    string `json:"signedAt"`
}

type Status struct {
	Provider    string `json:"provider"`
	Ready       bool   `json:"ready"`
	CardPresent bool   `json:"cardPresent"`
}

type HealthResponse struct {
	OK      bool   `json:"ok"`
	Name    string `json:"name"`
	Version string `json:"version"`
}
