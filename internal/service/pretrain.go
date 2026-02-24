package service

import (
	"graunt/internal/external"
	"graunt/internal/model"
	"graunt/pkg/nlp"
	"fmt"
)

type PretrainService struct {
	VLLMClient *external.VLLMClient
}

func NewPretrainService() *PretrainService {
	return &PretrainService{
		VLLMClient: external.NewVLLMClient(),
	}
}

func (s *PretrainService) FilterData(req model.PretrainFilterRequest) (bool, string) {
	entropy := nlp.CalculateShannonEntropy(req.Text)
	if entropy < req.EntropyThreshold {
		return false, fmt.Sprintf("dropped: entropy %f is lower than threshold %f", entropy, req.EntropyThreshold)
	}

	ngramRatio := nlp.CalculateNGramRepetitionRatio(req.Text, req.NGramN)
	if ngramRatio > req.NGramThreshold {
		return false, fmt.Sprintf("dropped: %d-gram repetition ratio %f is higher than threshold %f", req.NGramN, ngramRatio, req.NGramThreshold)
	}

	return true, "passed"
}

func (s *PretrainService) TextbookRewrite(req model.PretrainRewriteRequest) (string, error) {
	vllmReq := model.VLLMRequest{
		Model: req.TeacherModel,
		Messages: []model.Message{
			{Role: "system", Content: "You are an expert educator. Rewrite the following text into a high-quality, textbook-level explanation."},
			{Role: "user", Content: req.Text},
		},
		MaxTokens:   2048,
		Temperature: 0.3,
	}

	resp, err := s.VLLMClient.CallChatCompletion(req.VLLMBaseURL, vllmReq)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no choices returned from vLLM")
}

func (s *PretrainService) Distill(req model.PretrainDistillRequest) (interface{}, error) {
	vllmReq := model.VLLMRequest{
		Model: req.Model,
		Messages: []model.Message{
			{Role: "user", Content: req.Prompt},
		},
		MaxTokens:   1024,
		Temperature: 0.7,
	}

	if req.DistillType == "logits" {
		vllmReq.Logprobs = true
		vllmReq.TopLogprobs = 5
	}

	resp, err := s.VLLMClient.CallChatCompletion(req.VLLMBaseURL, vllmReq)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from distill model")
	}

	if req.DistillType == "logits" {
		return resp.Choices[0].Logprobs, nil
	}

	return resp.Choices[0].Message.Content, nil
}