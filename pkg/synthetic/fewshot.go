package synthetic

import (
	"graunt/internal/external"
	"graunt/internal/model"
	"graunt/internal/store"
	"fmt"
)

type FewshotSynthetic struct{}
func (s *FewshotSynthetic) Name() string { return "fewshot" }

func (s *FewshotSynthetic) Synthesize(prompt string, params map[string]interface{}, vllm *external.VLLMClient) (interface{}, error) {
	domain := "general"
	if d, ok := params["domain"].(string); ok { domain = d }

	expertSamples := store.GlobalDataStore.GetExpertData()
	refSamples := store.GlobalDataStore.GetReferenceData()

	sysPrompt := fmt.Sprintf("You are an expert in %s. Generate matching data format.\n\n", domain)
	for i, qa := range expertSamples { if i > 1 { break }; sysPrompt += fmt.Sprintf("Expert - Q: %s A: %s\n", qa.Question, qa.Answer) }
	for i, qa := range refSamples { if i > 1 { break }; sysPrompt += fmt.Sprintf("Ref - Q: %s A: %s\n", qa.Question, qa.Answer) }

	req := model.VLLMRequest{
		Model:       params["model"].(string),
		Messages:    []model.Message{{Role: "system", Content: sysPrompt}, {Role: "user", Content: prompt}},
		MaxTokens:   2048, Temperature: 0.8,
	}

	resp, err := vllm.CallChatCompletion(params["vllm_base_url"].(string), req)
	if err != nil || len(resp.Choices) == 0 { return nil, fmt.Errorf("fewshot generation failed") }
	return resp.Choices[0].Message.Content, nil
}