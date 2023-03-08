package games

import (
	"fmt"
	"go-chat/pkg/structures"

	"github.com/google/uuid"
)

type Value int

const (
	Noughts Value = iota + 1
	Crosses
)

type Player struct {
	ID uuid.UUID `json:"id"`
	// Name string    `json:"name"`
	// Value can either be 1 (noughts) or 2 (crosses)
	Value Value `json:"value"`
	// IPAddress is used internally to identify the player
	IPAddress string `json:"-"`
}

type Game struct {
	ID uuid.UUID `json:"id"`
	// Name string `json:"name"`
	Players [2]*Player `json:"players"`
}

// in-memory storage
type Repo struct {
	// games is a map of games
	games *structures.SyncMap[uuid.UUID, *Game]
}

func NewRepo() *Repo {
	r := &Repo{
		games: structures.NewSyncMap[uuid.UUID, *Game](),
	}
	return r
}

// Insert inserts a game into the repo
func (r *Repo) Insert(g *Game) error {
	r.games.Store(g.ID, g)
	return nil
}

func (r *Repo) Find(id uuid.UUID) (*Game, error) {
	g, ok := r.games.Load(id)
	if !ok {
		return nil, ErrGameNotFound
	}
	return g, nil
}

// RemovePlayer removes a player from a game
func (r *Repo) RemovePlayer(gameId, playerId uuid.UUID) {}

var ErrGameNotFound = fmt.Errorf("game not found")
