package model

type DynamicRequest struct {
	Algorithm   string                 `json:"algorithm"`     // 调用的算法名
	Text        string                 `json:"text"`          // 目标文本 (Filter/Rewrite使用)
	Prompt      string                 `json:"prompt"`        // 目标提示词 (Distill/Synthetic使用)
	Params      map[string]interface{} `json:"params"`        // 动态参数字典
	Model       string                 `json:"model"`         // 外部模型名称
	VLLMBaseURL string                 `json:"vllm_base_url"` // vLLM 地址
}

type PipelineFilterRequest struct {
	Text       string                 `json:"text"`
	Algorithms []string               `json:"algorithms"`
	Params     map[string]interface{} `json:"params"`
}

type QAPair struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Domain   string `json:"domain,omitempty"`
}

type DPOPair struct {
	Prompt   string `json:"prompt"`
	Chosen   string `json:"chosen"`
	Rejected string `json:"rejected"`
}

type PretrainClusterRequest struct {
	Texts []string `json:"texts"`
	K     int      `json:"k"`
}

type RLHFKnownEvalRequest struct {
	UserID        string `json:"user_id"`
	QuestionID    string `json:"question_id"`
	UserEval      bool   `json:"user_eval"`
	ActualCorrect bool   `json:"actual_correct"`
}

type RLHFInferRequest struct {
	UserID     string `json:"user_id"`
	QuestionID string `json:"question_id"`
	UserEval   bool   `json:"user_eval"`
}

type VLLMRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	Logprobs    bool      `json:"logprobs,omitempty"`
	TopLogprobs int       `json:"top_logprobs,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type VLLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Logprobs interface{} `json:"logprobs"`
	} `json:"choices"`
}