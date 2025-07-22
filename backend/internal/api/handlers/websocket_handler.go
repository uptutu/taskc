package handlers

import (
	"encoding/json"
	"log"

	"github.com/gofiber/websocket/v2"
)

type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// HandleTaskWebSocket 处理任务WebSocket连接
func HandleTaskWebSocket(c *websocket.Conn) {
	defer c.Close()

	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			break
		}

		if messageType == websocket.TextMessage {
			var wsMsg WebSocketMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				log.Println("JSON unmarshal error:", err)
				continue
			}

			switch wsMsg.Type {
			case "subscribe_tasks":
				// 订阅任务状态更新
				handleTaskSubscription(c)
			case "ping":
				// 心跳检测
				response := WebSocketMessage{
					Type: "pong",
					Data: nil,
				}
				if data, err := json.Marshal(response); err == nil {
					c.WriteMessage(websocket.TextMessage, data)
				}
			}
		}
	}
}

func handleTaskSubscription(c *websocket.Conn) {
	// 这里应该实现实际的任务状态推送逻辑
	// 可以通过Redis Pub/Sub或者定期查询数据库来实现
	response := WebSocketMessage{
		Type: "task_status_update",
		Data: map[string]interface{}{
			"message": "Subscribed to task updates",
		},
	}

	if data, err := json.Marshal(response); err == nil {
		c.WriteMessage(websocket.TextMessage, data)
	}
}