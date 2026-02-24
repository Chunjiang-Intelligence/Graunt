package external

import (
	"bytes"
	"data-flywheel/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type VLLMClient struct{}

func NewVLLMClient() *VLLMClient { return &VLLMClient{} }

func (c *VLLMClient) CallChatCompletion(baseURL string, req model.VLLMRequest) (*model.VLLMResponse, error) {
	if baseURL == "" { return nil, errors.New("vllm base url is empty") }
	
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(httpReq)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bts, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error: %s", string(bts))
	}

	var vllmResp model.VLLMResponse
	json.NewDecoder(resp.Body).Decode(&vllmResp)
	return &vllmResp, nil
}