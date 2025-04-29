package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/wilgnert/chirpy/internal/auth"
	"github.com/wilgnert/chirpy/internal/database"
)

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
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
	pass, _ := auth.HashPassword(p.Password)
	user, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{Email: p.Email, HashedPassword: pass})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create user at this time")
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id": user.ID.String(),
		"created_at": user.CreatedAt.String(),
		"updated_at": user.UpdatedAt.String(),
		"email": user.Email,
		"is_chirpy_red": user.ChirpyRedExpiresAt.Valid && time.Now().Before(user.ChirpyRedExpiresAt.Time),
	})
}

func (cfg *apiConfig) login(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
		Password string `json:"password"`
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

	ExpiresInSeconds := 60 * 60
	
	
	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), p.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.secret, time.Duration(ExpiresInSeconds) * time.Second)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if err := auth.CheckPasswordHash(user.HashedPassword, p.Password); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
		return
	}
	refresh_token, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
  rfsh_tkn, err := cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{UserID: user.ID, Token: refresh_token, ExpiresAt: time.Now().Add(60 * 24 * time.Hour)})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id": user.ID.String(),
		"created_at": user.CreatedAt.String(),
		"updated_at": user.UpdatedAt.String(),
		"email": user.Email,
		"token": token,
		"refresh_token": rfsh_tkn.Token,
		"is_chirpy_red": user.ChirpyRedExpiresAt.Valid && time.Now().Before(user.ChirpyRedExpiresAt.Time),
	})
}

func (cfg *apiConfig) refresh(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, http.StatusUnauthorized, "missing Authorization header")
		return
	}
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		respondWithError(w, http.StatusUnauthorized, "invalid Authorization header format")
		return
	}
	token := authHeader[7:]
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "missing token in Authorization header")
		return
	}
	refresh_token, err := cfg.dbQueries.GetRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	if refresh_token.RevokedAt.Valid {
		respondWithError(w, http.StatusUnauthorized, "token was already revoked")
		return
	}
	if refresh_token.ExpiresAt.Before(time.Now()) {
		respondWithError(w, http.StatusUnauthorized, "token expired")
		return
	}
	new_token , err := auth.MakeJWT(refresh_token.UserID, cfg.secret, time.Duration(60 * 60) * time.Second)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	
	respondWithJSON(w, http.StatusOK, map[string]string{
		"token": new_token,
	})
}

func (cfg *apiConfig) revoke(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, http.StatusUnauthorized, "missing Authorization header")
		return
	}
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		respondWithError(w, http.StatusUnauthorized, "invalid Authorization header format")
		return
	}
	token := authHeader[7:]
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "missing token in Authorization header")
		return
	}
	_, err := cfg.dbQueries.RevokeRefreshToken(r.Context(), token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	RespondNoContent(w, r)
}

func (cfg *apiConfig) updateUserEmailAndPassword(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondWithError(w, http.StatusUnauthorized, "missing Authorization header")
		return
	}
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		respondWithError(w, http.StatusUnauthorized, "invalid Authorization header format")
		return
	}
	bearerToken := authHeader[7:]
	if bearerToken == "" {
		respondWithError(w, http.StatusUnauthorized, "missing token in Authorization header")
		return
	}
	id, err := auth.ValidateJWT(bearerToken, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	var u database.UpdateUserEmailAndPasswordRow
	var p struct{
		Email string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	if err := decoder.Decode(&p); err != nil {
		if err := respondWithError(w, http.StatusBadRequest, "could not parse request body"); err != nil {
			fmt.Println("Could not respond to request")
		}
		return
	}

	hashed, err := auth.HashPassword(p.Password)
	if  err != nil {
		if err := respondWithError(w, http.StatusBadRequest, "could not parse request body"); err != nil {
			fmt.Println("Could not respond to request")
		}
		return
	}
	u, err = cfg.dbQueries.UpdateUserEmailAndPassword(r.Context(), database.UpdateUserEmailAndPasswordParams{ID: id, Email: p.Email, HashedPassword: hashed})
	if err != nil {
		fmt.Println(err.Error())
		respondWithError(w, http.StatusNotFound, "could not find user to update credentials with the request")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]any{
		"id": u.ID.String(),
		"created_at": u.CreatedAt.String(),
		"updated_at": u.UpdatedAt.String(),
		"email": u.Email,
		"token": bearerToken,
		"is_chirpy_red": u.ChirpyRedExpiresAt.Valid && time.Now().Before(u.ChirpyRedExpiresAt.Time),
	})
}

func (cfg *apiConfig) handleWebhook(w http.ResponseWriter, r *http.Request) {
	key, err := auth.GetAPIKey(r.Header)
	if err != nil || key != cfg.polka_key {
		respondWithError(w, http.StatusUnauthorized, "faulty api key")
		return
	}
	var p struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	if err := decoder.Decode(&p); err != nil {
		if err := respondWithError(w, http.StatusBadRequest, "could not parse request body"); err != nil {
			fmt.Println("Could not respond to request")
		}
		return
	}
	switch p.Event {
	case "user.upgraded":
		parsed_id, err := uuid.Parse(p.Data.UserID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not parse user")
			return
		}
		_, err = cfg.dbQueries.UpdateUserChirpyRed(
			r.Context(), 
			database.UpdateUserChirpyRedParams{
				ID: parsed_id, 
				ChirpyRedExpiresAt: sql.NullTime{
					Time: time.Now().Add(30*24*time.Hour),
					Valid: true,
				},
			},
		)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not find user")
			return
		}
	default:
		break
	}
	RespondNoContent(w, r)
}