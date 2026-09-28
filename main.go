package main

import (
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

func (cfg *apiConfig) resetServerHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset"))
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	route := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: route,
	}
	apiCfg := &apiConfig{}

	// Define routes
	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	route.Handle("/app/", fsHandler)

	route.HandleFunc("GET /api/healthz", handlerReadiness)

	route.HandleFunc("GET /admin/metrics", apiCfg.handlerWriteServerHist)
	route.HandleFunc("POST /admin/reset/", apiCfg.resetServerHits)

	route.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

	log.Fatal(server.ListenAndServe())
}
