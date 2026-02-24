package api

import (
	"data-flywheel/internal/model"
	"data-flywheel/internal/service"
	"encoding/json"
	"net/http"
)

type APIHandler struct {
	RLHF      *service.RLHFService
	Pretrain  *service.PretrainService
	Posttrain *service.PosttrainService
}

func NewAPIHandler() *APIHandler {
	return &APIHandler{
		RLHF:      service.NewRLHFService(),
		Pretrain:  service.NewPretrainService(),
		Posttrain: service.NewPosttrainService(),
	}
}

func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/rlhf/known_eval", h.handleRLHFKnownEval)
	mux.HandleFunc("POST /api/rlhf/infer", h.handleRLHFInfer)
	mux.HandleFunc("POST /api/pretrain/filter", h.handlePretrainFilter)
	mux.HandleFunc("POST /api/pretrain/rewrite", h.handlePretrainRewrite)
	mux.HandleFunc("POST /api/pretrain/distill", h.handlePretrainDistill)
	mux.HandleFunc("POST /api/posttrain/expert", h.handlePosttrainExpert)
	mux.HandleFunc("POST /api/posttrain/reference", h.handlePosttrainReference)
	mux.HandleFunc("POST /api/posttrain/synthetic", h.handlePosttrainSynthetic)
	mux.HandleFunc("POST /api/posttrain/distill", h.handlePosttrainDistill)
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func parseJSON(r *http.Request, dest interface{}) error {
	return json.NewDecoder(r.Body).Decode(dest)
}

func (h *APIHandler) handleRLHFKnownEval(w http.ResponseWriter, r *http.Request) {
	var req model.RLHFKnownEvalRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.RLHF.SubmitKnownEvaluation(req.UserID, req.UserEval, req.ActualCorrect)
	respondJSON(w, http.StatusOK, map[string]string{"status": "profile updated"})
}

func (h *APIHandler) handleRLHFInfer(w http.ResponseWriter, r *http.Request) {
	var req model.RLHFInferRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	prob := h.RLHF.InferUnknownEvaluation(req.UserID, req.UserEval)
	respondJSON(w, http.StatusOK, map[string]float64{"inferred_actual_correct_probability": prob})
}

func (h *APIHandler) handlePretrainFilter(w http.ResponseWriter, r *http.Request) {
	var req model.PretrainFilterRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	passed, reason := h.Pretrain.FilterData(req)
	respondJSON(w, http.StatusOK, map[string]interface{}{"passed": passed, "reason": reason})
}

func (h *APIHandler) handlePretrainRewrite(w http.ResponseWriter, r *http.Request) {
	var req model.PretrainRewriteRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	rewritten, err := h.Pretrain.TextbookRewrite(req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"rewritten_text": rewritten})
}

func (h *APIHandler) handlePretrainDistill(w http.ResponseWriter, r *http.Request) {
	var req model.PretrainDistillRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := h.Pretrain.Distill(req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"distilled_data": result})
}

func (h *APIHandler) handlePosttrainExpert(w http.ResponseWriter, r *http.Request) {
	var req model.PosttrainExpertRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.Posttrain.AddExpertData(req.QA)
	respondJSON(w, http.StatusOK, map[string]string{"status": "expert data stored"})
}

func (h *APIHandler) handlePosttrainReference(w http.ResponseWriter, r *http.Request) {
	var req model.PosttrainReferenceRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.Posttrain.AddReferenceData(req.QA)
	respondJSON(w, http.StatusOK, map[string]string{"status": "reference QA stored"})
}

func (h *APIHandler) handlePosttrainSynthetic(w http.ResponseWriter, r *http.Request) {
	var req model.PosttrainSyntheticRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	data, err := h.Posttrain.GenerateSyntheticData(req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"synthetic_data": data})
}

func (h *APIHandler) handlePosttrainDistill(w http.ResponseWriter, r *http.Request) {
	var req model.PretrainDistillRequest
	if err := parseJSON(r, &req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	result, err := h.Posttrain.Distill(req)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"distilled_data": result})
}