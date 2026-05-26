package analysis

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAnalyzeRequiresConfiguredUCIEngine(t *testing.T) {
	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", "")

	_, err := Analyze(context.Background(), Request{FEN: "9/9/9/9/9/9/9/9/9/9 w - - 0 1", SideToMove: "red"})
	if !errors.Is(err, ErrEngineUnavailable) {
		t.Fatalf("error = %v, want ErrEngineUnavailable", err)
	}
}

func TestConfigFromEnvUsesLargerEngineTimeWindow(t *testing.T) {
	t.Setenv("ENGINE_KIND", "")
	t.Setenv("ENGINE_PATH", "")
	t.Setenv("ENGINE_DEFAULT_TIME_MS", "")
	t.Setenv("ENGINE_MAX_TIME_MS", "")

	config := ConfigFromEnv()
	if config.DefaultTimeMS != 2000 {
		t.Fatalf("DefaultTimeMS = %d, want 2000", config.DefaultTimeMS)
	}
	if config.MaxTimeMS != 10000 {
		t.Fatalf("MaxTimeMS = %d, want 10000", config.MaxTimeMS)
	}
	if config.TimeoutSlack != 3*time.Second {
		t.Fatalf("TimeoutSlack = %s, want 3s", config.TimeoutSlack)
	}
}

func TestAnalyzeUsesUCIEngineOutput(t *testing.T) {
	enginePath := writeFakeUCIEngine(t)
	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", enginePath)

	response, err := Analyze(context.Background(), Request{
		FEN:        "9/9/9/9/9/9/9/9/9/9 w - - 0 1",
		SideToMove: "black",
		TimeMS:     20,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if response.Source != "uci" {
		t.Fatalf("source = %q, want uci", response.Source)
	}
	if response.Depth != 7 {
		t.Fatalf("depth = %d, want 7", response.Depth)
	}
	if response.Score.CP != -86 {
		t.Fatalf("score = %d, want -86 from red perspective when black is to move", response.Score.CP)
	}
	if response.BestMove == nil || response.BestMove.Notation != "h2e2" {
		t.Fatalf("bestMove = %#v, want h2e2", response.BestMove)
	}
	if len(response.PrincipalVariation) != 2 {
		t.Fatalf("pv length = %d, want 2", len(response.PrincipalVariation))
	}
}

func TestUCIAnalyzerReturnsEngineTimeout(t *testing.T) {
	enginePath := writeSlowUCIEngine(t)
	analyzer := NewUCIAnalyzer(Config{Path: enginePath})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := analyzer.Analyze(ctx, Request{
		FEN:        "9/9/9/9/9/9/9/9/9/9 w - - 0 1",
		SideToMove: "red",
		TimeMS:     1,
	})
	if !errors.Is(err, ErrEngineTimeout) {
		t.Fatalf("error = %v, want ErrEngineTimeout", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want wrapped context deadline", err)
	}
}

func writeFakeUCIEngine(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fake-uci.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    uci)
      echo "id name fake-uci"
      echo "uciok"
      ;;
    isready)
      echo "readyok"
      ;;
    go*)
      echo "info depth 7 score cp 86 pv h2e2 h9g7"
      echo "bestmove h2e2"
      ;;
    quit)
      exit 0
      ;;
  esac
done
`

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake engine: %v", err)
	}
	return path
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
      sleep 1
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
