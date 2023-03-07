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
		c.pubsub.unregister <- conn
	}()

	// registrer connection
	c.pubsub.register <- conn

	for {
		p, err := wsutil.ReadClientText(conn.rwc)
		if err != nil {
			log.Printf("read err: %v", err)
			return
		}

		// c.broadcaster.Publish(conn)
		response := fmt.Sprintf("%s:%s", conn.channel, p)

		// TODO -- broadcast to channel only
		c.pubsub.broadcast <- &Message{conn.channel, response}
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
	// pubsub handles publishing messages to all connections
	pubsub *PubSub
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

func NewClient() *Client {
	c := Client{
		pubsub: NewPubSub(),
	}
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
