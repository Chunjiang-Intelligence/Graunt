package nlp

import (
	"strings"
)

func CalculateNGramRepetitionRatio(text string, n int) float64 {
	words := strings.Fields(text)
	if len(words) < n {
		return 0.0
	}

	ngrams := make(map[string]int)
	totalNgrams := len(words) - n + 1

	for i := 0; i <= len(words)-n; i++ {
		gram := strings.Join(words[i:i+n], " ")
		ngrams[gram]++
	}

	uniqueNgrams := len(ngrams)
	return 1.0 - (float64(uniqueNgrams) / float64(totalNgrams))
}