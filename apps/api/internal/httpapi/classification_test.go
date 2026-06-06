package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassifyMoveThresholds(t *testing.T) {
	cases := []struct {
		cpLoss int
		want   string
	}{
		{0, string(classificationBest)},
		{10, string(classificationBest)},
		{11, string(classificationExcellent)},
		{30, string(classificationExcellent)},
		{31, string(classificationGood)},
		{80, string(classificationGood)},
		{81, string(classificationInaccuracy)},
		{150, string(classificationInaccuracy)},
		{151, string(classificationMistake)},
		{300, string(classificationMistake)},
		{301, string(classificationBlunder)},
		{1500, string(classificationBlunder)},
	}
	for _, c := range cases {
		got := classifyMove(c.cpLoss)
		if got != moveClassification(c.want) {
			t.Errorf("classifyMove(%d) = %q, want %q", c.cpLoss, got, c.want)
		}
	}
}

func TestCPLossIsAbsolute(t *testing.T) {
	// Both directions of the gap should produce the same magnitude.
	if got := cpLoss(100, 50); got != 50 {
		t.Errorf("cpLoss(100, 50) = %d, want 50", got)
	}
	if got := cpLoss(50, 100); got != 50 {
		t.Errorf("cpLoss(50, 100) = %d, want 50", got)
	}
	if got := cpLoss(0, 0); got != 0 {
		t.Errorf("cpLoss(0, 0) = %d, want 0", got)
	}
}

func TestLineEvaluationIncludesClassificationForPlayedMoves(t *testing.T) {
	enginePath := writeStubUCIEngine(t, "pikafish")
	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", enginePath)
	t.Setenv("ENGINE_FAIRY_PATH", "")
	t.Setenv("ENGINE_DEFAULT_TIME_MS", "5")
	t.Setenv("ENGINE_MAX_TIME_MS", "5")

	body := bytes.NewBufferString(`{
		"plies": [
			{ "ply": 0, "fen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/RNBAKABNR w - - 0 1", "sideToMove": "red" },
			{ "ply": 1, "fen": "rnbakabnr/9/1c5c1/p1p1p1p1p/9/9/P1P1P1P1P/1C5C1/9/R1BAKABNR b - - 0 1", "sideToMove": "black" }
		]
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/line-evaluation", body)
	response := httptest.NewRecorder()

	NewServer(nil).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Plies []struct {
			Ply            int    `json:"ply"`
			CpLoss         *int   `json:"cpLoss"`
			Classification string `json:"classification"`
			Status         string `json:"status"`
		} `json:"plies"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Plies) != 2 {
		t.Fatalf("plies count = %d, want 2", len(payload.Plies))
	}
	// Ply 0 has no preceding move, so no classification.
	if payload.Plies[0].CpLoss != nil {
		t.Errorf("ply 0 should not have cpLoss, got %v", *payload.Plies[0].CpLoss)
	}
	// Ply 1 corresponds to the move played at ply 0; should have a classification.
	if payload.Plies[1].CpLoss == nil {
		t.Fatalf("ply 1 should have cpLoss, got nil")
	}
	if payload.Plies[1].Classification == "" {
		t.Errorf("ply 1 should have classification, got empty")
	}
}
