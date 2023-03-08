package games

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/hyphengolang/services"
	"github.com/hyphengolang/websockets"
	"github.com/redis/go-redis/v9"
)

var _ http.Handler = (*Service)(nil)

type Service struct {
	// m is a HTTP multiplexer
	m services.Router
	// c is a websocket client
	c *websockets.Client
	// r is a repository
	r *Repo
}

// ServeHTTP implements http.Handler
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}

func NewService(opts ...Option) *Service {
	s := Service{
		m: services.NewRouter(),
	}

	for _, opt := range opts {
		opt(&s)
	}

	if s.r == nil {
		s.r = NewRepo()
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
	{
		s.m.Post("/", s.handleNewGame())
		s.m.Delete("/", s.handleCloseGame())
		s.m.Get("/play", s.handleP2PConn())
	}

	// websocket routes
	{
		s.c.On("join", s.onJoinRoom())
	}
}

func (s *Service) handleNewGame() http.HandlerFunc {
	type response struct {
		ID uuid.UUID `json:"id"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		g := Game{
			ID:      uuid.New(),
			Players: [2]*Player{},
		}

		if err := s.r.Insert(&g); err != nil {
			s.m.Respond(w, r, err, http.StatusInternalServerError)
			return
		}

		res := response{
			ID: g.ID,
		}

		s.m.Respond(w, r, res, http.StatusOK)
	}
}

func (s *Service) handleCloseGame() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.m.Respond(w, r, "close game", http.StatusOK)
	}
}

// TODO -- Auth would be a way of retrieving the player ID
// for now will just generate one randomly
func (s *Service) handleP2PConn() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gameId, err := uuid.Parse(r.URL.Query().Get("id"))
		if err != nil {
			s.m.Respond(w, r, err, http.StatusBadRequest)
			return
		}

		found, err := s.r.Find(gameId)
		if err != nil {
			s.m.Respond(w, r, err, http.StatusNotFound)
			return
		}

		if found.Players[0] != nil && found.Players[1] != nil {
			s.m.Respond(w, r, "game is full", http.StatusNotFound)
			return
		}

		playerId := uuid.New()
		{
			if found.Players[0] == nil {
				found.Players[0] = &Player{
					ID:        playerId,
					IPAddress: ReadUserIP(r),
				}
			} else {
				found.Players[1] = &Player{
					ID:        playerId,
					IPAddress: ReadUserIP(r),
				}
			}
		}

		// on connect send a push to player only
		// s.c.Push("join", found.ID.String(), playerId.String())

		// defer remove player from game
		defer func() {
			if found.Players[0] != nil && found.Players[0].ID == playerId {
				found.Players[0] = nil
			} else if found.Players[1] != nil && found.Players[1].ID == playerId {
				found.Players[1] = nil
			}
		}()

		ctx := websockets.NewContext(r.Context(), found.ID.String())
		r = r.WithContext(ctx)

		s.c.ServeHTTP(w, r)
	}
}

// send up-to-date game state
func (s *Service) onJoinRoom() websockets.HandlerFunc {
	type request struct {
		GameID uuid.UUID `json:"gameId"`
	}

	type response struct {
		Game *Game `json:"game"`
	}

	return func(w websockets.ResponseWriter, p *websockets.Payload) {
		var req request
		if err := json.Unmarshal(p.Data, &req); err != nil {
			log.Printf("error unmarshalling request: %v", err)
			return
		}

		found, _ := s.r.Find(req.GameID)

		b, err := json.Marshal(response{Game: found})
		if err != nil {
			log.Printf("error marshalling response: %v", err)
			return
		}

		w.Publish(p.Method, b)
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

// util
func ReadUserIP(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}
