package rewrite

import (
	"graunt/internal/external"
	"graunt/internal/model"
	"fmt"
)

type TextbookRewrite struct{}
func (r *TextbookRewrite) Name() string { return "textbook" }
func (r *TextbookRewrite) Rewrite(text string, params map[string]interface{}, vllm *external.VLLMClient) (string, error) {
	req := model.VLLMRequest{
		Model:       params["model"].(string),
		Messages:    []model.Message{{Role: "system", Content: "Rewrite into a textbook-level explanation."}, {Role: "user", Content: text}},
		MaxTokens:   2048, Temperature: 0.3,
	}
	resp, err := vllm.CallChatCompletion(params["vllm_base_url"].(string), req)
	if err != nil || len(resp.Choices) == 0 { return "", fmt.Errorf("failed to rewrite") }
	return resp.Choices[0].Message.Content, nil
}