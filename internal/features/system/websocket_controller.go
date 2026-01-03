package system

import (
	"log"

	"github.com/gofiber/contrib/websocket"
)

type WebSocketController struct {
}

func NewWebSocketController() *WebSocketController {
	return &WebSocketController{}
}

func (h *WebSocketController) HandleWebSocket(c *websocket.Conn) {
	var (
		mt  int
		msg []byte
		err error
	)
	for {
		if mt, msg, err = c.ReadMessage(); err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", msg)

		if err = c.WriteMessage(mt, msg); err != nil {
			log.Println("write:", err)
			break
		}
	}
}
