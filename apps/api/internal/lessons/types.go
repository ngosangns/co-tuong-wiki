package lessons

type Coordinate struct {
	File int `json:"file"`
	Rank int `json:"rank"`
}

type Move struct {
	ID      string     `json:"id"`
	Side    string     `json:"side"`
	From    Coordinate `json:"from"`
	To      Coordinate `json:"to"`
	Comment string     `json:"comment"`
}

type ChoiceOption struct {
	MoveID   string `json:"moveId"`
	Label    string `json:"label"`
	Verdict  string `json:"verdict"`
	Feedback string `json:"feedback"`
}

type Choice struct {
	Prompt  string         `json:"prompt"`
	Options []ChoiceOption `json:"options"`
}

type Line struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	InitialFEN string `json:"initialFen,omitempty"`
	Moves      []Move `json:"moves"`
}

type Lesson struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Difficulty string `json:"difficulty"`
	InitialFEN string `json:"initialFen,omitempty"`
	Lines      []Line `json:"lines"`
	Choice     Choice `json:"choice"`
}

type Summary struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Difficulty string `json:"difficulty"`
}
