package cluster

import (
	"math"
	"strings"
)

type Vector map[string]float64

func BuildTFIDF(texts []string) []Vector {
	n := len(texts)
	tfList := make([]map[string]float64, n)
	df := make(map[string]int)

	for i, text := range texts {
		tfList[i] = make(map[string]float64)
		words := strings.Fields(strings.ToLower(text))
		totalWords := float64(len(words))
		
		wordSet := make(map[string]bool)
		for _, w := range words {
			tfList[i][w]++
			wordSet[w] = true
		}
		for w := range tfList[i] {
			tfList[i][w] /= totalWords
		}
		for w := range wordSet {
			df[w]++
		}
	}

	vectors := make([]Vector, n)
	for i := 0; i < n; i++ {
		vectors[i] = make(Vector)
		for w, tf := range tfList[i] {
			idf := math.Log(float64(n) / float64(df[w]+1))
			vectors[i][w] = tf * idf
		}
	}
	return vectors
}