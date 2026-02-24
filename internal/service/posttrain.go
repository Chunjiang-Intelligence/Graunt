package service

import (
	"data-flywheel/internal/external"
	"data-flywheel/internal/model"
	"encoding/json"
	"fmt"
	"sync"
)

type PosttrainService struct {
	mu            sync.RWMutex
	ExpertData    []model.QAPair
	ReferenceData []model.QAPair
	VLLMClient    *external.VLLMClient
}

func NewPosttrainService() *PosttrainService {
	return &PosttrainService{
		ExpertData:    make([]model.QAPair, 0),
		ReferenceData: make([]model.QAPair, 0),
		VLLMClient:    external.NewVLLMClient(),
	}
}

func (s *PosttrainService) AddExpertData(qa model.QAPair) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExpertData = append(s.ExpertData, qa)
}

func (s *PosttrainService) AddReferenceData(qa model.QAPair) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ReferenceData = append(s.ReferenceData, qa)
}

func (s *PosttrainService) GenerateSyntheticData(req model.PosttrainSyntheticRequest) ([]model.QAPair, error) {
	s.mu.RLock()
	expertSamples := s.ExpertData
	refSamples := s.ReferenceData
	s.mu.RUnlock()

	systemPrompt := fmt.Sprintf("You are an expert in domain: %s. Generate %d new Question-Answer pairs in JSON format: [{\"question\":\"...\", \"answer\":\"...\"}]. Match the style and difficulty of the following examples.\n\n", req.Domain, req.Count)

	for i, qa := range expertSamples {
		if i > 2 {
			break
		}
		systemPrompt += fmt.Sprintf("Expert Example - Q: %s A: %s\n", qa.Question, qa.Answer)
	}
	for i, qa := range refSamples {
		if i > 2 {
			break
		}
		systemPrompt += fmt.Sprintf("Reference Example - Q: %s A: %s\n", qa.Question, qa.Answer)
	}

	vllmReq := model.VLLMRequest{
		Model: req.Model,
		Messages: []model.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: "Generate the JSON array now."},
		},
		MaxTokens:   2048,
		Temperature: 0.8,
	}

	resp, err := s.VLLMClient.CallChatCompletion(req.VLLMBaseURL, vllmReq)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from generation model")
	}

	content := resp.Choices[0].Message.Content
	var synthetics []model.QAPair
	err = json.Unmarshal([]byte(content), &synthetics)
	if err != nil {
		synthetics = []model.QAPair{{Question: "Raw Generated Output", Answer: content, Domain: req.Domain}}
	}

	return synthetics, nil
}

func (s *PosttrainService) Distill(req model.PretrainDistillRequest) (interface{}, error) {
	// Post-train 的蒸馏逻辑与 Pre-train 基础设施复用，但在服务层分离以备后续扩展对齐算法(如KTO数据生成)
	vllmReq := model.VLLMRequest{
		Model: req.Model,
		Messages: []model.Message{
			{Role: "user", Content: req.Prompt},
		},
		MaxTokens:   1024,
		Temperature: 0.5,
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
		return nil, fmt.Errorf("no response")
	}
	if req.DistillType == "logits" {
		return resp.Choices[0].Logprobs, nil
	}
	return resp.Choices[0].Message.Content, nil
}