package websocket

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/redis/go-redis/v9"
)

type Message struct {
	Channel string
	Payload string
}

func read(conn *connHandler, ps PSubcriber) {
	defer conn.Close()

	for {
		p, err := wsutil.ReadClientText(conn.rwc)
		if err != nil {
			log.Printf("read err: %v", err)
			return
		}

		msg := &Message{
			Channel: conn.channel,
			Payload: fmt.Sprintf("%s:%s", conn.channel, p),
		}

		if err := ps.Publish(msg); err != nil {
			log.Printf("publish err: %v", err)
			return
		}
	}
}

// close, write
func write(conn *connHandler) {
	defer conn.Close()

	for p := range conn.rcv {
		if err := wsutil.WriteServerText(conn.rwc, []byte(p.Payload)); err != nil {
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
	// ps handles publishing messages to all connections
	ps PSubcriber
}

// ServeHTTP implements http.Handler
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	channel, ok := FromContext(r.Context())
	if !ok {
		http.Error(w, "channel not found", http.StatusBadRequest)
		return
	}

	if channel == "" {
		http.Error(w, "channel is empty", http.StatusBadRequest)
		return
	}

	rwc, _, _, err := c.u.Upgrade(r, w)
	if err != nil {
		return
	}

	conn := connHandler{
		rwc:     rwc,
		channel: channel,
		rcv:     make(chan *Message, 256),
	}

	// var rc *redis.Client
	// rc.Conn().Publish(r.Context(), channel, "hello")
	// rc.Subscribe(r.Context(), channel)

	// ps := NewSubscriber()

	unset := c.ps.Set(&conn)
	defer unset()

	// rwc, channelName, Publisher
	go read(&conn, c.ps)
	write(&conn) // I don't think this needs to be in a goroutine
}

func NewClient(r *redis.Client) *Client {
	c := Client{
		ps: NewRSubscriber(r),
	}
	return &c
}

var _ io.ReadWriteCloser = (*connHandler)(nil)

type connHandler struct {
	// rwc is the underlying websocket connection
	rwc net.Conn
	// channel that the connection is subscribed to
	channel string
	// rcv is the channel that receives messages from the connection
	rcv chan *Message
}

func (c *connHandler) Consume(msg *Message) {
	c.rcv <- msg
}

func (c *connHandler) Produce() chan *Message {
	return c.rcv
}

// Read implements io.ReadWriteCloser
func (c *connHandler) Read(p []byte) (n int, err error) {
	return c.rwc.Read(p)
}

// Write implements io.ReadWriteCloser
func (c *connHandler) Write(p []byte) (n int, err error) {
	return c.rwc.Write(p)
}

func (c *connHandler) Close() error {
	if err := c.rwc.Close(); err != nil {
		return err
	}
	close(c.rcv)
	return nil
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
