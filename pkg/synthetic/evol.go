package synthetic

import (
	"graunt/internal/external"
	"graunt/internal/model"
	"fmt"
)

type EvolInstruct struct{}
func (e *EvolInstruct) Name() string { return "evol_instruct" }

func (e *EvolInstruct) Synthesize(prompt string, params map[string]interface{}, vllm *external.VLLMClient) (interface{}, error) {
	evolType := "in-depth"
	if t, ok := params["evol_type"].(string); ok { evolType = t }

	sys := "Make the given prompt more complex and add constraints."
	if evolType == "in-breadth" { sys = "Create a new prompt that belongs to the same domain but tackles a broader topic." }

	req := model.VLLMRequest{
		Model:       params["model"].(string),
		Messages:    []model.Message{{Role: "system", Content: sys}, {Role: "user", Content: prompt}},
		MaxTokens:   1024, Temperature: 0.7,
	}

	resp, err := vllm.CallChatCompletion(params["vllm_base_url"].(string), req)
	if err != nil || len(resp.Choices) == 0 { return nil, fmt.Errorf("evol failed") }
	return resp.Choices[0].Message.Content, nil
}