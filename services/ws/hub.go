package ws

import "encoding/json"

type (
	MsgType string
	Hub     struct {
		clients    map[*Client]bool
		broadcast  chan []byte
		register   chan *Client
		unregister chan *Client
	}
	Msg struct {
		Type MsgType     `json:"type"`
		Data interface{} `json:"data"`
	}
)

var ServerHub = NewHub()

func Init() {
	go ServerHub.Run()
}
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func BroadcastMsg(msg Msg) {
	bts, _ := json.Marshal(msg)
	ServerHub.broadcast <- bts
}
