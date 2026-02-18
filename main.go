package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/sbekti/ntlm-auth-proxy/pkg/protocol"
)

var (
	ntlmAuthPath string
	logLevel     string
)

const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelError = "ERROR"
)

func main() {
	var port string
	flag.StringVar(&ntlmAuthPath, "ntlm-auth-path", "ntlm_auth", "Path to ntlm_auth binary")
	flag.StringVar(&port, "port", "9555", "Port to listen on")
	flag.StringVar(&logLevel, "log-level", "INFO", "Log level (DEBUG, INFO, ERROR)")
	flag.Parse()

	// Normalize log level
	logLevel = strings.ToUpper(logLevel)

	http.HandleFunc("/auth", handleAuth)

	log.Printf("Starting ntlm-auth-proxy server on :%s", port)
	log.Printf("Using ntlm_auth at: %s", ntlmAuthPath)
	log.Printf("Log Level: %s", logLevel)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func logMsg(level string, format string, v ...interface{}) {
	shouldLog := false
	switch logLevel {
	case LogLevelDebug:
		shouldLog = true
	case LogLevelInfo:
		if level != LogLevelDebug {
			shouldLog = true
		}
	case LogLevelError:
		if level == LogLevelError {
			shouldLog = true
		}
	}

	if shouldLog {
		log.Printf("[%s] %s", level, fmt.Sprintf(format, v...))
	}
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req protocol.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	clientIP := r.RemoteAddr
	logMsg(LogLevelInfo, "Auth attempt for user '%s' domain '%s' from %s", req.Username, req.Domain, clientIP)

	args := []string{
		"--allow-mschapv2",
		"--username=" + req.Username,
		"--challenge=" + req.Challenge,
		"--nt-response=" + req.NTResponse,
	}

	if req.Domain != "" {
		args = append(args, "--domain="+req.Domain)
	}

	if req.RequestNTKey {
		args = append(args, "--request-nt-key")
	}

	logMsg(LogLevelDebug, "Executing: %s %v", ntlmAuthPath, args)

	cmd := exec.Command(ntlmAuthPath, args...)
	outputBytes, err := cmd.CombinedOutput()
	output := string(outputBytes)
	
	resp := protocol.AuthResponse{
		Authenticated: false,
	}

	if err == nil {
		resp.Authenticated = true
		logMsg(LogLevelInfo, "Auth SUCCESS for user '%s'", req.Username)

		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "NT_KEY: ") {
				resp.NTKey = strings.TrimSpace(strings.TrimPrefix(line, "NT_KEY: "))
				logMsg(LogLevelDebug, "Captured NT_KEY for user '%s'", req.Username)
				break
			}
		}
	} else {
		if exitErr, ok := err.(*exec.ExitError); ok {
			resp.Error = strings.TrimSpace(output)
			logMsg(LogLevelInfo, "Auth FAILURE for user '%s': %s (Exit: %d)", req.Username, resp.Error, exitErr.ExitCode())
		} else {
			logMsg(LogLevelError, "System error executing ntlm_auth: %v", err)
			http.Error(w, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logMsg(LogLevelError, "Failed to encode response: %v", err)
	}
}
