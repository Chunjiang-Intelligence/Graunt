package service

import (
	"data-flywheel/internal/external"
	"data-flywheel/internal/model"
	"fmt"
	"sync"
)

type PosttrainService struct {
	VLLMClient *external.VLLMClient
}

func NewPosttrainService() *PosttrainService {
	return &PosttrainService{VLLMClient: external.NewVLLMClient()}
}

func (s *PosttrainService) EvolInstruct(req model.EvolInstructRequest) (string, error) {
	systemPrompt := ""
	if req.EvolType == "in-depth" {
		systemPrompt = "I want you act as a Prompt Rewriter. Make the given prompt more complex, adding constraints and multiple steps of reasoning required."
	} else {
		systemPrompt = "I want you act as a Prompt Rewriter. Create a completely new prompt that belongs to the same domain but tackles a different, broader topic."
	}

	vllmReq := model.VLLMRequest{
		Model: req.Model,
		Messages: []model.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: req.Instruction},
		},
		MaxTokens: 1024, Temperature: 0.7,
	}

	resp, err := s.VLLMClient.CallChatCompletion(req.VLLMBaseURL, vllmReq)
	if err != nil || len(resp.Choices) == 0 { return "", fmt.Errorf("evol failed") }
	return resp.Choices[0].Message.Content, nil
}

func (s *PosttrainService) GenerateDPOPairs(req model.DPOConstructRequest) (*model.DPOPair, error) {
	var wg sync.WaitGroup
	var chosen, rejected string
	var errChosen, errRejected error

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := s.VLLMClient.CallChatCompletion(req.VLLMBaseURL, model.VLLMRequest{
			Model: req.Model,
			Messages: []model.Message{
				{Role: "system", Content: "You are a helpful, extremely accurate, and detailed AI assistant."},
				{Role: "user", Content: req.Prompt},
			},
			MaxTokens: 1024, Temperature: 0.2,
		})
		if err == nil && len(r.Choices) > 0 { chosen = r.Choices[0].Message.Content } else { errChosen = err }
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		r, err := s.VLLMClient.CallChatCompletion(req.VLLMBaseURL, model.VLLMRequest{
			Model: req.Model,
			Messages: []model.Message{
				{Role: "system", Content: "You are a lazy assistant. Give a very short, unhelpful, and generic answer."},
				{Role: "user", Content: req.Prompt},
			},
			MaxTokens: 1024, Temperature: 1.2,
		})
		if err == nil && len(r.Choices) > 0 { rejected = r.Choices[0].Message.Content } else { errRejected = err }
	}()

	wg.Wait()

	if errChosen != nil || errRejected != nil {
		return nil, fmt.Errorf("failed to generate pairs")
	}

	return &model.DPOPair{Prompt: req.Prompt, Chosen: chosen, Rejected: rejected}, nil
}