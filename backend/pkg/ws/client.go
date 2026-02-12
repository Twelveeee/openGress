package ws

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// WebSocket 写入与心跳参数。
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

type Client struct {
	// WS 客户端连接上下文。
	id        string
	playerID  string
	sessionID string
	conn      *websocket.Conn
	send      chan []byte
	hub       *Hub
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	// 创建新的 WS 客户端实例。
	return &Client{
		id:   uuid.New().String(),
		conn: conn,
		send: make(chan []byte, 64),
		hub:  hub,
	}
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) PlayerID() string {
	return c.playerID
}

func (c *Client) SetPlayerID(playerID string) {
	c.playerID = playerID
}

func (c *Client) SessionID() string {
	return c.sessionID
}

func (c *Client) SetSessionID(sessionID string) {
	c.sessionID = sessionID
}

func (c *Client) Close() {
	_ = c.conn.Close()
}

func (c *Client) Send(msg Message) {
	// 非阻塞发送，避免阻塞主循环。
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("WS marshal failed", "err", err)
		return
	}
	select {
	case c.send <- data:
	default:
		slog.Warn("WS send buffer full", "client_id", c.id)
	}
}

func (c *Client) ReadPump() {
	// 读取客户端消息并分发。
	defer func() {
		c.hub.Unregister(c)
	}()

	c.conn.SetReadLimit(1 << 20)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		msg, err := DecodeMessage(message)
		if err != nil {
			c.Send(NewMessage(MessageError, "", ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid message"}))
			continue
		}

		if handled := HandleSystemMessage(c, msg); handled {
			continue
		}

		c.hub.RouteMessage(c, msg)
	}
}

func (c *Client) WritePump() {
	// 持续写入消息并处理心跳。
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.Unregister(c)
	}()

	for {
		select {
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}
