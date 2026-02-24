package main

import (
	"graunt/internal/api"
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	handler := api.NewAPIHandler()
	handler.RegisterRoutes(mux)
	port := ":8080"
	fmt.Printf("Data Flywheel API started on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}