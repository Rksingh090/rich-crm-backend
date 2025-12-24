package api

import (
	"go-crm/internal/controllers"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type WebSocketApi struct {
	App        *fiber.App
	Controller *controllers.WebSocketController
}

func NewWebSocketApi() *WebSocketApi {
	return &WebSocketApi{}
}

// Setup registers WebSocket route
func (h *WebSocketApi) Setup(app *fiber.App) {
	app.Get("/ws", websocket.New(h.Controller.HandleWebSocket))
}
