package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/hyphengolang/websockets"

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

	c := websockets.NewClient(
		websockets.WithRedis(rc),
	)

	c.On("join", func(p websockets.ResponseWriter, b *websockets.Payload) {
		v, _ := b.Data.MarshalJSON()

		if err := p.Publish(b.Method, v); err != nil {
			log.Printf("publish err: %v", err)
		}
	})

	// join a chat room
	mux.Get("/play", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("id")

		// defer removing client from db
		// this works because ServeHTTP is blocking

		ctx := websockets.NewContext(r.Context(), roomID)
		r = r.WithContext(ctx)

		c.ServeHTTP(w, r) // blocking statement

		// runs when client disconnects
		log.Print("client disconnected")
	})

	srv := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	log.Printf("listening on %s", srv.Addr)
	return srv.ListenAndServe()
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
