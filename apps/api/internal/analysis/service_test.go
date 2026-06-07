package analysis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

func TestConfigFromEnvUsesInteractiveEngineTimeWindow(t *testing.T) {
	t.Setenv("ENGINE_KIND", "")
	t.Setenv("ENGINE_PATH", "")
	t.Setenv("ENGINE_DEFAULT_TIME_MS", "")
	t.Setenv("ENGINE_MAX_TIME_MS", "")

	config := ConfigFromEnv()
	if config.DefaultTimeMS != 500 {
		t.Fatalf("DefaultTimeMS = %d, want 500", config.DefaultTimeMS)
	}
	if config.MaxTimeMS != 10000 {
		t.Fatalf("MaxTimeMS = %d, want 10000", config.MaxTimeMS)
	}
	if config.TimeoutSlack != 3*time.Second {
		t.Fatalf("TimeoutSlack = %s, want 3s", config.TimeoutSlack)
	}
}

func TestAnalyzeUsesUCIEngineOutput(t *testing.T) {
	t.Cleanup(func() { sharedEngines.close() })
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

func TestAnalyzeCachesRepeatedPosition(t *testing.T) {
	t.Cleanup(func() { sharedEngines.close() })
	countPath := filepath.Join(t.TempDir(), "go-count.txt")
	enginePath := writeCountingUCIEngine(t, countPath)
	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", enginePath)

	request := Request{
		FEN:        "9/9/9/9/9/9/9/9/9/9 w - - 0 1",
		SideToMove: "red",
		TimeMS:     20,
	}
	if _, err := Analyze(context.Background(), request); err != nil {
		t.Fatalf("first Analyze returned error: %v", err)
	}
	if _, err := Analyze(context.Background(), request); err != nil {
		t.Fatalf("second Analyze returned error: %v", err)
	}

	count, err := os.ReadFile(countPath)
	if err != nil {
		t.Fatalf("read go count: %v", err)
	}
	if string(count) != "1\n" {
		t.Fatalf("go count = %q, want one engine search", string(count))
	}
}

func TestAnalyzeCoalescesConcurrentPosition(t *testing.T) {
	t.Cleanup(func() { sharedEngines.close() })
	countPath := filepath.Join(t.TempDir(), "go-count.txt")
	enginePath := writeDelayedCountingUCIEngine(t, countPath)
	t.Setenv("ENGINE_KIND", "uci")
	t.Setenv("ENGINE_PATH", enginePath)

	request := Request{
		FEN:        "9/9/9/9/9/9/9/9/9/9 w - - 0 1",
		SideToMove: "red",
		TimeMS:     1500,
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	statuses := make(chan string, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, meta, err := AnalyzeWithMeta(context.Background(), request)
			statuses <- meta.CacheStatus
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	close(statuses)

	for err := range errs {
		if err != nil {
			t.Fatalf("AnalyzeWithMeta returned error: %v", err)
		}
	}

	count, err := os.ReadFile(countPath)
	if err != nil {
		t.Fatalf("read go count: %v", err)
	}
	if string(count) != "1\n" {
		t.Fatalf("go count = %q, want one coalesced engine search", string(count))
	}

	coalesced := false
	for status := range statuses {
		if status == "coalesced" {
			coalesced = true
		}
	}
	if !coalesced {
		t.Fatal("expected one concurrent AnalyzeWithMeta call to report coalesced")
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

func writeCountingUCIEngine(t *testing.T, countPath string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "counting-uci.sh")
	script := fmt.Sprintf(`#!/bin/sh
count_file=%q
while IFS= read -r line; do
  case "$line" in
    uci)
      echo "id name counting-uci"
      echo "uciok"
      ;;
    isready)
      echo "readyok"
      ;;
    go*)
      count="$(cat "$count_file" 2>/dev/null || echo 0)"
      count=$((count + 1))
      echo "$count" > "$count_file"
      echo "info depth 7 score cp 86 pv h2e2 h9g7"
      echo "bestmove h2e2"
      ;;
    quit)
      exit 0
      ;;
  esac
done
`, countPath)

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write counting engine: %v", err)
	}
	return path
}

func writeDelayedCountingUCIEngine(t *testing.T, countPath string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "delayed-counting-uci.sh")
	script := fmt.Sprintf(`#!/bin/sh
count_file=%q
while IFS= read -r line; do
  case "$line" in
    uci)
      echo "id name delayed-counting-uci"
      echo "uciok"
      ;;
    isready)
      echo "readyok"
      ;;
    go*)
      count="$(cat "$count_file" 2>/dev/null || echo 0)"
      count=$((count + 1))
      echo "$count" > "$count_file"
      sleep 1
      echo "info depth 7 score cp 86 pv h2e2 h9g7"
      echo "bestmove h2e2"
      ;;
    quit)
      exit 0
      ;;
  esac
done
`, countPath)

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write delayed counting engine: %v", err)
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
