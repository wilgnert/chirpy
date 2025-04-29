package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

func chripyValidatorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body   string `json:"body"`
			UserID string `json:"user_id"`
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

func badWordsReplacementMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body   string `json:"body"`
			UserID string `json:"user_id"`
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

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) resetMetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.plataform != "dev" {
			respondWithError(w, http.StatusForbidden, "403 Forbidden")
			return
		}
		if err := cfg.dbQueries.DeleteAllRefreshTokens(r.Context()); err != nil {
			fmt.Println("error deleting all refresh tokens", err.Error())
		}

		if err := cfg.dbQueries.DeleteAllChirps(r.Context()); err != nil {
			fmt.Println("error deleting all chirps", err.Error())
		}
		if err := cfg.dbQueries.DeleteAllUsers(r.Context()); err != nil {
			fmt.Println("error deleting all users", err.Error())
		}
		cfg.fileserverHits.Store(0)
		next.ServeHTTP(w, r)
	})
}
