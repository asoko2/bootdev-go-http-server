package main

import (
	"database/sql"
	"fmt"
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
	db             *database.Queries
	platform       string
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) resetServerHits(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Reset is only allowed in dev environment."))
		return
	}
	cfg.fileserverHits.Store(0)

	err := cfg.db.DeleteAllUsers(r.Context())
	if err != nil {
		fmt.Println("Failed to reset server hits and users:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to reset the database." + err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0 and database reset to initial state."))
}

func main() {
	godotenv.Load()

	const filepathRoot = "."
	const port = "8080"
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM must be set")
	}

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
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
	}

	// Define routes
	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	route.Handle("/app/", fsHandler)

	route.HandleFunc("GET /api/healthz", handlerReadiness)

	route.HandleFunc("GET /admin/metrics", apiCfg.handlerWriteServerHist)
	route.HandleFunc("POST /admin/reset", apiCfg.resetServerHits)

	route.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)

	route.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	// Start the server
	// Log if server is started
	fmt.Println("Server is starting on port", port)
	log.Fatal(server.ListenAndServe())
}
