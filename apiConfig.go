package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	"github.com/wilgnert/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	plataform string
	secret string
	polka_key string
}

func (cfg *apiConfig) init() error {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)	
	}
	cfg.dbQueries = database.New(db)
	cfg.plataform = os.Getenv("PLATAFORM")
	cfg.secret = os.Getenv("SECRET")
	cfg.polka_key = os.Getenv("POLKA_KEY")
	return nil
}


func HandleHealth(w http.ResponseWriter, r *http.Request) {
	RespondOK(w, r)
}

func (cfg *apiConfig) showMetrics(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load()))
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) error {
	res, err := json.Marshal(payload)
	if err != nil {
		return err;
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write(res)
	return nil
}

func respondWithError(w http.ResponseWriter, code int, msg string) error {
	return respondWithJSON(w, code, map[string]string{"error": msg})
}

func RespondOK(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "OK")
}

func RespondNoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}