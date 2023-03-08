package main

import (
	"context"
	"flag"
	"fmt"
	"go-chat/games"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

func run() error {
	rc, err := newRedisConnection(context.Background(), "redis:6379")
	if err != nil {
		return err
	}
	defer rc.Close()

	mux := chi.NewRouter()
	{
		// simple logging for HTTP requests
		mux.Use(middleware.Logger)
	}
	// mounting services
	{
		mux.Mount("/games", newGamesService(rc))
	}

	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Printf("listening on %s", srv.Addr)
	return srv.ListenAndServe()
}

func newRedisConnection(ctx context.Context, url string) (*redis.Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	rc := redis.NewClient(opt)
	if err := rc.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return rc, nil
}

func newGamesService(rc *redis.Client) http.Handler {
	return games.NewService(games.WithRedis(rc))
}

var port int

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("go-chat: ")

	flag.IntVar(&port, "port", 8080, "port to listen on")
	flag.Parse()
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
