package connection

import (
	"encoding/json"
	"fmt"
	"net/http"
	"snake/src/snake/game"
	"snake/src/snake/player"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// Allow all connections by default; in production, restrict this!
	CheckOrigin: func(r *http.Request) bool { return true },
}

// New handler replaces the old websocket.Handler
func ConnectionHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	p := &player.Player{
		FromClient:     make(chan *player.Message, 0),
		ToClient:       make(chan interface{}, 0),
		HeadingChanges: make([]string, 0),
		Disconnected:   false,
		Path:           r.URL.Path,
	}

	quitPinger := make(chan int, 0)
	defer func() { quitPinger <- 0 }()
	go pinger(p, quitPinger)

	go sender(conn, p)

	quit := receiver(conn, p, r)

	myName, theirName := game.Pair(p)
	p.ToClient <- map[string]string{"myName": myName, "theirName": theirName}

	<-quit
}

func receiver(conn *websocket.Conn, p *player.Player, r *http.Request) chan int {
	quit := make(chan int, 0)
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			m := &player.Message{}
			if err := json.Unmarshal(msg, m); err != nil {
				break
			}
			if m.Ping != "" {
				fmt.Printf(
					"%s %s %s\n",
					time.Now().UTC(),
					time.Since(p.PingSent),
					r.RemoteAddr,
				)
			} else {
				p.FromClient <- m
			}
		}
		p.Disconnected = true
		quit <- 1
	}()
	return quit
}

func sender(conn *websocket.Conn, p *player.Player) {
	for m := range p.ToClient {
		// Marshal to JSON and send as TextMessage
		if data, err := json.Marshal(m); err == nil {
			conn.WriteMessage(websocket.TextMessage, data)
		}
	}
	conn.Close()
}

func pinger(p *player.Player, quit chan int) {
	p.PingSent = time.Now()
	p.ToClient <- map[string]string{"ping": "ping"}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			p.PingSent = t
			p.ToClient <- map[string]string{"ping": "ping"}
		case <-quit:
			return
		}
	}
}
