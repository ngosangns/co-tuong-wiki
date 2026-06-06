package httpapi

import "math"

type moveClassification string

const (
	classificationBest       moveClassification = "best"
	classificationExcellent  moveClassification = "excellent"
	classificationGood       moveClassification = "good"
	classificationInaccuracy moveClassification = "inaccuracy"
	classificationMistake   moveClassification = "mistake"
	classificationBlunder    moveClassification = "blunder"
)

// cpLoss is the absolute difference between the best continuation's score
// and the score after the actually-played move, both in the engine's
// normalized (red-perspective) coordinate. The magnitude is independent of
// which side played the move.
func cpLoss(bestScore int, actualScore int) int {
	delta := bestScore - actualScore
	if delta < 0 {
		delta = -delta
	}
	return delta
}

func classifyMove(cpLossValue int) moveClassification {
	switch {
	case cpLossValue <= 10:
		return classificationBest
	case cpLossValue <= 30:
		return classificationExcellent
	case cpLossValue <= 80:
		return classificationGood
	case cpLossValue <= 150:
		return classificationInaccuracy
	case cpLossValue <= 300:
		return classificationMistake
	default:
		return classificationBlunder
	}
}

// roundToInt rounds to the nearest integer.
func roundToInt(value float64) int {
	return int(math.Round(value))
}
