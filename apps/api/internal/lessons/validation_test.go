package lessons

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

type testPiece struct {
	id   string
	side string
	kind string
	pos  Coordinate
}

var standardBoard = []testPiece{
	{"br1", "black", "chariot", Coordinate{File: 0, Rank: 0}},
	{"bh1", "black", "horse", Coordinate{File: 1, Rank: 0}},
	{"be1", "black", "elephant", Coordinate{File: 2, Rank: 0}},
	{"ba1", "black", "advisor", Coordinate{File: 3, Rank: 0}},
	{"bg", "black", "general", Coordinate{File: 4, Rank: 0}},
	{"ba2", "black", "advisor", Coordinate{File: 5, Rank: 0}},
	{"be2", "black", "elephant", Coordinate{File: 6, Rank: 0}},
	{"bh2", "black", "horse", Coordinate{File: 7, Rank: 0}},
	{"br2", "black", "chariot", Coordinate{File: 8, Rank: 0}},
	{"bc1", "black", "cannon", Coordinate{File: 1, Rank: 2}},
	{"bc2", "black", "cannon", Coordinate{File: 7, Rank: 2}},
	{"bs1", "black", "soldier", Coordinate{File: 0, Rank: 3}},
	{"bs2", "black", "soldier", Coordinate{File: 2, Rank: 3}},
	{"bs3", "black", "soldier", Coordinate{File: 4, Rank: 3}},
	{"bs4", "black", "soldier", Coordinate{File: 6, Rank: 3}},
	{"bs5", "black", "soldier", Coordinate{File: 8, Rank: 3}},
	{"rs1", "red", "soldier", Coordinate{File: 0, Rank: 6}},
	{"rs2", "red", "soldier", Coordinate{File: 2, Rank: 6}},
	{"rs3", "red", "soldier", Coordinate{File: 4, Rank: 6}},
	{"rs4", "red", "soldier", Coordinate{File: 6, Rank: 6}},
	{"rs5", "red", "soldier", Coordinate{File: 8, Rank: 6}},
	{"rc1", "red", "cannon", Coordinate{File: 1, Rank: 7}},
	{"rc2", "red", "cannon", Coordinate{File: 7, Rank: 7}},
	{"rr1", "red", "chariot", Coordinate{File: 0, Rank: 9}},
	{"rh1", "red", "horse", Coordinate{File: 1, Rank: 9}},
	{"re1", "red", "elephant", Coordinate{File: 2, Rank: 9}},
	{"ra1", "red", "advisor", Coordinate{File: 3, Rank: 9}},
	{"rg", "red", "general", Coordinate{File: 4, Rank: 9}},
	{"ra2", "red", "advisor", Coordinate{File: 5, Rank: 9}},
	{"re2", "red", "elephant", Coordinate{File: 6, Rank: 9}},
	{"rh2", "red", "horse", Coordinate{File: 7, Rank: 9}},
	{"rr2", "red", "chariot", Coordinate{File: 8, Rank: 9}},
}

var fenKinds = map[rune]string{
	'k': "general",
	'a': "advisor",
	'b': "elephant",
	'e': "elephant",
	'n': "horse",
	'h': "horse",
	'r': "chariot",
	'c': "cannon",
	'p': "soldier",
}

func TestLessonMovesReplayLegally(t *testing.T) {
	repository, err := LoadRepository(DefaultDataPath())
	if err != nil {
		t.Fatalf("load repository: %v", err)
	}

	for _, lesson := range repository.lessons {
		if len(lesson.Lines) == 0 {
			t.Fatalf("%s: expected at least one line", lesson.ID)
		}

		moveIDs := map[string]bool{}
		for _, line := range lesson.Lines {
			board, err := lessonInitialBoard(lesson)
			if err != nil {
				t.Fatalf("%s: initial position: %v", lesson.ID, err)
			}

			for index, move := range line.Moves {
				if move.ID == "" {
					t.Fatalf("%s/%s move %d: id is required", lesson.ID, line.ID, index+1)
				}
				moveIDs[move.ID] = true

				if err := validateMove(board, move); err != nil {
					t.Fatalf("%s/%s move %d %s: %v", lesson.ID, line.ID, index+1, moveCoordinates(move), err)
				}
				board = applyTestMove(board, move)
				if generalsFace(board) {
					t.Fatalf("%s/%s move %d %s: generals face each other", lesson.ID, line.ID, index+1, moveCoordinates(move))
				}
			}
		}

		for _, option := range lesson.Choice.Options {
			if !moveIDs[option.MoveID] {
				t.Fatalf("%s choice %q: move id %q does not exist", lesson.ID, option.Label, option.MoveID)
			}
		}
	}
}

func moveCoordinates(move Move) string {
	return fmt.Sprintf("(%d,%d)->(%d,%d)", move.From.File, move.From.Rank, move.To.File, move.To.Rank)
}

func lessonInitialBoard(lesson Lesson) ([]testPiece, error) {
	if strings.TrimSpace(lesson.InitialFEN) != "" {
		return boardFromFEN(lesson.InitialFEN)
	}
	board := make([]testPiece, len(standardBoard))
	copy(board, standardBoard)
	return board, nil
}

func boardFromFEN(fen string) ([]testPiece, error) {
	placement := strings.Fields(fen)
	if len(placement) == 0 {
		return nil, fmt.Errorf("empty FEN")
	}

	ranks := strings.Split(placement[0], "/")
	if len(ranks) != 10 {
		return nil, fmt.Errorf("expected 10 ranks, got %d", len(ranks))
	}

	counts := map[string]int{}
	board := []testPiece{}
	for rank, row := range ranks {
		file := 0
		for _, symbol := range row {
			if unicode.IsDigit(symbol) {
				file += int(symbol - '0')
				continue
			}

			kind, ok := fenKinds[unicode.ToLower(symbol)]
			if !ok {
				return nil, fmt.Errorf("unsupported FEN piece %q", symbol)
			}
			side := "black"
			if unicode.IsUpper(symbol) {
				side = "red"
			}
			countKey := side + "-" + kind
			counts[countKey]++
			board = append(board, testPiece{
				id:   fmt.Sprintf("%s-%s-%d", side[:1], kind, counts[countKey]),
				side: side,
				kind: kind,
				pos:  Coordinate{File: file, Rank: rank},
			})
			file++
		}
		if file != 9 {
			return nil, fmt.Errorf("rank %d has %d files", rank+1, file)
		}
	}
	return board, nil
}

func validateMove(board []testPiece, move Move) error {
	piece := testPieceAt(board, move.From)
	if piece == nil {
		return fmt.Errorf("source square is empty")
	}
	if piece.side != move.Side {
		return fmt.Errorf("source has %s piece, want %s", piece.side, move.Side)
	}
	if target := testPieceAt(board, move.To); target != nil && target.side == piece.side {
		return fmt.Errorf("target square contains own piece")
	}

	dx := move.To.File - move.From.File
	dy := move.To.Rank - move.From.Rank

	switch piece.kind {
	case "chariot":
		if !orthogonal(dx, dy) {
			return fmt.Errorf("chariot must move orthogonally")
		}
		if countScreens(board, move.From, move.To) != 0 {
			return fmt.Errorf("chariot path is blocked")
		}
	case "cannon":
		if !orthogonal(dx, dy) {
			return fmt.Errorf("cannon must move orthogonally")
		}
		screens := countScreens(board, move.From, move.To)
		if testPieceAt(board, move.To) == nil && screens != 0 {
			return fmt.Errorf("cannon non-capture has %d screens", screens)
		}
		if testPieceAt(board, move.To) != nil && screens != 1 {
			return fmt.Errorf("cannon capture has %d screens, want 1", screens)
		}
	case "horse":
		if !((abs(dx) == 1 && abs(dy) == 2) || (abs(dx) == 2 && abs(dy) == 1)) {
			return fmt.Errorf("horse has invalid shape")
		}
		leg := move.From
		if abs(dx) == 2 {
			leg.File += sign(dx)
		} else {
			leg.Rank += sign(dy)
		}
		if testPieceAt(board, leg) != nil {
			return fmt.Errorf("horse leg is blocked")
		}
	case "elephant":
		if abs(dx) != 2 || abs(dy) != 2 {
			return fmt.Errorf("elephant has invalid shape")
		}
		if (piece.side == "red" && move.To.Rank < 5) || (piece.side == "black" && move.To.Rank > 4) {
			return fmt.Errorf("elephant cannot cross the river")
		}
		if testPieceAt(board, Coordinate{File: move.From.File + dx/2, Rank: move.From.Rank + dy/2}) != nil {
			return fmt.Errorf("elephant eye is blocked")
		}
	case "advisor":
		if abs(dx) != 1 || abs(dy) != 1 {
			return fmt.Errorf("advisor has invalid shape")
		}
		if !insidePalace(piece.side, move.To) {
			return fmt.Errorf("advisor leaves palace")
		}
	case "general":
		if abs(dx)+abs(dy) != 1 {
			return fmt.Errorf("general has invalid shape")
		}
		if !insidePalace(piece.side, move.To) {
			return fmt.Errorf("general leaves palace")
		}
	case "soldier":
		forward := 1
		crossedRiver := move.From.Rank >= 5
		if piece.side == "red" {
			forward = -1
			crossedRiver = move.From.Rank <= 4
		}
		if !(dx == 0 && dy == forward) && !(crossedRiver && abs(dx) == 1 && dy == 0) {
			return fmt.Errorf("soldier has invalid shape")
		}
	default:
		return fmt.Errorf("unknown piece kind %q", piece.kind)
	}

	return nil
}

func applyTestMove(board []testPiece, move Move) []testPiece {
	moving := testPieceAt(board, move.From)
	next := make([]testPiece, 0, len(board))
	for _, piece := range board {
		if piece.id != moving.id && piece.pos == move.To {
			continue
		}
		if piece.id == moving.id {
			piece.pos = move.To
		}
		next = append(next, piece)
	}
	return next
}

func testPieceAt(board []testPiece, coordinate Coordinate) *testPiece {
	for index := range board {
		if board[index].pos == coordinate {
			return &board[index]
		}
	}
	return nil
}

func countScreens(board []testPiece, from Coordinate, to Coordinate) int {
	count := 0
	df := sign(to.File - from.File)
	dr := sign(to.Rank - from.Rank)
	for file, rank := from.File+df, from.Rank+dr; file != to.File || rank != to.Rank; file, rank = file+df, rank+dr {
		if testPieceAt(board, Coordinate{File: file, Rank: rank}) != nil {
			count++
		}
	}
	return count
}

func generalsFace(board []testPiece) bool {
	var redGeneral, blackGeneral *testPiece
	for index := range board {
		if board[index].kind != "general" {
			continue
		}
		if board[index].side == "red" {
			redGeneral = &board[index]
		} else {
			blackGeneral = &board[index]
		}
	}
	if redGeneral == nil || blackGeneral == nil || redGeneral.pos.File != blackGeneral.pos.File {
		return false
	}
	return countScreens(board, redGeneral.pos, blackGeneral.pos) == 0
}

func insidePalace(side string, coordinate Coordinate) bool {
	if coordinate.File < 3 || coordinate.File > 5 {
		return false
	}
	if side == "red" {
		return coordinate.Rank >= 7 && coordinate.Rank <= 9
	}
	return coordinate.Rank >= 0 && coordinate.Rank <= 2
}

func orthogonal(dx int, dy int) bool {
	return (dx == 0 && dy != 0) || (dx != 0 && dy == 0)
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func sign(value int) int {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return 0
}
