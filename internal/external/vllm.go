package external

import (
	"bytes"
	"graunt/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type VLLMClient struct{}

func NewVLLMClient() *VLLMClient {
	return &VLLMClient{}
}

func (c *VLLMClient) CallChatCompletion(baseURL string, req model.VLLMRequest) (*model.VLLMResponse, error) {
	if baseURL == "" {
		return nil, errors.New("vllm base url is empty")
	}
	url := fmt.Sprintf("%s/v1/chat/completions", baseURL)

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vllm api error: status %d, body: %s", resp.StatusCode, string(respBytes))
	}

	var vllmResp model.VLLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&vllmResp); err != nil {
		return nil, err
	}

	return &vllmResp, nil
}