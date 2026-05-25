package lessons

type Coordinate struct {
	File int `json:"file"`
	Rank int `json:"rank"`
}

type Move struct {
	ID         string     `json:"id"`
	Side       string     `json:"side"`
	From       Coordinate `json:"from"`
	To         Coordinate `json:"to"`
	Notation   string     `json:"notation"`
	Title      string     `json:"title"`
	Comment    string     `json:"comment"`
	Evaluation string     `json:"evaluation,omitempty"`
}

type ChoiceOption struct {
	MoveID   string `json:"moveId"`
	Label    string `json:"label"`
	Verdict  string `json:"verdict"`
	Feedback string `json:"feedback"`
}

type Choice struct {
	ID      string         `json:"id"`
	Prompt  string         `json:"prompt"`
	Options []ChoiceOption `json:"options"`
}

type Line struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Moves       []Move `json:"moves"`
}

type Lesson struct {
	ID         string   `json:"id"`
	Slug       string   `json:"slug"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Category   string   `json:"category"`
	Difficulty string   `json:"difficulty"`
	Tags       []string `json:"tags"`
	Principles []string `json:"principles"`
	InitialFEN string   `json:"initialFen,omitempty"`
	Lines      []Line   `json:"lines"`
	Choice     Choice   `json:"choice"`
}

type Summary struct {
	ID         string   `json:"id"`
	Slug       string   `json:"slug"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Category   string   `json:"category"`
	Difficulty string   `json:"difficulty"`
	Tags       []string `json:"tags"`
}
