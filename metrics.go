package main

import (
	"fmt"
	"log"
	"net/http"
)

func (cfg *apiConfig) handlerWriteServerHist(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	page := fmt.Sprintf(`
<html>
	<body>
		<h1>Welcome, Chirpy Admin</h1>
		<p>Chirpy has been visited %d times!</p>
	</body>
</html>
`, hits)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(page))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
