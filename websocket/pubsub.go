package websocket

import (
	"go-chat/pkg/structures"
)

type PSubcriber interface {
	Publisher
	Subscriber
	Set(conn *connHandler) (remove func())
}

type Publisher interface {
	Publish(msg *Message)
}

type Subscriber interface {
	Subscribe() <-chan *Message
}

var _ PSubcriber = (*psub)(nil)

type psub struct {
	// broadcast is the channel that receives messages from the server
	broadcast chan *Message
	// register is the channel that registers new connections
	register chan *connHandler
	// unregister is the channel that unregisters connections
	unregister chan *connHandler
	// connections is the list of connections
	connections *structures.SyncMap[string, structures.Set[*connHandler]]
}

func (p *psub) Set(conn *connHandler) (unset func()) {
	p.register <- conn
	return func() { p.unregister <- conn }
}

// Publish implements PSubcriber
func (p *psub) Publish(msg *Message) {
	p.broadcast <- msg
}

// Subscribe implements PSubcriber
func (p *psub) Subscribe() <-chan *Message {
	return p.broadcast
}

func (p *psub) listen() {
	for {
		select {
		// channelName, connenction
		case conn := <-p.register:
			if conns, ok := p.connections.Load(conn.channel); ok {
				conns.Add(conn)
			} else {
				p.connections.Store(conn.channel, structures.NewSet(conn))
			}
		// channelName, connenction
		case conn := <-p.unregister:
			if conns, ok := p.connections.Load(conn.channel); ok {
				conns.Remove(conn)
			}
		case msg := <-p.Subscribe():
			if conns, ok := p.connections.Load(msg.Channel); ok {
				for conn := range conns {
					select {
					case conn.rcv <- msg:
					default:
						conns.Remove(conn)
					}
				}
			}
		}
	}
}

func newSubscriber() *psub {
	ps := psub{
		broadcast:   make(chan *Message, 256),
		register:    make(chan *connHandler),
		unregister:  make(chan *connHandler),
		connections: structures.NewSyncMap[string, structures.Set[*connHandler]](),
	}
	go ps.listen()
	return &ps
}

/*
function init(clientId, interval, roomId) {
	if (!clientId)
		throw new Error("clientId is required")
	let ws = new WebSocket(`ws://localhost:8080/play?id=${roomId ?? "default"}`);
	ws.onmessage = (e) => console.log(e.data);
	setInterval(() => ws.send(clientId), interval ?? 1000)
}
*/
