package lessons

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestLoadRepository(t *testing.T) {
	repository, err := LoadRepository(DefaultDataPath())
	if err != nil {
		t.Fatalf("load repository: %v", err)
	}

	if got := len(repository.List("", "", "")); got != 811 {
		t.Fatalf("expected 811 lessons, got %d", got)
	}

	if !hasLessonTitle(repository.lessons, "Pháo Đầu đối Bình Phong Mã - ham quân mất nhịp") {
		t.Fatal("expected central cannon tempo trap lesson")
	}
}

func TestLessonIDsUseUUID(t *testing.T) {
	repository, err := LoadRepository(DefaultDataPath())
	if err != nil {
		t.Fatalf("load repository: %v", err)
	}

	lineIDs := map[string]bool{}
	for _, lesson := range repository.lessons {
		if !uuidPattern.MatchString(lesson.ID) {
			t.Fatalf("%s: lesson id must be a UUID", lesson.ID)
		}
		for _, line := range lesson.Lines {
			if !uuidPattern.MatchString(line.ID) {
				t.Fatalf("%s/%s: line id must be a UUID", lesson.ID, line.ID)
			}
			if lineIDs[line.ID] {
				t.Fatalf("%s/%s: duplicate line id", lesson.ID, line.ID)
			}
			lineIDs[line.ID] = true

			for _, move := range line.Moves {
				if !uuidPattern.MatchString(move.ID) {
					t.Fatalf("%s/%s/%s: move id must be a UUID", lesson.ID, line.ID, move.ID)
				}
			}
		}
	}
}

func TestLessonTitlesDoNotIncludeLessonNumberPrefix(t *testing.T) {
	repository, err := LoadRepository(DefaultDataPath())
	if err != nil {
		t.Fatalf("load repository: %v", err)
	}

	prefixPattern := regexp.MustCompile(`^Bài \d{3}:`)
	for _, lesson := range repository.lessons {
		if prefixPattern.MatchString(lesson.Title) {
			t.Fatalf("%s: lesson title should not include lesson number prefix: %q", lesson.ID, lesson.Title)
		}
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

func TestLessonLinePayloadsDoNotUseDescriptionField(t *testing.T) {
	data, err := os.ReadFile(DefaultDataPath())
	if err != nil {
		t.Fatalf("read lessons data: %v", err)
	}

	var lessons []struct {
		ID    string                       `json:"id"`
		Lines []map[string]json.RawMessage `json:"lines"`
	}
	if err := json.Unmarshal(data, &lessons); err != nil {
		t.Fatalf("decode lessons data: %v", err)
	}

	for _, lesson := range lessons {
		for _, line := range lesson.Lines {
			if _, exists := line["description"]; exists {
				t.Fatalf("%s/%s: line payload should not include description", lesson.ID, string(line["id"]))
			}
		}
	}
}

func TestLessonPayloadsDoNotUseTagsField(t *testing.T) {
	data, err := os.ReadFile(DefaultDataPath())
	if err != nil {
		t.Fatalf("read lessons data: %v", err)
	}

	var lessons []map[string]json.RawMessage
	if err := json.Unmarshal(data, &lessons); err != nil {
		t.Fatalf("decode lessons data: %v", err)
	}

	for _, lesson := range lessons {
		if _, exists := lesson["tags"]; exists {
			t.Fatalf("%s: lesson payload should not include tags", string(lesson["id"]))
		}
	}
}

func TestLessonPayloadsDoNotUseSummaryOrPrinciplesFields(t *testing.T) {
	data, err := os.ReadFile(DefaultDataPath())
	if err != nil {
		t.Fatalf("read lessons data: %v", err)
	}

	var lessons []map[string]json.RawMessage
	if err := json.Unmarshal(data, &lessons); err != nil {
		t.Fatalf("decode lessons data: %v", err)
	}

	for _, lesson := range lessons {
		if _, exists := lesson["summary"]; exists {
			t.Fatalf("%s: lesson payload should not include summary", string(lesson["id"]))
		}
		if _, exists := lesson["principles"]; exists {
			t.Fatalf("%s: lesson payload should not include principles", string(lesson["id"]))
		}
	}
}

func TestLessonMovePayloadsDoNotUseNotationField(t *testing.T) {
	data, err := os.ReadFile(DefaultDataPath())
	if err != nil {
		t.Fatalf("read lessons data: %v", err)
	}

	var lessons []struct {
		ID    string `json:"id"`
		Lines []struct {
			ID    string                       `json:"id"`
			Moves []map[string]json.RawMessage `json:"moves"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(data, &lessons); err != nil {
		t.Fatalf("decode lessons data: %v", err)
	}

	for _, lesson := range lessons {
		for _, line := range lesson.Lines {
			for _, move := range line.Moves {
				if _, exists := move["notation"]; exists {
					t.Fatalf("%s/%s/%s: move payload should not include notation", lesson.ID, line.ID, string(move["id"]))
				}
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
		a.Comment == b.Comment
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

func hasLessonTitle(lessons []Lesson, title string) bool {
	for _, lesson := range lessons {
		if lesson.Title == title {
			return true
		}
	}
	return false
}
