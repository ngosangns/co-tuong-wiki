package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeTimeoutReturnsGatewayTimeout(t *testing.T) {
	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", writeSlowUCIEngine(t))
	t.Setenv("ENGINE_DEFAULT_TIME_MS", "1")
	t.Setenv("ENGINE_MAX_TIME_MS", "1")

	body := bytes.NewBufferString(`{"fen":"9/9/9/9/9/9/9/9/9/9 w - - 0 1","sideToMove":"red"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/analyze", body)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusGatewayTimeout, response.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if strings.Contains(payload["error"], "context deadline exceeded") {
		t.Fatalf("error leaks context deadline: %q", payload["error"])
	}
	if payload["error"] == "" {
		t.Fatal("expected public error message")
	}
}

func writeSlowUCIEngine(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "slow-uci.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    uci)
      echo "id name slow-uci"
      echo "uciok"
      ;;
    isready)
      echo "readyok"
      ;;
    go*)
      sleep 2
      ;;
    quit)
      exit 0
      ;;
  esac
done
`

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write slow engine: %v", err)
	}
	return path
}
