package main

import (
	"net/http"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"errors"
	"github.com/google/uuid"

	"github.com/simplyluckyy/chirpy/internal/database"
	"github.com/simplyluckyy/chirpy/internal/auth"
)

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) middlewareMetricInc(next http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerServerHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`
<html>

<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
</body>

</html>
	`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerResetHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	cfg.db.Reset(r.Context())
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func validateChirp(body string) (string, error) {
	
	const maxLength = 140
	if len(body) > maxLength {
		return "", errors.New("Chirp is too long")
	}

	words := strings.Split(body, " ")

	for i, word := range words {
		if (strings.ToLower(word) == "kerfuffle") {
			words[i] = "****"
		} else if (strings.ToLower(word) == "sharbert") {
			words[i] = "****"
		} else if (strings.ToLower(word) == "fornax") {
			words[i] = "****"
		}
	}

	cleaned := strings.Join(words, " ")

	return cleaned, nil
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	type response struct {
		User
	}
	
	decoder := json.NewDecoder(r.Body)
	userParams := params{}
	err := decoder.Decode(&userParams)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't decode params", err)
		return
	}

	hashedPassword, err := auth.HashPassword(userParams.Password)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't hash password", err)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email: userParams.Email,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't create user", err)
		return
	}
	
	jsonResponse(w, http.StatusCreated, response{
		User: User{
			ID: user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email: user.Email,
		},
	})
	
}

func (cfg *apiConfig) handlerCreateChirp(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	uncleaned := params{}
	err := decoder.Decode(&uncleaned)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't decode params", err)
		return
	}

	cleaned, err := validateChirp(uncleaned.Body)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	
	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body: cleaned,
		UserID: uncleaned.UserID,
	})
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't create chirp", err)
		return
	}

	jsonResponse(w, http.StatusCreated, Chirp{
		ID:		   chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:	   chirp.Body,
		UserID:	   chirp.UserID,
	})
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't get chirps", err)
		return
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID: dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			UserID: dbChirp.UserID,
			Body: dbChirp.Body,
		})
	}

	jsonResponse(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) handlerGetChirpID(w http.ResponseWriter, r *http.Request) {
	chirpIDstring := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDstring)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		errorResponse(w, http.StatusNotFound, "Couldn't find chipr", err)
		return
	}

	jsonResponse(w, http.StatusOK, Chirp{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		UserID: chirp.UserID,
		Body:  chirp.Body,
	})
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Password string `json:"password"`
		Email string `json:"email"`
	}
	type response struct {
		User
	}

	decoder := json.NewDecoder(r.Body)
	userParams := params{}
	err := decoder.Decode(&userParams)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't decode params", err)
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), userParams.Email)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Incorrect Email or Password", err)
		return
	}

	match, err := auth.CheckPasswordHash(userParams.Password, user.HashedPassword)
	if err != nil || !match {
		errorResponse(w, http.StatusUnauthorized, "Incorrect Email or Password", err)
		return
	}

	jsonResponse(w, http.StatusOK, response{
		User: User{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	})
}

func errorResponse(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {log.Println(err)}
	if code > 499 {log.Printf("Responding with 5XX eeror: %s", msg)}
	
	type errorJson struct {
		Error string `json:"error"`
	}
	jsonResponse(w, code, errorJson{
		Error: msg,
	})
}

func jsonResponse(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(code)
	w.Write(dat)
}