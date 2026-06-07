package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func writeOpeningBookFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "opening-book.json")
	body := []byte(`{
		"version": 1,
		"depth": 8,
		"source": "test-fixture",
		"stats": {"positions": 1, "edges": 2, "popular_edges": 2, "depth": 8, "lessons_processed": 1},
		"positions": {
			"rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1": {
				"moves": [
					{"move": "3,7->2,4", "notation": "Pháo 8 thoái 5", "name": "Pháo đầu", "frequency": 305, "popularity": 0.7439},
					{"move": "0,6->0,5", "notation": "Tốt 1 tiến 1", "name": "", "frequency": 61, "popularity": 0.1488}
				],
				"visits": 410
			}
		}
	}`)
	if err := writeFile(path, body, 0o644); err != nil {
		t.Fatalf("write opening book: %v", err)
	}
	return path
}

func TestOpeningEndpointReturnsPopularMoves(t *testing.T) {
	t.Setenv("OPENING_BOOK_FILE", writeOpeningBookFixture(t))

	request := httptest.NewRequest(http.MethodGet, "/api/opening?fen="+urlEncode("rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"), nil)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Available bool `json:"available"`
		FEN       string `json:"fen"`
		Visits    int    `json:"visits"`
		Moves     []struct {
			Notation   string  `json:"notation"`
			Name       string  `json:"name"`
			Frequency  int     `json:"frequency"`
			Popularity float64 `json:"popularity"`
		} `json:"moves"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.Available {
		t.Errorf("expected available=true")
	}
	if payload.Visits != 410 {
		t.Errorf("visits = %d, want 410", payload.Visits)
	}
	if len(payload.Moves) != 2 {
		t.Fatalf("moves count = %d, want 2", len(payload.Moves))
	}
	if payload.Moves[0].Notation != "Pháo 8 thoái 5" {
		t.Errorf("top move = %q, want Pháo 8 thoái 5", payload.Moves[0].Notation)
	}
	if payload.Moves[0].Name != "Pháo đầu" {
		t.Errorf("top move name = %q, want Pháo đầu", payload.Moves[0].Name)
	}
}

func TestOpeningEndpointReturnsEmptyForUnknownPosition(t *testing.T) {
	t.Setenv("OPENING_BOOK_FILE", writeOpeningBookFixture(t))

	request := httptest.NewRequest(http.MethodGet, "/api/opening?fen=9/9/9/9/9/9/9/9/9/9+w+-+0+1", nil)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}

	var payload struct {
		Moves []any `json:"moves"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Moves) != 0 {
		t.Errorf("expected empty moves, got %d", len(payload.Moves))
	}
}

func TestOpeningEndpointReturns304OnMatchingETag(t *testing.T) {
	t.Setenv("OPENING_BOOK_FILE", writeOpeningBookFixture(t))

	fen := "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1"
	first := httptest.NewRequest(http.MethodGet, "/api/opening?fen="+urlEncode(fen), nil)
	firstResponse := httptest.NewRecorder()
	NewServer(nil).ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", firstResponse.Code)
	}
	etag := firstResponse.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header on first response")
	}

	second := httptest.NewRequest(http.MethodGet, "/api/opening?fen="+urlEncode(fen), nil)
	second.Header.Set("If-None-Match", etag)
	secondResponse := httptest.NewRecorder()
	NewServer(nil).ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusNotModified {
		t.Fatalf("second status = %d, want 304", secondResponse.Code)
	}
}

func TestOpeningEndpointRejectsMissingFEN(t *testing.T) {
	t.Setenv("OPENING_BOOK_FILE", writeOpeningBookFixture(t))
	request := httptest.NewRequest(http.MethodGet, "/api/opening", nil)
	response := httptest.NewRecorder()
	NewServer(nil).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}
