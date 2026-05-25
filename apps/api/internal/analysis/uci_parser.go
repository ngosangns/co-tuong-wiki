package analysis

import (
	"strconv"
	"strings"
)

type uciSearchResult struct {
	Depth    int
	ScoreCP  int
	BestMove string
	PV       []string
}

func parseUCILine(result *uciSearchResult, line string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}

	switch fields[0] {
	case "bestmove":
		if len(fields) >= 2 && fields[1] != "(none)" {
			result.BestMove = fields[1]
		}
	case "info":
		parseUCIInfo(result, fields)
	}
}

func parseUCIInfo(result *uciSearchResult, fields []string) {
	for index := 1; index < len(fields); index += 1 {
		switch fields[index] {
		case "depth":
			if index+1 < len(fields) {
				if depth, err := strconv.Atoi(fields[index+1]); err == nil {
					result.Depth = depth
				}
				index += 1
			}
		case "score":
			if index+2 < len(fields) {
				scoreType := fields[index+1]
				scoreValue, err := strconv.Atoi(fields[index+2])
				if err == nil {
					if scoreType == "cp" {
						result.ScoreCP = scoreValue
					}
					if scoreType == "mate" {
						result.ScoreCP = mateScore(scoreValue)
					}
				}
				index += 2
			}
		case "pv":
			if index+1 < len(fields) {
				result.PV = append([]string{}, fields[index+1:]...)
			}
			return
		}
	}
}

func mateScore(mateIn int) int {
	if mateIn < 0 {
		return -30000 - mateIn
	}
	return 30000 - mateIn
}
