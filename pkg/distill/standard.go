package distill

import (
	"graunt/internal/external"
	"graunt/internal/model"
	"fmt"
)

type StandardDistill struct{}

func (d *StandardDistill) Name() string { return "standard_distill" }

func (d *StandardDistill) Distill(prompt string, params map[string]interface{}, vllm *external.VLLMClient) (interface{}, error) {
	distillType := "sft"
	if val, ok := params["distill_type"].(string); ok { distillType = val }

	req := model.VLLMRequest{
		Model:       params["model"].(string),
		Messages:    []model.Message{{Role: "user", Content: prompt}},
		MaxTokens:   1024,
		Temperature: 0.7,
	}

	if distillType == "logits" {
		req.Logprobs = true
		req.TopLogprobs = 5
	}

	resp, err := vllm.CallChatCompletion(params["vllm_base_url"].(string), req)
	if err != nil || len(resp.Choices) == 0 { return nil, fmt.Errorf("distill failed") }

	if distillType == "logits" {
		return resp.Choices[0].Logprobs, nil
	}
	return resp.Choices[0].Message.Content, nil
}