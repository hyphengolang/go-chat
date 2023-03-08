package games

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hyphengolang/websockets"
	"github.com/redis/go-redis/v9"
)

var _ http.Handler = (*Service)(nil)

type Service struct {
	// m is a HTTP multiplexer
	m chi.Router
	// c is a websocket client
	c *websockets.Client
}

// ServeHTTP implements http.Handler
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func NewService(opts ...Option) *Service {
	s := Service{
		m: chi.NewRouter(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	if s.c == nil {
		// if redis is not specified, use in-memory
		s.c = websockets.NewClient()
	}

	s.routes()
	return &s
}

func (s *Service) routes() {
	// HTTP routes
	s.m.Get("/", s.handleHello())
	s.m.Get("/play", s.handleP2PConn())

	// websocket routes
	s.c.On("join", s.onJoinRoom())
}

func (s *Service) handleP2PConn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// channel is required else the websocket client
		// will error
		channel := r.URL.Query().Get("id")
		if channel == "" {
			channel = "default"
		}

		ctx := websockets.NewContext(r.Context(), channel)
		r = r.WithContext(ctx)

		s.c.ServeHTTP(w, r)
	}
}

func (s *Service) handleHello() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}
}

func (s *Service) onJoinRoom() websockets.HandlerFunc {
	return func(w websockets.ResponseWriter, p *websockets.Payload) {
		v, _ := p.Data.MarshalJSON()

		w.Publish(p.Method, v)
	}
}

type Option func(*Service)

func WithRedis(r *redis.Client) Option {
	return func(s *Service) {
		s.c = websockets.NewClient(
			websockets.WithRedis(r),
		)
	}
}
