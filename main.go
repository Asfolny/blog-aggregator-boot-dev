package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
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
	cfg := apiConfig{dbQueries}
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

	mux.HandleFunc(
		"GET /v1/users",
		cfg.middlewareAuth(func(w http.ResponseWriter, r *http.Request, user database.User) {
			respondWithJSON(w, 201, user)
		}),
	)

	mux.HandleFunc(
		"POST /v1/feeds",
		cfg.middlewareAuth(func(w http.ResponseWriter, r *http.Request, user database.User) {
			type createFeedsInput struct {
				Name string `json:"name"`
				Url  string `json:"url"`
			}

			var input createFeedsInput

			err := json.NewDecoder(r.Body).Decode(&input)
			if err != nil {
				respondWithError(w, 400, "Invalid body to create feed")
				return
			}

			uuid := uuid.New()
			feed, err := cfg.DB.CreateFeed(
				ctx,
				database.CreateFeedParams{
					ID:     uuid,
					Name:   input.Name,
					Url:    input.Url,
					UserID: user.ID,
				},
			)
			if err != nil {
				respondWithError(w, 500, "Failed to create feed")
				fmt.Fprintf(os.Stderr, "%v\n", err)
				return
			}

			// Create a new feed follow for the user who added the feed
			feedFollow, err := cfg.DB.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
				FeedID: feed.ID,
				UserID: user.ID,
			})

			respondWithJSON(
				w,
				201,
				map[string]interface{}{"feed": feed, "feed_folloow": feedFollow},
			)
		}),
	)

	mux.HandleFunc("GET /v1/feeds", func(w http.ResponseWriter, r *http.Request) {
		feeds, err := cfg.DB.GetFeeds(ctx)
		if err != nil {
			respondWithError(w, 500, "Failed to get all feeds")
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		respondWithJSON(w, 200, feeds)
	})

	mux.HandleFunc(
		"POST /v1/feed_follow",
		cfg.middlewareAuth(func(w http.ResponseWriter, r *http.Request, user database.User) {
			type feedFollowCreateInput struct {
				FeedId uuid.UUID `json:"feed_id"`
			}
			var input feedFollowCreateInput

			err := json.NewDecoder(r.Body).Decode(&input)
			if err != nil {
				respondWithError(w, 400, "Invalid body to create feed follow")
				return
			}

			feedFollow, err := cfg.DB.CreateFeedFollow(
				ctx,
				database.CreateFeedFollowParams{
					FeedID: input.FeedId,
					UserID: user.ID,
				},
			)
			if err != nil {
				respondWithError(w, 500, "Failed to create feed follow")
				fmt.Fprintf(os.Stderr, "%v\n", err)
				return
			}

			respondWithJSON(w, 201, feedFollow)
		}),
	)

	mux.HandleFunc(
		"DELETE /v1/feed_follow/{feedId}",
		cfg.middlewareAuth(func(w http.ResponseWriter, r *http.Request, user database.User) {
			feedId, err := uuid.Parse(r.PathValue("feedId"))
			if err != nil {
				respondWithError(w, 404, "Feed ID not found")
				return
			}

			err = cfg.DB.DeleteFeedFollow(
				ctx,
				database.DeleteFeedFollowParams{
					FeedID: feedId,
					UserID: user.ID,
				},
			)
			if err != nil {
				respondWithError(w, 500, "Failed to delete feed follow")
				fmt.Fprintf(os.Stderr, "%v\n", err)
				return
			}

			respondWithJSON(w, 202, map[string]string{"message": "DONE"})
		}),
	)

	mux.HandleFunc(
		"GET /v1/feed_follow",
		cfg.middlewareAuth(func(w http.ResponseWriter, r *http.Request, user database.User) {
			feeds, err := cfg.DB.AllFeedFollowsByUser(ctx, user.ID)
			if err != nil {
				respondWithError(w, 404, "Failed to find your feed follows")
				return
			}

			respondWithJSON(w, 200, feeds)
		}),
	)

	mux.HandleFunc(
		"GET /v1/posts",
		cfg.middlewareAuth(func(w http.ResponseWriter, r *http.Request, user database.User) {
			posts, err := cfg.DB.GetPostsByUser(ctx, user.ID)
			if err != nil {
				respondWithError(w, 404, "Failed to find user's interested posts")
				return
			}

			respondWithJSON(w, 200, posts)
		}),
	)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		WriteTimeout:      5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go scraper(10, cfg)

	fmt.Printf("Server starting on %v\n", server.Addr)

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func scraper(limit int32, apiCfg apiConfig) {
	fmt.Println("Starting scraper worker")
	ctx := context.Background()

	for {
		feeds, err := apiCfg.DB.GetNextFeedsToFetch(ctx, limit)
		if err != nil {
			log.Fatalln(err)
		}

		var wg sync.WaitGroup
		for _, feed := range feeds {
			wg.Add(1)
			go scrapeFeed(feed.Url, feed.ID, apiCfg.DB)
		}

		wg.Wait()

		time.Sleep(10 * time.Second)
	}
}

func scrapeFeed(url string, feedId uuid.UUID, db *database.Queries) {
	response, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch feed %s\n", url)
		return
	}
	defer response.Body.Close()

	type RssChannelItem struct {
		XMLName     xml.Name `xml:"item"`
		Title       string   `xml:"title"`
		Link        string   `xml:"link"`
		PubDate     string   `xml:"pubDate"`
		Description string   `xml:"description"`
	}

	type RssFeed struct {
		RssChannel struct {
			XMLName xml.Name         `xml:"channel"`
			Title   string           `xml:"title"`
			Items   []RssChannelItem `xml:"item"`
		} `xml:"channel"`
	}

	dat, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get the full body of %v; %v\n", url, err)
		return
	}

	var xmlResp RssFeed
	err = xml.Unmarshal(dat, &xmlResp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse xml %v\n", err)
		return
	}

	for _, item := range xmlResp.RssChannel.Items {
		pubDate, timeParseErr := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", item.PubDate)
		publishedAt := sql.NullTime{Time: pubDate, Valid: true}

		if timeParseErr != nil {
			publishedAt.Valid = false
		}

		_, err = db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			Title:       item.Title,
			Url:         item.Link,
			Description: sql.NullString{String: item.Description, Valid: true},
			FeedID:      feedId,
			PublishedAt: publishedAt,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
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

func (cfg *apiConfig) middlewareAuth(handler authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		splitAuth := strings.Split(authHeader, " ")

		if len(splitAuth) < 2 || splitAuth[0] != "ApiKey" {
			respondWithError(
				w,
				400,
				"To get get user, please send Authorization: ApiKey <KEY> header",
			)
			return
		}

		user, err := cfg.DB.GetUserByApiKey(context.Background(), splitAuth[1])
		if err != nil {
			respondWithError(w, 404, "Failed to find user by API Key")
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}

		handler(w, r, user)
	}
}

type authedHandler func(http.ResponseWriter, *http.Request, database.User)
