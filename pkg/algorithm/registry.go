package service

import (
	"graunt/pkg/algorithm"
	"fmt"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	filters    map[string]algorithm.FilterAlgorithm
	rewrites   map[string]algorithm.RewriteAlgorithm
	distills   map[string]algorithm.DistillAlgorithm
	synthetics map[string]algorithm.SyntheticAlgorithm
}

var GlobalRegistry = &Registry{
	filters:    make(map[string]algorithm.FilterAlgorithm),
	rewrites:   make(map[string]algorithm.RewriteAlgorithm),
	distills:   make(map[string]algorithm.DistillAlgorithm),
	synthetics: make(map[string]algorithm.SyntheticAlgorithm),
}

func RegisterFilter(algo algorithm.FilterAlgorithm) {
	GlobalRegistry.mu.Lock()
	defer GlobalRegistry.mu.Unlock()
	GlobalRegistry.filters[algo.Name()] = algo
}

func RegisterRewrite(algo algorithm.RewriteAlgorithm) {
	GlobalRegistry.mu.Lock()
	defer GlobalRegistry.mu.Unlock()
	GlobalRegistry.rewrites[algo.Name()] = algo
}

func RegisterDistill(algo algorithm.DistillAlgorithm) {
	GlobalRegistry.mu.Lock()
	defer GlobalRegistry.mu.Unlock()
	GlobalRegistry.distills[algo.Name()] = algo
}

func RegisterSynthetic(algo algorithm.SyntheticAlgorithm) {
	GlobalRegistry.mu.Lock()
	defer GlobalRegistry.mu.Unlock()
	GlobalRegistry.synthetics[algo.Name()] = algo
}

func GetFilter(name string) (algorithm.FilterAlgorithm, error) {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()
	if algo, ok := GlobalRegistry.filters[name]; ok { return algo, nil }
	return nil, fmt.Errorf("filter algorithm '%s' not found", name)
}

func GetRewrite(name string) (algorithm.RewriteAlgorithm, error) {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()
	if algo, ok := GlobalRegistry.rewrites[name]; ok { return algo, nil }
	return nil, fmt.Errorf("rewrite algorithm '%s' not found", name)
}

func GetDistill(name string) (algorithm.DistillAlgorithm, error) {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()
	if algo, ok := GlobalRegistry.distills[name]; ok { return algo, nil }
	return nil, fmt.Errorf("distill algorithm '%s' not found", name)
}

func GetSynthetic(name string) (algorithm.SyntheticAlgorithm, error) {
	GlobalRegistry.mu.RLock()
	defer GlobalRegistry.mu.RUnlock()
	if algo, ok := GlobalRegistry.synthetics[name]; ok { return algo, nil }
	return nil, fmt.Errorf("synthetic algorithm '%s' not found", name)
}