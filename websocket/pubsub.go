package websocket

import "go-chat/pkg/structures"

type PubSub struct {
	// broadcast is the channel that receives messages from the server
	broadcast chan *Message
	// register is the channel that registers new connections
	register chan *connHandler
	// unregister is the channel that unregisters connections
	unregister chan *connHandler
	// connections is the list of connections
	connections *structures.SyncMap[string, structures.Set[*connHandler]]
}

func (ps *PubSub) listen() {
	for {
		select {
		case conn := <-ps.register:
			if conns, ok := ps.connections.Load(conn.channel); ok {
				conns.Add(conn)
			} else {
				ps.connections.Store(conn.channel, structures.NewSet(conn))
			}
		case conn := <-ps.unregister:
			if conns, ok := ps.connections.Load(conn.channel); ok {
				conns.Remove(conn)

				close(conn.rcv)
			}
		case msg := <-ps.broadcast:
			if conns, ok := ps.connections.Load(msg.channel); ok {
				for conn := range conns {
					select {
					case conn.rcv <- msg:
					default:
						close(conn.rcv)
						delete(conns, conn)
					}
				}
			}
		}
	}
}

func NewPubSub() *PubSub {
	ps := PubSub{
		broadcast:   make(chan *Message, 256),
		register:    make(chan *connHandler),
		unregister:  make(chan *connHandler),
		connections: structures.NewSyncMap[string, structures.Set[*connHandler]](),
	}
	go ps.listen()
	return &ps
}
