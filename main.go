package main

import (
	"github.com/SimplyLuckyy/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"net/http"
	"log"
	"sync/atomic"
	"os"
	"database/sql"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db *database.Queries
}

func main() {
	const rootPath = "."
	const port = "8080"
	dbURL := os.Getenv("DB_URL")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)
	
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db: dbQueries,
	}

	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", apiCfg.middlewareMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(rootPath)))))
	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerServerHits)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerResetHits)
	serveMux.HandleFunc("POST /api/validate_chirp", handlerValidate)

	serverStruct := &http.Server{
		Handler: serveMux,
		Addr: ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", rootPath, port)
	log.Fatal(serverStruct.ListenAndServe())

}

