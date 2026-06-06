package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeSlowUCIEngineFor(t *testing.T, path string) {
	t.Helper()
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
      echo "info depth 1 score cp 12"
      echo "bestmove b0b1"
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
}

func TestLineEvaluationEndpointReturnsOneResultPerPly(t *testing.T) {
	enginePath := filepath.Join(t.TempDir(), "uci.sh")
	writeSlowUCIEngineFor(t, enginePath)

	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", enginePath)
	t.Setenv("ENGINE_DEFAULT_TIME_MS", "10")
	t.Setenv("ENGINE_MAX_TIME_MS", "10")

	body := bytes.NewBufferString(`{
		"openingFen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1",
		"plies": [
			{ "ply": 0, "fen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1", "sideToMove": "red" },
			{ "ply": 1, "fen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR b - - 0 1", "sideToMove": "black" }
		]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/line-evaluation", body)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Engine string `json:"engine"`
		Plies  []struct {
			Ply        int    `json:"ply"`
			FEN        string `json:"fen"`
			SideToMove string `json:"sideToMove"`
			BestMove   string `json:"bestMove"`
			Depth      int    `json:"depth"`
			Status     string `json:"status"`
		} `json:"plies"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Engine != "pikafish" {
		t.Fatalf("engine = %q, want pikafish", payload.Engine)
	}
	if len(payload.Plies) != 2 {
		t.Fatalf("plies count = %d, want 2", len(payload.Plies))
	}
	for _, ply := range payload.Plies {
		if ply.Status != "ok" {
			t.Errorf("ply %d status = %q, want ok", ply.Ply, ply.Status)
		}
		if ply.BestMove == "" {
			t.Errorf("ply %d has empty bestMove", ply.Ply)
		}
	}
}

func TestLineEvaluationEndpointCacheHitOnSecondRequest(t *testing.T) {
	enginePath := filepath.Join(t.TempDir(), "uci.sh")
	writeSlowUCIEngineFor(t, enginePath)
	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", enginePath)
	t.Setenv("ENGINE_DEFAULT_TIME_MS", "10")
	t.Setenv("ENGINE_MAX_TIME_MS", "10")

	body := bytes.NewBufferString(`{
		"plies": [
			{ "ply": 0, "fen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1", "sideToMove": "red" }
		]
	}`)
	server := NewServer(nil)

	first := httptest.NewRequest(http.MethodPost, "/api/line-evaluation", body)
	firstResponse := httptest.NewRecorder()
	server.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", firstResponse.Code)
	}
	if firstResponse.Header().Get("X-Cache") != "miss" {
		t.Fatalf("first X-Cache = %q, want miss", firstResponse.Header().Get("X-Cache"))
	}

	second := httptest.NewRequest(http.MethodPost, "/api/line-evaluation", bytes.NewBufferString(`{
		"plies": [
			{ "ply": 0, "fen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1", "sideToMove": "red" }
		]
	}`))
	secondResponse := httptest.NewRecorder()
	server.ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("second status = %d, want 200", secondResponse.Code)
	}
	if secondResponse.Header().Get("X-Cache") != "hit" {
		t.Fatalf("second X-Cache = %q, want hit", secondResponse.Header().Get("X-Cache"))
	}
}

func TestLineEvaluationEndpointRejectsOversize(t *testing.T) {
	plies := make([]map[string]any, 70)
	for i := range plies {
		plies[i] = map[string]any{
			"ply":        i,
			"fen":        "9/9/9/9/9/9/9/9/9/9 w - - 0 1",
			"sideToMove": "red",
		}
	}
	payload, _ := json.Marshal(map[string]any{"plies": plies})
	request := httptest.NewRequest(http.MethodPost, "/api/line-evaluation", bytes.NewBuffer(payload))
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", response.Code, response.Body.String())
	}
}
