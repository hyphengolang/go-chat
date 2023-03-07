package websocket

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func read(conn *connHandler, c *Client) {
	defer func() {
		conn.Close()
		c.unregister <- conn
	}()

	// registrer connection
	c.register <- conn

	for {
		p, err := wsutil.ReadClientText(conn.rwc)
		if err != nil {
			log.Printf("read err: %v", err)
			return
		}

		response := fmt.Sprintf("%s:%s", conn.channel, p)

		// TODO -- broadcast to channel only
		c.broadcast <- &Message{conn.channel, response}
	}
}

func write(conn *connHandler) {
	defer conn.Close()

	for p := range conn.rcv {
		err := wsutil.WriteServerText(conn.rwc, []byte(p.text))
		if err != nil {
			log.Printf("write err: %v", err)
			return
		}
	}
}

var _ http.Handler = (*Client)(nil)

// Client is a websocket client
// currently only supports text messages
type Client struct {
	// u upgrades the HTTP request to a websocket connection
	u ws.HTTPUpgrader
	// broadcast is the channel that receives messages from the server
	broadcast chan *Message
	// register is the channel that registers new connections
	register chan *connHandler
	// unregister is the channel that unregisters connections
	unregister chan *connHandler
	// connections is the list of connections
	connections map[*connHandler]bool
}

// ServeHTTP implements http.Handler
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rwc, _, _, err := c.u.Upgrade(r, w)
	if err != nil {
		return
	}

	channel, ok := FromContext(r.Context())
	if !ok {
		channel = "default"
	}

	conn := connHandler{
		rwc:     rwc,
		channel: channel,
		rcv:     make(chan *Message, 256),
	}

	go read(&conn, c)
	go write(&conn)
}

func (c *Client) listen() {
	for {
		select {
		case conn := <-c.register:
			c.connections[conn] = true
		case conn := <-c.unregister:
			// conn should have already been closed
			delete(c.connections, conn)
		case msg := <-c.broadcast:
			// TODO -- redis subscribe goes here
			for conn := range c.connections {
				select {
				case conn.rcv <- msg:
				default:
					conn.Close()
					delete(c.connections, conn)
				}
			}
		}
	}
}

func NewClient() *Client {
	c := Client{
		broadcast:   make(chan *Message, 256),
		register:    make(chan *connHandler),
		unregister:  make(chan *connHandler),
		connections: make(map[*connHandler]bool),
	}
	go c.listen()
	return &c
}

type connHandler struct {
	// rwc is the underlying websocket connection
	rwc net.Conn
	// channel that the connection is subscribed to
	channel string
	// rcv is the channel that receives messages from the connection
	rcv chan *Message
}

func (c *connHandler) Close() error {
	close(c.rcv)
	return c.rwc.Close()
}

type Message struct {
	channel string
	text    string
}

type Context string

var (
	channelKey Context = "channel"
)

func NewContext(ctx context.Context, channel string) context.Context {
	return context.WithValue(ctx, channelKey, channel)
}

func FromContext(ctx context.Context) (string, bool) {
	channel, ok := ctx.Value(channelKey).(string)
	return channel, ok
}
