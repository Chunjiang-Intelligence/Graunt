package algorithm

import (
	"graunt/internal/external"
)

type FilterAlgorithm interface {
	Name() string
	Evaluate(text string, params map[string]interface{}) (bool, string)
}

type RewriteAlgorithm interface {
	Name() string
	Rewrite(text string, params map[string]interface{}, vllm *external.VLLMClient) (string, error)
}

type DistillAlgorithm interface {
	Name() string
	Distill(prompt string, params map[string]interface{}, vllm *external.VLLMClient) (interface{}, error)
}

type SyntheticAlgorithm interface {
	Name() string
	Synthesize(prompt string, params map[string]interface{}, vllm *external.VLLMClient) (interface{}, error)
}