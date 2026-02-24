package model

type PipelineFilterRequest struct {
	Text       string                 `json:"text"`
	Algorithms []string               `json:"algorithms"`
	Params     map[string]interface{} `json:"params"`
}

type PretrainClusterRequest struct {
	Texts []string `json:"texts"`
	K     int      `json:"k"`
}

type PretrainRewriteRequest struct {
	Text         string `json:"text"`
	TeacherModel string `json:"teacher_model"`
	VLLMBaseURL  string `json:"vllm_base_url"`
}

type QAPair struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Domain   string `json:"domain,omitempty"`
}

type EvolInstructRequest struct {
	Instruction string `json:"instruction"`
	EvolType    string `json:"evol_type"` // "in-depth" or "in-breadth"
	Model       string `json:"model"`
	VLLMBaseURL string `json:"vllm_base_url"`
}

type DPOPair struct {
	Prompt   string `json:"prompt"`
	Chosen   string `json:"chosen"`
	Rejected string `json:"rejected"`
}

type DPOConstructRequest struct {
	Prompt      string `json:"prompt"`
	Model       string `json:"model"`
	VLLMBaseURL string `json:"vllm_base_url"`
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
	} `json:"choices"`
}