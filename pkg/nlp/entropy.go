package nlp

import (
	"math"
)

func CalculateShannonEntropy(text string) float64 {
	if len(text) == 0 {
		return 0
	}

	frequency := make(map[rune]float64)
	totalChars := 0

	for _, char := range text {
		frequency[char]++
		totalChars++
	}

	entropy := 0.0
	for _, count := range frequency {
		prob := count / float64(totalChars)
		entropy -= prob * math.Log2(prob)
	}

	return entropy
}