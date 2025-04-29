package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/google/uuid"
	"github.com/wilgnert/chirpy/internal/auth"
	"github.com/wilgnert/chirpy/internal/database"
)

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string `json:"body"`
	}
	var p parameters
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	if err := decoder.Decode(&p); err != nil {
		fmt.Println(err.Error())
		if err := respondWithError(w, http.StatusInternalServerError, "Something went wrong"); err != nil {
			fmt.Println("Could not respond to request")
		}
		return
	}
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	parsedID, err := auth.ValidateJWT(bearerToken, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{Body: p.Body, UserID: parsedID})
	if err != nil {
		fmt.Println(err.Error())
		respondWithError(w, http.StatusInternalServerError, "could not create chirp at this time")
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]string{
		"id":         chirp.ID.String(),
		"created_at": chirp.CreatedAt.String(),
		"updated_at": chirp.UpdatedAt.String(),
		"body":       chirp.Body,
		"user_id":    parsedID.String(),
	})
}

func (cfg *apiConfig) getAllChirps(w http.ResponseWriter, r *http.Request) {
	var chirps []database.Chirp
	var err error

	if authorId := r.URL.Query().Get("author_id"); authorId != "" {
		parsed, err := uuid.Parse(authorId)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "author not found")
		}
		chirps, err = cfg.dbQueries.GetAllChirpsFromAuthorID(r.Context(), parsed)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not retrieve chirps")
			return
		}
	} else {
		chirps, err = cfg.dbQueries.GetAllChirps(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "could not retrieve chirps")
			return
		}
	}

	if sortParam := r.URL.Query().Get("sort"); sortParam != "" {
		if sortParam == "desc" {
			sort.Slice(chirps, func(i, j int) bool {
				return chirps[i].CreatedAt.After(chirps[j].CreatedAt)
			})
		}
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) getChirpByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not retrieve chirp")
		return
	}
	
	chirp, err := cfg.dbQueries.GetChirpByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not retrieve chirp")
		return
	}
	respondWithJSON(w, http.StatusOK, chirp)
}

func (cfg *apiConfig) deleteChirpByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not retrieve chirp")
		return
	}
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	parsedID, err := auth.ValidateJWT(bearerToken, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	chirp, err := cfg.dbQueries.GetChirpByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not retrieve chirp")
		return
	}
	if parsedID.String() != chirp.UserID.String() {
		respondWithError(w, http.StatusForbidden, "you are not allowed")
		return		
	}
	err = cfg.dbQueries.DeleteChirpByID(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not delete chirp")
	}
	
	RespondNoContent(w, r);
}
