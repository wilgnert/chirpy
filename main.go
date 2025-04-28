package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/wilgnert/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	plataform string
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

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func(cfg *apiConfig) resetMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.plataform != "dev" {
			respondWithError(w, http.StatusForbidden, "403 Forbidden")
			return
		}
		cfg.dbQueries.DeleteAllUsers(r.Context())
		cfg.fileserverHits.Store(0);
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}
	var p parameters
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	if err := decoder.Decode(&p); err != nil {
		if err := respondWithError(w, http.StatusInternalServerError, "Something went wrong"); err != nil {
			fmt.Println("Could not respond to request")
		}
		return
	}
	user, err := cfg.dbQueries.CreateUser(r.Context(), p.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create user at this time")
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]string{
		"id": user.ID.String(),
		"created_at": user.CreatedAt.String(),
		"updated_at": user.UpdatedAt.String(),
		"email": user.Email,
	})
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

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	RespondOK(w, r)
}

func chripyValidator(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	var p parameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		if err := respondWithError(w, 500, "Something went wrong"); err != nil {
			fmt.Println("Could not respond to request")
		}
		return
	}
	if len(p.Body) > 140 {
		if err := respondWithError(w, 400, "Chirp is too long"); err != nil {
			fmt.Println("Could not respond to request")
		}
		return
	}
	if err := respondWithJSON(w, 200, map[string]string{"cleaned_body":p.Body}); err != nil {
		fmt.Println("Could not respond to request")
	}
}

func badWordsReplacementMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}
		var p parameters
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&p); err != nil {
			if err := respondWithError(w, 500, "Something went wrong"); err != nil {
				fmt.Println("Could not respond to request")
			}
			return
		}
		badWords := []string{"kerfuffle", "sharbert", "fornax"}
		for _, badWord := range badWords {
			re := regexp.MustCompile(fmt.Sprintf(`(?i)\b%s\b`, regexp.QuoteMeta(badWord))) // QuoteMeta escapes special regex chars
			replacement := strings.Repeat("*", 4)
			p.Body = re.ReplaceAllString(p.Body, replacement)
		}
		modifiedBodyBytes, err := json.Marshal(p)
		if err != nil {
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
			fmt.Printf("Error encoding JSON: %v", err)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(modifiedBodyBytes))
		r.ContentLength = int64(len(modifiedBodyBytes))

		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("failed to connect to db: %w", err)
		return
	}
	
	apiCfg := apiConfig{}
	apiCfg.dbQueries = database.New(db)
	apiCfg.plataform = os.Getenv("PLATAFORM")

	fileserverHandler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	respondOkHandler := http.HandlerFunc(RespondOK)


	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileserverHandler))
	mux.HandleFunc("GET /admin/healthz", HandleHealth)
	mux.HandleFunc("GET /admin/metrics", apiCfg.showMetrics)
	mux.Handle("POST /admin/reset", apiCfg.resetMetricsMiddleware(respondOkHandler))
	mux.Handle("POST /api/validate_chirp", badWordsReplacementMiddleware(http.HandlerFunc(chripyValidator)))
	mux.Handle("POST /api/users", http.HandlerFunc(apiCfg.createUser))

	server := http.Server{}
	server.Handler = mux
	server.Addr = ":8080"
	fmt.Printf("Starting server on http://localhost%v/\n", server.Addr)
	server.ListenAndServe();
	
}