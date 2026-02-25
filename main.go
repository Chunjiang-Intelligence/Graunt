package main

import (
	"graunt/internal/api"
	"graunt/internal/service"
	
	"graunt/pkg/distill"
	"graunt/pkg/filter"
	"graunt/pkg/rewrite"
	"graunt/pkg/synthetic"
	
	"fmt"
	"log"
	"net/http"
)

func main() {
	service.RegisterFilter(&filter.EntropyFilter{})
	service.RegisterFilter(&filter.NGramFilter{})
	service.RegisterFilter(filter.NewMinHashFilter())
	service.RegisterFilter(&filter.ReadabilityFilter{})
	service.RegisterRewrite(&rewrite.TextbookRewrite{})
	service.RegisterRewrite(&rewrite.PIIMaskRewrite{})
	service.RegisterDistill(&distill.StandardDistill{})
	service.RegisterSynthetic(&synthetic.FewshotSynthetic{})
	service.RegisterSynthetic(&synthetic.EvolInstruct{})
	service.RegisterSynthetic(&synthetic.DPOConstruct{})
	service.RegisterSynthetic(&synthetic.ConstitutionalAI{})

	mux := http.NewServeMux()
	handler := api.NewAPIHandler()
	handler.RegisterRoutes(mux)

	port := ":8080"
	fmt.Printf("Data Flywheel Super-Registry API started on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}