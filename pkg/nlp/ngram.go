package nlp

import (
	"fmt"
	"strings"
)

type NGramFilter struct{}

func (f *NGramFilter) Name() string { return "ngram" }

func (f *NGramFilter) Evaluate(text string, params map[string]interface{}) (bool, string) {
	n := 3
	threshold := 0.5
	if nv, ok := params["ngram_n"].(float64); ok { n = int(nv) }
	if tv, ok := params["ngram_threshold"].(float64); ok { threshold = tv }

	words := strings.Fields(text)
	if len(words) < n { return true, "ok" }

	ngrams := make(map[string]int)
	total := len(words) - n + 1
	for i := 0; i <= len(words)-n; i++ {
		gram := strings.Join(words[i:i+n], " ")
		ngrams[gram]++
	}

	ratio := 1.0 - (float64(len(ngrams)) / float64(total))
	if ratio > threshold {
		return false, fmt.Sprintf("ngram rep ratio %f > %f", ratio, threshold)
	}
	return true, "ok"
}