package synthetic

import (
	"graunt/internal/external"
	"graunt/internal/model"
	"fmt"
)

type ConstitutionalAI struct{}

func (c *ConstitutionalAI) Name() string { return "constitutional_ai" }

func (c *ConstitutionalAI) Synthesize(prompt string, params map[string]interface{}, vllm *external.VLLMClient) (interface{}, error) {
	principle := "Ensure the response is helpful, harmless, and completely objective."
	if p, ok := params["principle"].(string); ok { principle = p }
	baseURL := params["vllm_base_url"].(string)
	modelName := params["model"].(string)

	resp1, err := vllm.CallChatCompletion(baseURL, model.VLLMRequest{
		Model: modelName, Messages: []model.Message{{Role: "user", Content: prompt}}, MaxTokens: 1024,
	})
	if err != nil { return nil, err }
	initialDraft := resp1.Choices[0].Message.Content

	critiquePrompt := fmt.Sprintf("Draft: %s\n\nCritique the draft based on this principle: '%s'. Identify any violations.", initialDraft, principle)
	resp2, err := vllm.CallChatCompletion(baseURL, model.VLLMRequest{
		Model: modelName, Messages: []model.Message{{Role: "user", Content: critiquePrompt}}, MaxTokens: 512,
	})
	if err != nil { return nil, err }
	critique := resp2.Choices[0].Message.Content

	revisePrompt := fmt.Sprintf("Original Draft: %s\nCritique: %s\n\nRewrite the draft to address the critique.", initialDraft, critique)
	resp3, err := vllm.CallChatCompletion(baseURL, model.VLLMRequest{
		Model: modelName, Messages: []model.Message{{Role: "user", Content: revisePrompt}}, MaxTokens: 1024,
	})
	if err != nil { return nil, err }
	
	return map[string]string{
		"initial":  initialDraft,
		"critique": critique,
		"revised":  resp3.Choices[0].Message.Content,
	}, nil
}