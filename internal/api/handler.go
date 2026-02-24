package api

import (
	"graunt/internal/external"
	"graunt/internal/model"
	"graunt/internal/service"
	"graunt/pkg/cluster"
	"encoding/json"
	"net/http"
)

type APIHandler struct {
	VLLMClient *external.VLLMClient
}

func NewAPIHandler() *APIHandler {
	return &APIHandler{VLLMClient: external.NewVLLMClient()}
}

func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/pipeline/filter", h.handleDynamicFilter)
	mux.HandleFunc("POST /api/dynamic/rewrite", h.handleDynamicRewrite)
	mux.HandleFunc("POST /api/dynamic/distill", h.handleDynamicDistill)
	mux.HandleFunc("POST /api/dynamic/synthetic", h.handleDynamicSynthetic)
	mux.HandleFunc("POST /api/pretrain/cluster", h.handleCluster)
	mux.HandleFunc("POST /api/data/expert", h.handleAddExpert)
	mux.HandleFunc("POST /api/data/reference", h.handleAddReference)
}

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
func parse(r *http.Request, dest interface{}) error { return json.NewDecoder(r.Body).Decode(dest) }

func (h *APIHandler) handleDynamicFilter(w http.ResponseWriter, r *http.Request) {
	var req model.PipelineFilterRequest
	if err := parse(r, &req); err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }

	for _, algoName := range req.Algorithms {
		algo, err := service.GetFilter(algoName)
		if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
		keep, reason := algo.Evaluate(req.Text, req.Params)
		if !keep {
			respond(w, 200, map[string]interface{}{"passed": false, "reason": "Failed at " + algoName + ": " + reason})
			return
		}
	}
	respond(w, 200, map[string]interface{}{"passed": true, "reason": "ok"})
}

func (h *APIHandler) handleDynamicRewrite(w http.ResponseWriter, r *http.Request) {
	var req model.DynamicRequest
	if err := parse(r, &req); err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }

	algo, err := service.GetRewrite(req.Algorithm)
	if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
	
	if req.Params == nil { req.Params = make(map[string]interface{}) }
	req.Params["model"] = req.Model
	req.Params["vllm_base_url"] = req.VLLMBaseURL

	result, err := algo.Rewrite(req.Text, req.Params, h.VLLMClient)
	if err != nil { respond(w, 500, map[string]string{"error": err.Error()}); return }
	respond(w, 200, map[string]string{"rewritten": result})
}

func (h *APIHandler) handleDynamicDistill(w http.ResponseWriter, r *http.Request) {
	var req model.DynamicRequest
	if err := parse(r, &req); err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }

	algo, err := service.GetDistill(req.Algorithm)
	if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }

	if req.Params == nil { req.Params = make(map[string]interface{}) }
	req.Params["model"] = req.Model
	req.Params["vllm_base_url"] = req.VLLMBaseURL

	result, err := algo.Distill(req.Prompt, req.Params, h.VLLMClient)
	if err != nil { respond(w, 500, map[string]string{"error": err.Error()}); return }
	respond(w, 200, map[string]interface{}{"distilled": result})
}

func (h *APIHandler) handleDynamicSynthetic(w http.ResponseWriter, r *http.Request) {
	var req model.DynamicRequest
	if err := parse(r, &req); err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }

	algo, err := service.GetSynthetic(req.Algorithm)
	if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }

	if req.Params == nil { req.Params = make(map[string]interface{}) }
	req.Params["model"] = req.Model
	req.Params["vllm_base_url"] = req.VLLMBaseURL

	result, err := algo.Synthesize(req.Prompt, req.Params, h.VLLMClient)
	if err != nil { respond(w, 500, map[string]string{"error": err.Error()}); return }
	respond(w, 200, map[string]interface{}{"synthetic": result})
}

func (h *APIHandler) handleCluster(w http.ResponseWriter, r *http.Request) {
	var req model.PretrainClusterRequest
	if err := parse(r, &req); err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
	
	vectors := cluster.BuildTFIDF(req.Texts)
	assignments := cluster.KMeans(vectors, req.K, 100)
	
	res := make(map[int][]string)
	for i, c := range assignments { res[c] = append(res[c], req.Texts[i]) }
	respond(w, 200, res)
}

func (h *APIHandler) handleAddExpert(w http.ResponseWriter, r *http.Request) {
	var req model.QAPair
	if err := parse(r, &req); err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
	service.GlobalDataStore.AddExpert(req)
	respond(w, 200, map[string]string{"status": "ok"})
}
func (h *APIHandler) handleAddReference(w http.ResponseWriter, r *http.Request) {
	var req model.QAPair
	if err := parse(r, &req); err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
	service.GlobalDataStore.AddReference(req)
	respond(w, 200, map[string]string{"status": "ok"})
}