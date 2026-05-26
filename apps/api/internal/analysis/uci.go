package analysis

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type UCIAnalyzer struct {
	config Config
}

func NewUCIAnalyzer(config Config) UCIAnalyzer {
	return UCIAnalyzer{config: config}
}

func (analyzer UCIAnalyzer) Analyze(ctx context.Context, request Request) (Response, error) {
	command := exec.CommandContext(ctx, analyzer.config.Path)
	stdin, err := command.StdinPipe()
	if err != nil {
		return Response{}, fmt.Errorf("open engine stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return Response{}, fmt.Errorf("open engine stdout: %w", err)
	}

	var stderr bytes.Buffer
	command.Stderr = &stderr

	if err := command.Start(); err != nil {
		return Response{}, fmt.Errorf("start UCI engine: %w", err)
	}

	result, runErr := analyzer.runSearch(ctx, stdin, stdout, request)
	closeErr := stdin.Close()
	if runErr != nil && command.Process != nil {
		_ = command.Process.Kill()
	}
	waitErr := command.Wait()
	if runErr != nil {
		return Response{}, withEngineStderr(runErr, stderr.String())
	}
	if closeErr != nil && !errors.Is(closeErr, io.ErrClosedPipe) {
		return Response{}, fmt.Errorf("close engine stdin: %w", closeErr)
	}
	if waitErr != nil {
		return Response{}, withEngineStderr(fmt.Errorf("wait for engine: %w", waitErr), stderr.String())
	}

	return analyzer.responseFromResult(request, result)
}

func (analyzer UCIAnalyzer) runSearch(ctx context.Context, stdin io.Writer, stdout io.Reader, request Request) (uciSearchResult, error) {
	lines := make(chan string)
	errs := make(chan error, 1)

	go scanLines(stdout, lines, errs)

	if err := writeEngineCommand(stdin, "uci"); err != nil {
		return uciSearchResult{}, err
	}
	if err := waitForLine(ctx, lines, errs, func(line string) bool { return line == "uciok" }); err != nil {
		return uciSearchResult{}, fmt.Errorf("initialize UCI engine: %w", err)
	}

	if err := writeEngineCommand(stdin, "isready"); err != nil {
		return uciSearchResult{}, err
	}
	if err := waitForLine(ctx, lines, errs, func(line string) bool { return line == "readyok" }); err != nil {
		return uciSearchResult{}, fmt.Errorf("wait for UCI engine readiness: %w", err)
	}

	if err := writeEngineCommand(stdin, "ucinewgame"); err != nil {
		return uciSearchResult{}, err
	}
	if err := writeEngineCommand(stdin, "position fen "+request.FEN); err != nil {
		return uciSearchResult{}, err
	}
	if err := writeEngineCommand(stdin, analyzer.searchCommand(request)); err != nil {
		return uciSearchResult{}, err
	}

	result := uciSearchResult{}
	for {
		select {
		case <-ctx.Done():
			return uciSearchResult{}, engineContextError(ctx.Err())
		case err := <-errs:
			return uciSearchResult{}, err
		case line, ok := <-lines:
			if !ok {
				return uciSearchResult{}, errors.New("engine exited before bestmove")
			}
			parseUCILine(&result, line)
			if result.BestMove != "" {
				if err := writeEngineCommand(stdin, "quit"); err != nil {
					return uciSearchResult{}, err
				}
				return result, nil
			}
		}
	}
}

func (analyzer UCIAnalyzer) searchCommand(request Request) string {
	depth := clampPositive(request.Depth, analyzer.config.DefaultDepth, analyzer.config.MaxDepth)
	if request.Depth > 0 {
		return fmt.Sprintf("go depth %d", depth)
	}

	timeMS := clampPositive(request.TimeMS, analyzer.config.DefaultTimeMS, analyzer.config.MaxTimeMS)
	return fmt.Sprintf("go movetime %d", timeMS)
}

func (analyzer UCIAnalyzer) responseFromResult(request Request, result uciSearchResult) (Response, error) {
	redScore := normalizeScoreForRed(result.ScoreCP, request.SideToMove)
	bestMove := moveFromNotation(result.BestMove, redScore)
	if bestMove == nil {
		return Response{}, errors.New("engine returned an unsupported bestmove")
	}

	pv := make([]Move, 0, len(result.PV))
	for index, notation := range result.PV {
		if index > 0 {
			if move := moveFromNotationWithoutScore(notation); move != nil {
				pv = append(pv, *move)
			}
			continue
		}
		if move := moveFromNotation(notation, redScore); move != nil {
			pv = append(pv, *move)
		}
	}
	if len(pv) == 0 {
		pv = append(pv, *bestMove)
	}

	return Response{
		FEN:                request.FEN,
		SideToMove:         request.SideToMove,
		Score:              Score{CP: redScore, Perspective: "red"},
		BestMove:           bestMove,
		PrincipalVariation: pv,
		Depth:              result.Depth,
		Source:             "uci",
		Message:            "Đánh giá bằng engine UCI được cấu hình ở backend.",
	}, nil
}

func scanLines(reader io.Reader, lines chan<- string, errs chan<- error) {
	defer close(lines)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lines <- strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		errs <- err
	}
}

func waitForLine(ctx context.Context, lines <-chan string, errs <-chan error, match func(string) bool) error {
	for {
		select {
		case <-ctx.Done():
			return engineContextError(ctx.Err())
		case err := <-errs:
			return err
		case line, ok := <-lines:
			if !ok {
				return errors.New("engine output closed")
			}
			if match(line) {
				return nil
			}
		}
	}
}

func engineContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		// Convert low-level context deadlines into the engine error contract used by callers.
		return fmt.Errorf("%w: %w", ErrEngineTimeout, err)
	}
	return err
}

func writeEngineCommand(writer io.Writer, command string) error {
	_, err := fmt.Fprintln(writer, command)
	if err != nil {
		return fmt.Errorf("write UCI command %q: %w", command, err)
	}
	return nil
}

func moveFromNotation(notation string, score int) *Move {
	move := moveFromNotationWithoutScore(notation)
	if move == nil {
		return nil
	}
	move.Score = &score
	return move
}

func moveFromNotationWithoutScore(notation string) *Move {
	if len(notation) < 4 {
		return nil
	}
	from, ok := coordinateFromEngineSquare(notation[:2])
	if !ok {
		return nil
	}
	to, ok := coordinateFromEngineSquare(notation[2:4])
	if !ok {
		return nil
	}
	return &Move{From: from, To: to, Notation: notation[:4]}
}

func coordinateFromEngineSquare(square string) (Coordinate, bool) {
	files := "abcdefghi"
	if len(square) != 2 {
		return Coordinate{}, false
	}
	file := strings.IndexByte(files, square[0])
	rank := int(square[1] - '0')
	if file < 0 || rank < 0 || rank > 9 {
		return Coordinate{}, false
	}
	return Coordinate{File: file, Rank: 9 - rank}, true
}

func normalizeScoreForRed(score int, sideToMove string) int {
	if sideToMove == "black" {
		return -score
	}
	return score
}

func clampPositive(value int, fallback int, max int) int {
	if value <= 0 {
		value = fallback
	}
	if value > max {
		return max
	}
	return value
}

func withEngineStderr(err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, stderr)
}
