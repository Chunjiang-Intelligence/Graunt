package cluster

import (
	"math"
	"math/rand"
	"time"
)

func EuclideanDistance(v1, v2 Vector) float64 {
	sum := 0.0
	keys := make(map[string]bool)
	for k := range v1 { keys[k] = true }
	for k := range v2 { keys[k] = true }

	for k := range keys {
		val1 := v1[k]
		val2 := v2[k]
		sum += (val1 - val2) * (val1 - val2)
	}
	return math.Sqrt(sum)
}

func KMeans(vectors []Vector, k int, maxIters int) []int {
	n := len(vectors)
	if n == 0 || k <= 0 {
		return nil
	}
	if k >= n {
		res := make([]int, n)
		for i := range res { res[i] = i }
		return res
	}

	rand.Seed(time.Now().UnixNano())
	centroids := make([]Vector, k)
	for i := 0; i < k; i++ {
		centroids[i] = vectors[rand.Intn(n)]
	}

	assignments := make([]int, n)

	for iter := 0; iter < maxIters; iter++ {
		changed := false
		
		for i, v := range vectors {
			minDist := math.MaxFloat64
			bestCluster := 0
			for c, centroid := range centroids {
				dist := EuclideanDistance(v, centroid)
				if dist < minDist {
					minDist = dist
					bestCluster = c
				}
			}
			if assignments[i] != bestCluster {
				assignments[i] = bestCluster
				changed = true
			}
		}

		if !changed {
			break
		}

		newCentroids := make([]Vector, k)
		counts := make([]int, k)
		for i := 0; i < k; i++ { newCentroids[i] = make(Vector) }

		for i, v := range vectors {
			clusterIdx := assignments[i]
			counts[clusterIdx]++
			for word, val := range v {
				newCentroids[clusterIdx][word] += val
			}
		}

		for c := 0; c < k; c++ {
			if counts[c] > 0 {
				for word := range newCentroids[c] {
					newCentroids[c][word] /= float64(counts[c])
				}
				centroids[c] = newCentroids[c]
			}
		}
	}
	return assignments
}