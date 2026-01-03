package system

import (
	"go-crm/internal/common/api"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type WebSocketApi struct {
	Controller *WebSocketController
}

func NewWebSocketApi(controller *WebSocketController) api.Route {
	return &WebSocketApi{
		Controller: controller,
	}
}

func (h *WebSocketApi) Setup(app *fiber.App) {
	app.Get("/api/ws", websocket.New(h.Controller.HandleWebSocket))
}
