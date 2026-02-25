package filter

import (
	"fmt"
	"math"
)

type EntropyFilter struct{}
func (f *EntropyFilter) Name() string { return "entropy" }
func (f *EntropyFilter) Evaluate(text string, params map[string]interface{}) (bool, string) {
	threshold := 2.0
	if t, ok := params["entropy_threshold"].(float64); ok { threshold = t }

	freq := make(map[rune]float64)
	total := 0
	for _, char := range text { freq[char]++; total++ }

	entropy := 0.0
	for _, count := range freq {
		prob := count / float64(total)
		entropy -= prob * math.Log2(prob)
	}

	if entropy < threshold { return false, fmt.Sprintf("entropy %f < %f", entropy, threshold) }
	return true, "ok"
}