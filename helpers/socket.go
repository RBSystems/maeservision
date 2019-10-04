package helpers

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

//Client is a websocket client that also has a send channel
type Client struct {
	// the websocket connection
	conn *websocket.Conn

	// bufferend channel of outbound messages
	send chan interface{}
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 5 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
	client  = Client{}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// allowing all origins!!
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

//ServeWebsocket opens a websocket and serves it
func ServeWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	client := &Client{conn: conn, send: make(chan interface{}, 256)}

	go client.write()
}

func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteJSON(msg)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}
