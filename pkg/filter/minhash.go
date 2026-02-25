package filter

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"sync"
)

const numHashFunctions = 100

type MinHashFilter struct {
	mu        sync.RWMutex
	SignStore [][]uint32
}

func NewMinHashFilter() *MinHashFilter { return &MinHashFilter{SignStore: make([][]uint32, 0)} }
func (f *MinHashFilter) Name() string { return "minhash" }

func getSignature(text string) []uint32 {
	words := strings.Fields(strings.ToLower(text))
	sig := make([]uint32, numHashFunctions)
	for i := range sig { sig[i] = math.MaxUint32 }

	for _, word := range words {
		for i := 0; i < numHashFunctions; i++ {
			h := fnv.New32a(); h.Write([]byte(word)); hashVal := h.Sum32() ^ uint32(i)
			if hashVal < sig[i] { sig[i] = hashVal }
		}
	}
	return sig
}

func jaccardSimilarity(sig1, sig2 []uint32) float64 {
	matches := 0
	for i := 0; i < numHashFunctions; i++ { if sig1[i] == sig2[i] { matches++ } }
	return float64(matches) / float64(numHashFunctions)
}

func (f *MinHashFilter) Evaluate(text string, params map[string]interface{}) (bool, string) {
	threshold := 0.8
	if t, ok := params["minhash_threshold"].(float64); ok { threshold = t }

	sig := getSignature(text)
	f.mu.RLock()
	for _, storedSig := range f.SignStore {
		sim := jaccardSimilarity(sig, storedSig)
		if sim >= threshold {
			f.mu.RUnlock()
			return false, fmt.Sprintf("duplicate found, similarity: %f", sim)
		}
	}
	f.mu.RUnlock()

	f.mu.Lock()
	f.SignStore = append(f.SignStore, sig)
	f.mu.Unlock()
	return true, "unique"
}