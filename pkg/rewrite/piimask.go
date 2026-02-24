package rewrite

import (
	"graunt/internal/external"
	"regexp"
)

type PIIMaskRewrite struct{}

func (r *PIIMaskRewrite) Name() string { return "pii_mask" }

func (r *PIIMaskRewrite) Rewrite(text string, params map[string]interface{}, vllm *external.VLLMClient) (string, error) {
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phoneRegex := regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`)

	masked := emailRegex.ReplaceAllString(text, "[EMAIL]")
	masked = phoneRegex.ReplaceAllString(masked, "[PHONE]")

	return masked, nil
}