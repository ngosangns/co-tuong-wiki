package lessons

import "testing"

func TestLoadRepository(t *testing.T) {
	repository, err := LoadRepository(DefaultDataPath())
	if err != nil {
		t.Fatalf("load repository: %v", err)
	}

	if got := len(repository.List("", "", "", "")); got != 20 {
		t.Fatalf("expected 20 lessons, got %d", got)
	}

	if _, ok := repository.Get("central-cannon-tempo-trap"); !ok {
		t.Fatal("expected central-cannon-tempo-trap lesson")
	}
}

func TestLessonMoveGraphDataUsesSharedPrefixIDs(t *testing.T) {
	repository, err := LoadRepository(DefaultDataPath())
	if err != nil {
		t.Fatalf("load repository: %v", err)
	}

	for _, lesson := range repository.lessons {
		if len(lesson.Lines) < 2 {
			continue
		}

		sharedPrefixLength := sharedMoveContentPrefixLength(lesson.Lines)
		if sharedPrefixLength == 0 {
			t.Fatalf("%s: expected at least one shared move before branches", lesson.ID)
		}

		for lineIndex, line := range lesson.Lines[1:] {
			for moveIndex := 0; moveIndex < sharedPrefixLength; moveIndex++ {
				wantID := lesson.Lines[0].Moves[moveIndex].ID
				if gotID := line.Moves[moveIndex].ID; gotID != wantID {
					t.Fatalf(
						"%s line %d move %d: shared graph move id = %q, want %q",
						lesson.ID,
						lineIndex+1,
						moveIndex,
						gotID,
						wantID,
					)
				}
			}
		}

		for _, option := range lesson.Choice.Options {
			lineID, moveIndex, ok := findMove(lesson.Lines, option.MoveID)
			if !ok {
				t.Fatalf("%s choice %s: move id %q does not exist", lesson.ID, option.Label, option.MoveID)
			}
			if moveIndex < sharedPrefixLength {
				t.Fatalf(
					"%s choice %s: move id %q points to shared prefix move %d in line %s",
					lesson.ID,
					option.Label,
					option.MoveID,
					moveIndex,
					lineID,
				)
			}
		}
	}
}

func sharedMoveContentPrefixLength(lines []Line) int {
	prefixLength := len(lines[0].Moves)
	for _, line := range lines[1:] {
		index := 0
		for index < prefixLength && index < len(line.Moves) && sameGraphMove(lines[0].Moves[index], line.Moves[index]) {
			index++
		}
		prefixLength = index
	}
	return prefixLength
}

func sameGraphMove(a Move, b Move) bool {
	return a.Side == b.Side &&
		a.From == b.From &&
		a.To == b.To &&
		a.Notation == b.Notation &&
		a.Title == b.Title
}

func findMove(lines []Line, moveID string) (string, int, bool) {
	for _, line := range lines {
		for index, move := range line.Moves {
			if move.ID == moveID {
				return line.ID, index, true
			}
		}
	}
	return "", 0, false
}
