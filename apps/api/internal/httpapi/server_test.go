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

func TestCombinedLessonEndpointLoadsBuiltArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "combined-lesson.json")
	body := []byte(`{
		"id": "7d75cc2b-a793-59ec-a71c-26fd9e172372",
		"title": "Tổng hợp toàn bộ lesson",
		"category": "Tổng hợp",
		"difficulty": "Tất cả",
		"lines": [
			{
				"id": "0c653267-670b-517f-9fe0-48323c52267a",
				"title": "Line có FEN riêng",
				"initialFen": "9/9/9/9/9/9/9/9/9/9 w - - 0 1",
				"moves": [
					{
						"id": "0563eb4a-8203-5b1b-8c6a-77eb05d5fd45",
						"side": "red",
						"from": { "file": 0, "rank": 6 },
						"to": { "file": 0, "rank": 5 },
						"comment": "first"
					}
				]
			}
		],
		"choice": { "prompt": "", "options": [] }
	}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write combined lesson: %v", err)
	}
	t.Setenv("COMBINED_LESSON_FILE", path)

	request := httptest.NewRequest(http.MethodGet, "/api/combined-lesson", nil)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var payload struct {
		ID         string `json:"id"`
		InitialFEN string `json:"initialFen"`
		Lines      []struct {
			InitialFEN string `json:"initialFen"`
			Phase      string `json:"phase"`
			PieceCount int    `json:"pieceCount"`
			MoveCount  int    `json:"moveCount"`
			Moves      []any  `json:"moves"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode combined lesson: %v", err)
	}
	if payload.InitialFEN == "" {
		t.Fatalf("combined overview should expose the default starting FEN")
	}
	if payload.ID == "" || len(payload.Lines) != 1 || payload.Lines[0].InitialFEN == "" || payload.Lines[0].MoveCount != 1 {
		t.Fatalf("unexpected combined lesson payload: %#v", payload)
	}
	if payload.Lines[0].Phase != "endgame" || payload.Lines[0].PieceCount != 0 {
		t.Fatalf("unexpected line phase metadata: %#v", payload.Lines[0])
	}
	if payload.Lines[0].Moves != nil {
		t.Fatalf("combined overview should not include moves: %#v", payload.Lines[0].Moves)
	}
}

func TestCombinedLessonEndpointUsesConditionalCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "combined-lesson.json")
	body := []byte(`{
		"id": "7d75cc2b-a793-59ec-a71c-26fd9e172372",
		"title": "Tổng hợp toàn bộ lesson",
		"category": "Tổng hợp",
		"difficulty": "Tất cả",
		"lines": [],
		"choice": { "prompt": "", "options": [] }
	}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write combined lesson: %v", err)
	}
	t.Setenv("COMBINED_LESSON_FILE", path)

	server := NewServer(nil)
	firstRequest := httptest.NewRequest(http.MethodGet, "/api/combined-lesson", nil)
	firstResponse := httptest.NewRecorder()

	server.ServeHTTP(firstResponse, firstRequest)

	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d; body = %s", firstResponse.Code, http.StatusOK, firstResponse.Body.String())
	}
	etag := firstResponse.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}
	if firstResponse.Header().Get("X-Data-Version") == "" {
		t.Fatal("expected X-Data-Version header")
	}

	secondRequest := httptest.NewRequest(http.MethodGet, "/api/combined-lesson", nil)
	secondRequest.Header.Set("If-None-Match", etag)
	secondResponse := httptest.NewRecorder()

	server.ServeHTTP(secondResponse, secondRequest)

	if secondResponse.Code != http.StatusNotModified {
		t.Fatalf("second status = %d, want %d; body = %s", secondResponse.Code, http.StatusNotModified, secondResponse.Body.String())
	}
	if secondResponse.Body.Len() != 0 {
		t.Fatalf("304 response should not include a body: %q", secondResponse.Body.String())
	}
	if secondResponse.Header().Get("X-Cache") != "revalidate" {
		t.Fatalf("X-Cache = %q, want revalidate", secondResponse.Header().Get("X-Cache"))
	}
}

func TestCombinedLessonEndpointUsesDefaultFENForOpeningRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "combined-lesson.json")
	body := []byte(`{
		"id": "7d75cc2b-a793-59ec-a71c-26fd9e172372",
		"title": "Tổng hợp toàn bộ lesson",
		"category": "Tổng hợp",
		"difficulty": "Tất cả",
		"lines": [
			{
				"id": "0c653267-670b-517f-9fe0-48323c52267a",
				"title": "Khai cuộc mặc định",
				"moves": []
			}
		],
		"choice": { "prompt": "", "options": [] }
	}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write combined lesson: %v", err)
	}
	t.Setenv("COMBINED_LESSON_FILE", path)

	request := httptest.NewRequest(http.MethodGet, "/api/combined-lesson", nil)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var payload struct {
		InitialFEN string `json:"initialFen"`
		Lines      []struct {
			Phase      string `json:"phase"`
			PieceCount int    `json:"pieceCount"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode combined lesson: %v", err)
	}
	if payload.InitialFEN != defaultXiangqiFEN {
		t.Fatalf("initialFen = %q, want default board FEN", payload.InitialFEN)
	}
	if len(payload.Lines) != 1 || payload.Lines[0].Phase != "opening" || payload.Lines[0].PieceCount != 32 {
		t.Fatalf("unexpected opening metadata: %#v", payload.Lines)
	}
}

func TestCombinedLessonLineMovesEndpointReturnsWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "combined-lesson.json")
	body := []byte(`{
		"id": "7d75cc2b-a793-59ec-a71c-26fd9e172372",
		"title": "Tổng hợp toàn bộ lesson",
		"category": "Tổng hợp",
		"difficulty": "Tất cả",
		"lines": [
			{
				"id": "0c653267-670b-517f-9fe0-48323c52267a",
				"title": "Line có FEN riêng",
				"moves": [
					{
						"id": "0563eb4a-8203-5b1b-8c6a-77eb05d5fd45",
						"side": "red",
						"from": { "file": 0, "rank": 6 },
						"to": { "file": 0, "rank": 5 },
						"comment": "first"
					},
					{
						"id": "8b9b7996-d567-5321-9fa2-07756d1fd8bb",
						"side": "black",
						"from": { "file": 0, "rank": 3 },
						"to": { "file": 0, "rank": 4 },
						"comment": "second"
					}
				]
			}
		],
		"choice": { "prompt": "", "options": [] }
	}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write combined lesson: %v", err)
	}
	t.Setenv("COMBINED_LESSON_FILE", path)

	request := httptest.NewRequest(http.MethodGet, "/api/combined-lesson/lines/0c653267-670b-517f-9fe0-48323c52267a/moves?from=1&limit=1", nil)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var payload struct {
		From       int `json:"from"`
		TotalMoves int `json:"totalMoves"`
		Moves      []struct {
			Comment string `json:"comment"`
		} `json:"moves"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode move window: %v", err)
	}
	if payload.From != 1 || payload.TotalMoves != 2 || len(payload.Moves) != 1 || payload.Moves[0].Comment != "second" {
		t.Fatalf("unexpected move window: %#v", payload)
	}
}

func TestCombinedLessonMovesEndpointReturnsStepForAllLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "combined-lesson.json")
	body := []byte(`{
		"id": "7d75cc2b-a793-59ec-a71c-26fd9e172372",
		"title": "Tổng hợp toàn bộ lesson",
		"category": "Tổng hợp",
		"difficulty": "Tất cả",
		"lines": [
			{
				"id": "0c653267-670b-517f-9fe0-48323c52267a",
				"title": "Line một",
				"moves": [
					{
						"id": "0563eb4a-8203-5b1b-8c6a-77eb05d5fd45",
						"side": "red",
						"from": { "file": 0, "rank": 6 },
						"to": { "file": 0, "rank": 5 },
						"comment": "line one first"
					},
					{
						"id": "8b9b7996-d567-5321-9fa2-07756d1fd8bb",
						"side": "black",
						"from": { "file": 0, "rank": 3 },
						"to": { "file": 0, "rank": 4 },
						"comment": "line one second"
					}
				]
			},
			{
				"id": "5cf7616d-83f6-56af-b02e-39ca0b526d2b",
				"title": "Line hai",
				"initialFen": "9/9/9/9/9/9/9/9/9/9 w - - 0 1",
				"moves": [
					{
						"id": "6aa750b8-9c52-55da-aa57-6d9f4db5bb8e",
						"side": "red",
						"from": { "file": 2, "rank": 6 },
						"to": { "file": 2, "rank": 5 },
						"comment": "line two first"
					}
				]
			}
		],
		"choice": { "prompt": "", "options": [] }
	}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write combined lesson: %v", err)
	}
	t.Setenv("COMBINED_LESSON_FILE", path)

	request := httptest.NewRequest(http.MethodGet, "/api/combined-lesson/moves?from=0&limit=1&phase=opening", nil)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var payload struct {
		From  int `json:"from"`
		Limit int `json:"limit"`
		Lines []struct {
			LineID     string `json:"lineId"`
			Phase      string `json:"phase"`
			PieceCount int    `json:"pieceCount"`
			TotalMoves int    `json:"totalMoves"`
			Moves      []struct {
				Comment string `json:"comment"`
			} `json:"moves"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode step window: %v", err)
	}
	if payload.From != 0 || payload.Limit != 1 || len(payload.Lines) != 1 {
		t.Fatalf("unexpected step metadata: %#v", payload)
	}
	if len(payload.Lines[0].Moves) != 1 || payload.Lines[0].Moves[0].Comment != "line one first" {
		t.Fatalf("unexpected first line step: %#v", payload.Lines[0])
	}
	if payload.Lines[0].Phase != "opening" || payload.Lines[0].PieceCount != 32 {
		t.Fatalf("unexpected first line phase metadata: %#v", payload.Lines[0])
	}
}

func TestCombinedLessonNextStepsEndpointReturnsActiveNodeWindow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "combined-lesson.json")
	body := []byte(`{
		"id": "7d75cc2b-a793-59ec-a71c-26fd9e172372",
		"title": "Tổng hợp toàn bộ lesson",
		"category": "Tổng hợp",
		"difficulty": "Tất cả",
		"lines": [
			{
				"id": "0c653267-670b-517f-9fe0-48323c52267a",
				"title": "Line một",
				"moves": [
					{
						"id": "0563eb4a-8203-5b1b-8c6a-77eb05d5fd45",
						"side": "red",
						"from": { "file": 0, "rank": 6 },
						"to": { "file": 0, "rank": 5 },
						"comment": "shared first"
					},
					{
						"id": "8b9b7996-d567-5321-9fa2-07756d1fd8bb",
						"side": "black",
						"from": { "file": 0, "rank": 3 },
						"to": { "file": 0, "rank": 4 },
						"comment": "line one second"
					}
				]
			},
			{
				"id": "5cf7616d-83f6-56af-b02e-39ca0b526d2b",
				"title": "Line hai",
				"moves": [
					{
						"id": "6aa750b8-9c52-55da-aa57-6d9f4db5bb8e",
						"side": "red",
						"from": { "file": 0, "rank": 6 },
						"to": { "file": 0, "rank": 5 },
						"comment": "shared first"
					},
					{
						"id": "d0e92d35-f4bd-5e31-8e55-09a0e0d2f9c7",
						"side": "black",
						"from": { "file": 2, "rank": 3 },
						"to": { "file": 2, "rank": 4 },
						"comment": "line two second"
					}
				]
			},
			{
				"id": "099e6161-1111-5000-8000-000000000001",
				"title": "Line khác prefix",
				"moves": [
					{
						"id": "099e6161-1111-5000-8000-000000000002",
						"side": "red",
						"from": { "file": 2, "rank": 6 },
						"to": { "file": 2, "rank": 5 },
						"comment": "different first"
					}
				]
			}
		],
		"choice": { "prompt": "", "options": [] }
	}`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write combined lesson: %v", err)
	}
	t.Setenv("COMBINED_LESSON_FILE", path)

	requestBody := bytes.NewBufferString(`{
		"phase": "opening",
		"initialFen": "` + defaultXiangqiFEN + `",
		"from": 1,
		"limit": 1,
		"prefix": [
			{
				"side": "red",
				"from": { "file": 0, "rank": 6 },
				"to": { "file": 0, "rank": 5 }
			}
		]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/combined-lesson/next-steps", requestBody)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var payload struct {
		From  int `json:"from"`
		Limit int `json:"limit"`
		Lines []struct {
			LineID string `json:"lineId"`
			Moves  []struct {
				Comment string `json:"comment"`
			} `json:"moves"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode next step window: %v", err)
	}
	if payload.From != 1 || payload.Limit != 1 || len(payload.Lines) != 2 {
		t.Fatalf("unexpected active node metadata: %#v", payload)
	}
	comments := []string{payload.Lines[0].Moves[0].Comment, payload.Lines[1].Moves[0].Comment}
	if !containsString(comments, "line one second") || !containsString(comments, "line two second") {
		t.Fatalf("unexpected active node moves: %#v", payload.Lines)
	}
}

func TestCombinedLessonEndpointReportsMissingArtifact(t *testing.T) {
	t.Setenv("COMBINED_LESSON_FILE", filepath.Join(t.TempDir(), "missing.json"))

	request := httptest.NewRequest(http.MethodGet, "/api/combined-lesson", nil)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
