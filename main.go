package main

import (
	"github.com/simplyluckyy/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"net/http"
	"log"
	"sync/atomic"
	"os"
	"database/sql"
	"time"
	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
	platform string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Password  string	`json:"-"`
}

type Chirp struct {
	ID		  uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID	  uuid.UUID `json:"user_id"`
	Body	  string	`json:"body"`

}


func main() {
	godotenv.Load()

	const rootPath = "."
	const port = "8080"
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	db, err := sql.Open("postgres", dbURL)
	
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}
	dbQueries := database.New(db)
	dbPlatform := os.Getenv("PLATFORM")
	
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db: dbQueries,
		platform: dbPlatform,
	}

	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", apiCfg.middlewareMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(rootPath)))))
	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerServerHits)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerResetHits)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpID)
	serveMux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	serveMux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	serverStruct := &http.Server{
		Handler: serveMux,
		Addr: ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", rootPath, port)
	log.Fatal(serverStruct.ListenAndServe())

}

