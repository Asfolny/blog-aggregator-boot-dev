package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")

	mux := http.NewServeMux()

	// Test the respondWithJSON function
	mux.HandleFunc("GET /v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		respondWithJSON(w, 200, map[string]string{"status": "ok"})
	})

	// Test the respondWithError function
	mux.HandleFunc("GET /v1/err", func(w http.ResponseWriter, r *http.Request) {
		respondWithError(w, 500, "Internal Server Error")
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		WriteTimeout:      5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	fmt.Printf("Server starting on %v\n", server.Addr)

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(code)
	_, writeErr := w.Write(dat)

	if writeErr != nil {
		log.Printf("Write failes: %v", err)
	}
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code >= 500 {
		log.Printf("Responding with 5XX error: %s", msg)
	}

	type errorResponse struct {
		Error string `json:"error"`
	}

	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}
