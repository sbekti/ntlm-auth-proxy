package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/sbekti/ntlm-auth-proxy/pkg/protocol"
)

func main() {
	// Define flags to mimic ntlm_auth and add our proxy config
	var (
		username     string
		domain       string
		challenge    string
		ntResponse   string
		requestNTKey bool
		allowMSCHAPv2 bool
		httpAddress  string
	)

	flag.StringVar(&username, "username", "", "The username to authenticate")
	flag.StringVar(&domain, "domain", "", "The domain to authenticate against")
	flag.StringVar(&challenge, "challenge", "", "The MS-CHAP challenge")
	flag.StringVar(&ntResponse, "nt-response", "", "The MS-CHAP NT-Response")
	flag.BoolVar(&requestNTKey, "request-nt-key", false, "Request the NT Key")
	flag.BoolVar(&allowMSCHAPv2, "allow-mschapv2", false, "Allow MS-CHAPv2 (ignored, always allowed)")
	flag.StringVar(&httpAddress, "http-address", "http://localhost:9555", "Address of the ntlm-auth-proxy server")

	flag.Parse()

	if username == "" || challenge == "" || ntResponse == "" {
		fmt.Fprintln(os.Stderr, "Error: --username, --challenge, and --nt-response are required.")
		os.Exit(1)
	}

	// Prepare the request
	reqBody := protocol.AuthRequest{
		Username:     username,
		Domain:       domain,
		Challenge:    challenge,
		NTResponse:   ntResponse,
		RequestNTKey: requestNTKey,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		os.Exit(1)
	}

	// Send the request
	resp, err := http.Post(httpAddress, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request to proxy: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Proxy returned status %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var authResp protocol.AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling response: %v\nBody: %s\n", err, string(body))
		os.Exit(1)
	}

	// Mimic ntlm_auth output
	if authResp.Authenticated {
		if authResp.NTKey != "" {
			fmt.Printf("NT_KEY: %s\n", authResp.NTKey)
		}
		fmt.Println("OK: Authentication successful")
		os.Exit(0)
	} else {
		if authResp.Error != "" {
			fmt.Fprintf(os.Stderr, "%s\n", authResp.Error)
		}
		fmt.Println("Logon failure") // Standard ntlm_auth failure message
		os.Exit(1)
	}
}
