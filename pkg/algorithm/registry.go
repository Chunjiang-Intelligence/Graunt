package service

import (
	"graunt/pkg/algorithm"
	"fmt"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	algorithms map[string]algorithm.FilterAlgorithm
}

var globalRegistry = &Registry{
	algorithms: make(map[string]algorithm.FilterAlgorithm),
}

func Register(algo algorithm.FilterAlgorithm) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.algorithms[algo.Name()] = algo
}

func GetAlgorithm(name string) (algorithm.FilterAlgorithm, error) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	algo, exists := globalRegistry.algorithms[name]
	if !exists {
		return nil, fmt.Errorf("algorithm '%s' not found", name)
	}
	return algo, nil
}