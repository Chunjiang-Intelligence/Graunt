package minhash

import (
	"hash/fnv"
	"math"
	"strings"
)

const NumHashFunctions = 100

func GetSignature(text string) []uint32 {
	words := strings.Fields(strings.ToLower(text))
	sig := make([]uint32, NumHashFunctions)
	for i := range sig {
		sig[i] = math.MaxUint32
	}

	for _, word := range words {
		for i := 0; i < NumHashFunctions; i++ {
			h := hashString(word, uint32(i))
			if h < sig[i] {
				sig[i] = h
			}
		}
	}
	return sig
}

func JaccardSimilarity(sig1, sig2 []uint32) float64 {
	matches := 0
	for i := 0; i < NumHashFunctions; i++ {
		if sig1[i] == sig2[i] {
			matches++
		}
	}
	return float64(matches) / float64(NumHashFunctions)
}

func hashString(s string, seed uint32) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32() ^ seed
}