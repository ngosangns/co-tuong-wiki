package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestAnalyzeEndpointMultiEngineReturnsAgreement(t *testing.T) {
	// Build two engines with different binaries so we can verify both report
	// independently and the response includes both results.
	fairyPath := writeStubUCIEngine(t, "fairy")
	pikafishPath := writeStubUCIEngine(t, "pikafish")

	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", pikafishPath)
	t.Setenv("ENGINE_FAIRY_PATH", fairyPath)
	t.Setenv("ENGINE_DEFAULT_TIME_MS", "5")
	t.Setenv("ENGINE_MAX_TIME_MS", "5")

	body := bytes.NewBufferString(`{
		"fen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1",
		"sideToMove": "red",
		"engines": ["pikafish", "fairy-stockfish"]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/analyze", body)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Primary   *struct {
			Source string `json:"source"`
			Error  string `json:"error"`
		} `json:"primary"`
		Secondary *struct {
			Source string `json:"source"`
		} `json:"secondary"`
		Engines      []string `json:"engines"`
		Agreement    string   `json:"agreement"`
		AgreementMove string  `json:"agreementMove"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Primary == nil {
		t.Fatalf("expected primary result, got %s", response.Body.String())
	}
	if payload.Primary.Source != "pikafish" {
		t.Errorf("primary source = %q, want pikafish", payload.Primary.Source)
	}
	if payload.Secondary == nil {
		t.Fatalf("expected secondary result, got %s", response.Body.String())
	}
	if payload.Secondary.Source != "fairy-stockfish" {
		t.Errorf("secondary source = %q, want fairy-stockfish", payload.Secondary.Source)
	}
	if len(payload.Engines) != 2 {
		t.Errorf("engines = %v, want 2 entries", payload.Engines)
	}
}

func TestAnalyzeEndpointDefaultSingleEngineKeepsLegacyShape(t *testing.T) {
	enginePath := writeStubUCIEngine(t, "pikafish")
	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", enginePath)
	t.Setenv("ENGINE_FAIRY_PATH", "")
	t.Setenv("ENGINE_DEFAULT_TIME_MS", "5")
	t.Setenv("ENGINE_MAX_TIME_MS", "5")

	body := bytes.NewBufferString(`{
		"fen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1",
		"sideToMove": "red"
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/analyze", body)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Primary *struct {
			Source string `json:"source"`
		} `json:"primary"`
		Secondary *struct{} `json:"secondary"`
		Engines   []string `json:"engines"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Primary == nil {
		t.Fatalf("expected primary result")
	}
	if payload.Secondary != nil {
		t.Errorf("expected no secondary result, got %+v", payload.Secondary)
	}
	if len(payload.Engines) != 1 || payload.Engines[0] != "pikafish" {
		t.Errorf("engines = %v, want [pikafish]", payload.Engines)
	}
}

// writeStubUCIEngine writes a minimal UCI shim that responds with a fixed
// bestmove and exits when asked. It's reused by the analyze and line-eval
// tests to avoid spawning the real Pikafish binary.
func writeStubUCIEngine(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name+".sh")
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
    setoption*)
      ;;
    go*)
      echo "info depth 1 score cp 10"
      echo "bestmove b0b1"
      ;;
    quit)
      exit 0
      ;;
  esac
done
`
	if err := writeFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write engine: %v", err)
	}
	return path
}
