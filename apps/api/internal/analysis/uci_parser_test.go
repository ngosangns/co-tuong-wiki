package analysis

import "testing"

func TestParseUCILineCapturesSearchInfo(t *testing.T) {
	result := uciSearchResult{}

	parseUCILine(&result, "info depth 9 score cp 123 pv h2e2 h9g7")
	parseUCILine(&result, "bestmove h2e2")

	if result.Depth != 9 {
		t.Fatalf("depth = %d, want 9", result.Depth)
	}
	if result.ScoreCP != 123 {
		t.Fatalf("score = %d, want 123", result.ScoreCP)
	}
	if result.BestMove != "h2e2" {
		t.Fatalf("bestmove = %q, want h2e2", result.BestMove)
	}
	if len(result.PV) != 2 || result.PV[0] != "h2e2" || result.PV[1] != "h9g7" {
		t.Fatalf("pv = %#v, want h2e2 h9g7", result.PV)
	}
}

func TestParseUCILineConvertsMateScore(t *testing.T) {
	result := uciSearchResult{}

	parseUCILine(&result, "info depth 4 score mate -2 pv a0a1")

	if result.ScoreCP >= -29900 {
		t.Fatalf("mate score = %d, want large negative cp", result.ScoreCP)
	}
}
