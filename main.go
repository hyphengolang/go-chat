package main

import (
	"context"
	"go-chat/websocket"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func run() error {
	opt, err := redis.ParseURL("redis://localhost:6379")
	if err != nil {
		return err
	}
	rc := redis.NewClient(opt)
	defer rc.Close()

	if err := rc.Ping(context.Background()).Err(); err != nil {
		return err
	}

	mux := chi.NewRouter()

	// simple hello world
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	c := websocket.NewClient(
		websocket.WithRedis(rc),
	)

	// join a chat room
	mux.Get("/play", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("id")

		// defer removing client from db
		// this works because ServeHTTP is blocking

		ctx := websocket.NewContext(r.Context(), roomID)
		r = r.WithContext(ctx)

		c.ServeHTTP(w, r) // blocking statement

		// runs when client disconnects
		log.Print("client disconnected")
	})

	srv := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Printf("listening on %s", srv.Addr)
	return srv.ListenAndServe()
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("go-chat: ")
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("error: %v", err)
	}
}
