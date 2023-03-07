package websocket

import (
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var _ http.Handler = (*Client)(nil)

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

	for {
		p, err := wsutil.ReadClientText(rwc)
		if err != nil {
			return
		}

		err = wsutil.WriteServerText(rwc, p)
		if err != nil {
			return
		}
	}
}

func NewClient() *Client {
	c := Client{}

	return &c
}
