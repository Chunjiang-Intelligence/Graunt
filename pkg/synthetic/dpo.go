package synthetic

import (
	"graunt/internal/external"
	"graunt/internal/model"
	"fmt"
	"sync"
)

type DPOConstruct struct{}
func (d *DPOConstruct) Name() string { return "dpo_pairs" }

func (d *DPOConstruct) Synthesize(prompt string, params map[string]interface{}, vllm *external.VLLMClient) (interface{}, error) {
	var wg sync.WaitGroup
	var chosen, rejected string
	var err1, err2 error

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := vllm.CallChatCompletion(params["vllm_base_url"].(string), model.VLLMRequest{
			Model: params["model"].(string), Messages: []model.Message{{Role: "system", Content: "Give a perfect answer."}, {Role: "user", Content: prompt}}, MaxTokens: 1024, Temperature: 0.2,
		})
		if err == nil && len(r.Choices) > 0 { chosen = r.Choices[0].Message.Content } else { err1 = err }
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := vllm.CallChatCompletion(params["vllm_base_url"].(string), model.VLLMRequest{
			Model: params["model"].(string), Messages: []model.Message{{Role: "system", Content: "Give a terrible answer."}, {Role: "user", Content: prompt}}, MaxTokens: 1024, Temperature: 1.2,
		})
		if err == nil && len(r.Choices) > 0 { rejected = r.Choices[0].Message.Content } else { err2 = err }
	}()

	wg.Wait()
	if err1 != nil || err2 != nil { return nil, fmt.Errorf("failed generating pairs") }

	return model.DPOPair{Prompt: prompt, Chosen: chosen, Rejected: rejected}, nil
}