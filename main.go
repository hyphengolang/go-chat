package main

import (
	"go-chat/websocket"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func run() error {
	mux := chi.NewRouter()

	// simple hello world
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	c := websocket.NewClient()

	// create a new chat room

	// join a chat room
	mux.Get("/play", func(w http.ResponseWriter, r *http.Request) {
		roomID := "#123"

		ctx := websocket.NewContext(r.Context(), roomID)
		r = r.WithContext(ctx)

		c.ServeHTTP(w, r)
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
