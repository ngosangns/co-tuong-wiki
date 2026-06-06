package analysis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type PersistentUCIAnalyzer struct {
	config Config
	mu     sync.Mutex
	engine *uciEngineProcess
}

type uciEngineProcess struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	lines   chan string
	errs    chan error
	stderr  *bytes.Buffer
}

func NewPersistentUCIAnalyzer(config Config) *PersistentUCIAnalyzer {
	return &PersistentUCIAnalyzer{config: config}
}

func (analyzer *PersistentUCIAnalyzer) Analyze(ctx context.Context, request Request) (Response, error) {
	analyzer.mu.Lock()
	defer analyzer.mu.Unlock()

	engine, err := analyzer.ensureEngine(ctx)
	if err != nil {
		return Response{}, err
	}

	result, err := analyzer.runSearch(ctx, engine, request)
	if err != nil {
		// A timed-out or protocol-broken UCI process cannot be trusted for the next request.
		if ctx.Err() != nil {
			analyzer.killLocked()
		} else {
			analyzer.stopLocked()
		}
		return Response{}, withEngineStderr(err, engine.stderr.String())
	}

	return UCIAnalyzer{config: analyzer.config}.responseFromResult(request, result)
}

func (analyzer *PersistentUCIAnalyzer) Close() error {
	analyzer.mu.Lock()
	defer analyzer.mu.Unlock()

	return analyzer.stopLocked()
}

func (analyzer *PersistentUCIAnalyzer) ensureEngine(ctx context.Context) (*uciEngineProcess, error) {
	if analyzer.engine != nil {
		return analyzer.engine, nil
	}

	command := exec.Command(analyzer.config.Path)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open engine stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open engine stdout: %w", err)
	}

	stderr := &bytes.Buffer{}
	command.Stderr = stderr

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start UCI engine: %w", err)
	}

	engine := &uciEngineProcess{
		command: command,
		stdin:   stdin,
		lines:   make(chan string),
		errs:    make(chan error, 1),
		stderr:  stderr,
	}
	go scanLines(stdout, engine.lines, engine.errs)

	if err := analyzer.initializeEngine(ctx, engine); err != nil {
		_ = stopEngineProcess(engine)
		return nil, withEngineStderr(err, stderr.String())
	}

	analyzer.engine = engine
	return engine, nil
}

func (analyzer *PersistentUCIAnalyzer) initializeEngine(ctx context.Context, engine *uciEngineProcess) error {
	if err := writeEngineCommand(engine.stdin, "uci"); err != nil {
		return err
	}
	if err := waitForLine(ctx, engine.lines, engine.errs, func(line string) bool { return line == "uciok" }); err != nil {
		return fmt.Errorf("initialize UCI engine: %w", err)
	}
	if err := writeEngineCommand(engine.stdin, "isready"); err != nil {
		return err
	}
	if err := waitForLine(ctx, engine.lines, engine.errs, func(line string) bool { return line == "readyok" }); err != nil {
		return fmt.Errorf("wait for UCI engine readiness: %w", err)
	}
	return nil
}

func (analyzer *PersistentUCIAnalyzer) runSearch(ctx context.Context, engine *uciEngineProcess, request Request) (uciSearchResult, error) {
	if err := writeEngineCommand(engine.stdin, "ucinewgame"); err != nil {
		return uciSearchResult{}, err
	}
	if err := writeEngineCommand(engine.stdin, "position fen "+request.FEN); err != nil {
		return uciSearchResult{}, err
	}
	if err := writeEngineCommand(engine.stdin, UCIAnalyzer{config: analyzer.config}.searchCommand(request)); err != nil {
		return uciSearchResult{}, err
	}

	result := uciSearchResult{}
	for {
		select {
		case <-ctx.Done():
			return uciSearchResult{}, engineContextError(ctx.Err())
		case err := <-engine.errs:
			return uciSearchResult{}, err
		case line, ok := <-engine.lines:
			if !ok {
				return uciSearchResult{}, errors.New("engine exited before bestmove")
			}
			parseUCILine(&result, line)
			if result.BestMove != "" {
				return result, nil
			}
		}
	}
}

func (analyzer *PersistentUCIAnalyzer) stopLocked() error {
	if analyzer.engine == nil {
		return nil
	}
	err := stopEngineProcess(analyzer.engine)
	analyzer.engine = nil
	return err
}

func (analyzer *PersistentUCIAnalyzer) killLocked() error {
	if analyzer.engine == nil {
		return nil
	}
	err := killEngineProcess(analyzer.engine)
	analyzer.engine = nil
	return err
}

func stopEngineProcess(engine *uciEngineProcess) error {
	if engine == nil || engine.command == nil {
		return nil
	}

	_ = writeEngineCommand(engine.stdin, "quit")
	closeErr := engine.stdin.Close()
	waitErr := engine.command.Wait()
	if closeErr != nil && !errors.Is(closeErr, io.ErrClosedPipe) {
		return fmt.Errorf("close engine stdin: %w", closeErr)
	}
	if waitErr != nil {
		return withEngineStderr(fmt.Errorf("wait for engine: %w", waitErr), engine.stderr.String())
	}
	return nil
}

func killEngineProcess(engine *uciEngineProcess) error {
	if engine == nil || engine.command == nil {
		return nil
	}

	if engine.command.Process != nil {
		_ = engine.command.Process.Kill()
	}
	_ = engine.stdin.Close()
	if err := engine.command.Wait(); err != nil {
		return withEngineStderr(fmt.Errorf("wait for engine: %w", err), engine.stderr.String())
	}
	return nil
}
