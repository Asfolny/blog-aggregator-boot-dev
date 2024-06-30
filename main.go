package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/Asfolny/blog-aggregator-boot-dev/internal/database"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	dbURL := os.Getenv("DB_CONN")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	dbQueries := database.New(db)
	ctx := context.Background()

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

	mux.HandleFunc("POST /v1/users", func(w http.ResponseWriter, r *http.Request) {
		type createUserInput struct {
			Name   string `json:"name"`
			ApiKey string `json:"api_key"`
		}
		var input createUserInput

		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			respondWithError(w, 400, "Invalid body to create user")
			return
		}

		uuid := uuid.New()
		user, err := dbQueries.CreateUser(
			ctx,
			database.CreateUserParams{ID: uuid, Name: input.Name, ApiKey: input.ApiKey},
		)
		if err != nil {
			respondWithError(w, 500, "Failed to create user")
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		respondWithJSON(w, 201, user)
	})

	mux.HandleFunc("GET /v1/users", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		splitAuth := strings.Split(authHeader, " ")

		if len(splitAuth) < 2 {
			respondWithError(
				w,
				400,
				"To get get user, please send Authorization: ApiKey <KEY> header",
			)
			return
		}

		if splitAuth[0] != "ApiKey" {
			respondWithError(
				w,
				400,
				"To get get user, please send Authorization: ApiKey <KEY> header",
			)
			return
		}

		user, err := dbQueries.GetUserByApiKey(
			ctx,
			splitAuth[1],
		)
		if err != nil {
			respondWithError(w, 404, "Failed to find user by API Key")
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		respondWithJSON(w, 201, user)
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
		fmt.Fprintf(os.Stderr, "Error marshalling JSON: %s\n", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(code)
	_, writeErr := w.Write(dat)

	if writeErr != nil {
		fmt.Fprintf(os.Stderr, "Write failes: %v\n", err)
	}
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code >= 500 {
		fmt.Fprintf(os.Stderr, "Responding with 5XX error: %s\n", msg)
	}

	type errorResponse struct {
		Error string `json:"error"`
	}

	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

type apiConfig struct {
	DB *database.Queries
}
