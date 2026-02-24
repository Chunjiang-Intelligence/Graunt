package filter

import (
	"fmt"
	"strings"
)

type ReadabilityFilter struct{}

func (f *ReadabilityFilter) Name() string { return "readability_fog" }

func (f *ReadabilityFilter) Evaluate(text string, params map[string]interface{}) (bool, string) {
	minFog := 6.0
	if val, ok := params["min_fog_index"].(float64); ok { minFog = val }

	words := strings.Fields(text)
	sentences := strings.Split(text, ".")
	if len(words) == 0 || len(sentences) == 0 {
		return false, "empty text"
	}

	complexWords := 0
	for _, word := range words {
		if countSyllables(word) >= 3 {
			complexWords++
		}
	}

	wordsPerSentence := float64(len(words)) / float64(len(sentences))
	percentageComplex := (float64(complexWords) / float64(len(words))) * 100
	fogIndex := 0.4 * (wordsPerSentence + percentageComplex)

	if fogIndex < minFog {
		return false, fmt.Sprintf("fog index %f < threshold %f", fogIndex, minFog)
	}
	return true, "ok"
}

func countSyllables(word string) int {
	word = strings.ToLower(word)
	vowels := "aeiouy"
	count := 0
	prevVowel := false
	for _, char := range word {
		isVowel := strings.ContainsRune(vowels, char)
		if isVowel && !prevVowel {
			count++
		}
		prevVowel = isVowel
	}
	if strings.HasSuffix(word, "e") { count-- }
	if count <= 0 { count = 1 }
	return count
}