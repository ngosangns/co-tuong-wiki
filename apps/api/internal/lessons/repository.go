package lessons

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Repository struct {
	lessons []Lesson
	byID    map[string]Lesson
}

func DefaultDataPath() string {
	if fromEnv := os.Getenv("LESSONS_FILE"); fromEnv != "" {
		return fromEnv
	}

	_, sourceFile, _, _ := runtime.Caller(0)
	apiRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	candidates := []string{
		filepath.Join(apiRoot, "data", "lessons", "lessons.json"),
		filepath.Join("apps", "api", "data", "lessons", "lessons.json"),
		filepath.Join("data", "lessons", "lessons.json"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return candidates[0]
}

func LoadRepository(path string) (*Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var loaded []Lesson
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, err
	}

	repo := &Repository{
		lessons: loaded,
		byID:    make(map[string]Lesson, len(loaded)),
	}

	for _, lesson := range loaded {
		if lesson.ID == "" {
			return nil, errors.New("lesson id is required")
		}
		if _, exists := repo.byID[lesson.ID]; exists {
			return nil, errors.New("duplicate lesson id: " + lesson.ID)
		}
		repo.byID[lesson.ID] = lesson
	}

	return repo, nil
}

func (r *Repository) List(category string, query string, difficulty string) []Summary {
	category = strings.TrimSpace(strings.ToLower(category))
	query = strings.TrimSpace(strings.ToLower(query))
	difficulty = strings.TrimSpace(strings.ToLower(difficulty))

	summaries := make([]Summary, 0, len(r.lessons))
	for _, lesson := range r.lessons {
		if category != "" && strings.ToLower(lesson.Category) != category {
			continue
		}
		if difficulty != "" && strings.ToLower(lesson.Difficulty) != difficulty {
			continue
		}
		if query != "" && !lessonMatchesQuery(lesson, query) {
			continue
		}
		summaries = append(summaries, Summary{
			ID:         lesson.ID,
			Title:      lesson.Title,
			Category:   lesson.Category,
			Difficulty: lesson.Difficulty,
		})
	}

	return summaries
}

func (r *Repository) Get(id string) (Lesson, bool) {
	lesson, ok := r.byID[id]
	return lesson, ok
}

func (r *Repository) Categories() []string {
	seen := map[string]bool{}
	categories := make([]string, 0)

	for _, lesson := range r.lessons {
		if seen[lesson.Category] {
			continue
		}
		seen[lesson.Category] = true
		categories = append(categories, lesson.Category)
	}

	return categories
}

func lessonMatchesQuery(lesson Lesson, query string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		lesson.Title,
		lesson.Category,
		lesson.Difficulty,
	}, " "))

	return strings.Contains(haystack, query)
}
