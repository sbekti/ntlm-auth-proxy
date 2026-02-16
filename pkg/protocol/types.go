package protocol

// AuthRequest represents the specific parameters for an MS-CHAPv2 NTLM auth request
type AuthRequest struct {
	Username     string `json:"username"`
	Domain       string `json:"domain"`
	Challenge    string `json:"challenge"`
	NTResponse   string `json:"nt_response"`
	RequestNTKey bool   `json:"request_nt_key"`
}

// AuthResponse represents the result of the NTLM authentication
type AuthResponse struct {
	Authenticated bool   `json:"authenticated"`
	NTKey         string `json:"nt_key,omitempty"`
	Error         string `json:"error,omitempty"`
}
