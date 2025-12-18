package main

import (
	"net/http"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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

func handlerValidate(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Body string `jason:"body"`
	}
	type vals struct {
		CleanedBody string `json:"cleaned_body"`
	} 

	decoder := json.NewDecoder(r.Body)
	chirp := params{}
	err := decoder.Decode(&chirp)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't decode params", err)
		return
	}
	
	const maxLength = 140
	if len(chirp.Body) > maxLength {
		errorResponse(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	words := strings.Split(chirp.Body, " ")

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

	jsonResponse(w, http.StatusOK, vals{
		CleanedBody: cleaned,
	})
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Email string `json:"email"`
	}
	type response struct {
		User
	}
	
	decoder := json.NewDecoder(r.Body)
	userEmail := params{}
	err := decoder.Decode(&userEmail)
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Couldn't decode params", err)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), userEmail.Email)
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