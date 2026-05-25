package analysis

type Coordinate struct {
	File int `json:"file"`
	Rank int `json:"rank"`
}

type Move struct {
	From     Coordinate `json:"from"`
	To       Coordinate `json:"to"`
	Notation string     `json:"notation,omitempty"`
	Score    *int       `json:"score,omitempty"`
}

type Request struct {
	FEN        string `json:"fen"`
	SideToMove string `json:"sideToMove"`
	NextMove   *Move  `json:"nextMove,omitempty"`
	TimeMS     int    `json:"timeMs,omitempty"`
	Depth      int    `json:"depth,omitempty"`
}

type Response struct {
	FEN                string `json:"fen"`
	SideToMove         string `json:"sideToMove"`
	Score              Score  `json:"score"`
	BestMove           *Move  `json:"bestMove,omitempty"`
	PrincipalVariation []Move `json:"principalVariation"`
	Depth              int    `json:"depth"`
	Source             string `json:"source"`
	Message            string `json:"message,omitempty"`
}

type Score struct {
	CP          int    `json:"cp"`
	Perspective string `json:"perspective"`
}
