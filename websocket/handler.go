package websocket

import (
	"encoding/json"
)

type Handler interface {
	Serve(w ResponseWriter, r *Request)
}

type HandlerFunc func(w ResponseWriter, r *Request)

func (c *Client) On(method string, h Handler) {
	if c.m == nil {
		panic("websocket: client mux is nil")
	}

	e := muxEntry{method, h}

	if _, ok := c.m[method]; ok {
		panic("websocket: client mux entry already exists")
	}

	c.m[method] = e
}

func (c *Client) match(method string) (Handler, bool) {
	if c.m == nil {
		panic("websocket: client mux is nil")
	}

	e, ok := c.m[method]
	if !ok {
		return nil, false
	}

	return e.h, true
}

type muxEntry struct {
	method string
	h      Handler
}

type ResponseWriter interface {
	Publish([]byte) error
}

var _ ResponseWriter = (*response)(nil)

type response struct {
	ps PSubcriber
}

func (r *response) Publish(b []byte) error {
	panic("implement me")
}

type Request struct {
	Typ  string          `json:"type"`
	Body json.RawMessage `json:"payload"`
}
