package main

import (
	"net/http"
	"log"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	const rootPath = "."
	const port = "8080"
	
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", apiCfg.middlewareMetricInc(http.StripPrefix("/app", http.FileServer(http.Dir(rootPath)))))
	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerServerHits)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerResetHits)

	serverStruct := &http.Server{
		Handler: serveMux,
		Addr: ":" + port,
	}

	log.Printf("Serving files from %s on port: %s\n", rootPath, port)
	log.Fatal(serverStruct.ListenAndServe())

}

