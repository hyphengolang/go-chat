package websocket

import (
	"context"
	"net"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func readWrite(conn *connHandler) {
	for {
		p, err := wsutil.ReadClientText(conn.rwc)
		if err != nil {
			return
		}

		// channel:text
		p = append([]byte(conn.channel+":"), p...)

		err = wsutil.WriteServerText(conn.rwc, p)
		if err != nil {
			return
		}
	}
}

// Client is a websocket client
// currently only supports text messages
type Client struct {
	// u upgrades the HTTP request to a websocket connection
	u ws.HTTPUpgrader
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
	}

	go readWrite(&conn)
}

var _ http.Handler = (*Client)(nil)

func NewClient() *Client {
	c := Client{}

	return &c
}

type connHandler struct {
	// rwc is the underlying websocket connection
	rwc net.Conn
	// channel that the connection is subscribed to
	channel string
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
