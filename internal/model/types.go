package model

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

type PretrainFilterRequest struct {
	Text             string  `json:"text"`
	EntropyThreshold float64 `json:"entropy_threshold"`
	NGramN           int     `json:"n_gram_n"`
	NGramThreshold   float64 `json:"n_gram_threshold"`
}

type PretrainRewriteRequest struct {
	Text         string `json:"text"`
	TeacherModel string `json:"teacher_model"`
	VLLMBaseURL  string `json:"vllm_base_url"`
}

type PretrainDistillRequest struct {
	Prompt      string `json:"prompt"`
	DistillType string `json:"distill_type"`
	Model       string `json:"model"`
	VLLMBaseURL string `json:"vllm_base_url"`
}

type QAPair struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Domain   string `json:"domain"`
}

type PosttrainExpertRequest struct {
	ExpertID string `json:"expert_id"`
	QA       QAPair `json:"qa"`
}

type PosttrainReferenceRequest struct {
	BenchmarkName string `json:"benchmark_name"`
	QA            QAPair `json:"qa"`
}

type PosttrainSyntheticRequest struct {
	Domain      string `json:"domain"`
	Count       int    `json:"count"`
	Model       string `json:"model"`
	VLLMBaseURL string `json:"vllm_base_url"`
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