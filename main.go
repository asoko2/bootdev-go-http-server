package main

import (
	"database/sql"
	"go-http-server/internal/database"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
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
	godotenv.Load()

	const filepathRoot = "."
	const port = "8080"
	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)

	route := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: route,
	}
	apiCfg := &apiConfig{
		dbQueries: dbQueries,
	}

	// Define routes
	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	route.Handle("/app/", fsHandler)

	route.HandleFunc("GET /api/healthz", handlerReadiness)

	route.HandleFunc("GET /admin/metrics", apiCfg.handlerWriteServerHist)
	route.HandleFunc("POST /admin/reset/", apiCfg.resetServerHits)

	route.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

	log.Fatal(server.ListenAndServe())
}
