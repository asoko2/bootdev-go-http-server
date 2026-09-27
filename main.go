package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) handlerWriteServerHist(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	hits := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
	w.Write([]byte(hits))
}

func (cfg *apiConfig) resetServerHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset"))
}

func main() {
	route := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: route,
	}
	apiCfg := &apiConfig{}

	// Define routes
	route.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	route.HandleFunc("GET /api/healthz", handlerReadiness)
	route.HandleFunc("GET /api/metrics", apiCfg.handlerWriteServerHist)
	route.HandleFunc("POST /api/reset/", apiCfg.resetServerHits)

	log.Fatal(server.ListenAndServe())
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
