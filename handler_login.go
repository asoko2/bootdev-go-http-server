package main

import (
	"encoding/json"
	"fmt"
	"go-http-server/internal/auth"
	"go-http-server/internal/database"
	"net/http"
	"time"
)

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameter struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to decode request body", err)
		return
	}

	// Authenticate the user
	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password", err)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password", err)
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate JWT", err)
		return
	}

	refreshToken := auth.MakeRefreshToken()

	dbRefreshToken, err := cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
	})

	respondWithJSON(w, http.StatusOK, response{
		User: User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
			Token:     accessToken,
		},
		Token:        accessToken,
		RefreshToken: dbRefreshToken.Token,
	})
}

func (cfg *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	type resposne struct {
		Token string `json:"token"`
	}

	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing bearer token", err)
		return
	}

	user, err := cfg.db.GetUserFromRefreshToken(r.Context(), bearerToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing refresh token", err)
		return
	}

	newToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate JWT", err)
		return
	}

	respondWithJSON(w, http.StatusOK, resposne{
		Token: newToken,
	})
}

func (cfg *apiConfig) handlerRevokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing bearer token", err)
		return
	}

	fmt.Println("revoking for token : ", bearerToken)

	_, err = cfg.db.RevokeRefreshToken(r.Context(), bearerToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke refresh token", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}
