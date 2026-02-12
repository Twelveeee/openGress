package ws

import (
	"encoding/json"
	"log/slog"
	"time"
)

func DecodeMessage(data []byte) (Message, error) {
	// 解析客户端消息并补全时间戳。
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, err
	}
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}
	return msg, nil
}

func HandleSystemMessage(client *Client, msg Message) bool {
	// 处理 PING 等系统消息。
	switch msg.Type {
	case MessagePing:
		client.Send(NewMessage(MessagePong, msg.ID, nil))
		return true
	default:
		return false
	}
}

func routeMessage(hub *Hub, client *Client, msg Message) {
	// 业务消息路由入口，仅负责分发。
	switch msg.Type {
	case MessageConnect:
		handleConnect(hub, client, msg)
	case MessagePlayerTargetUpdate:
		handlePlayerTargetUpdate(hub, client, msg)
	case MessagePlayerViewUpdate:
		handlePlayerViewUpdate(hub, client, msg)
	case MessagePlayerToggleAutoHack:
		handlePlayerToggleAutoHack(hub, client, msg)
	case MessageHackPortal:
		handleHackPortal(hub, client, msg)
	case MessagePlayerDeployRes:
		handlePlayerDeployRes(hub, client, msg)
	case MessagePlayerDeployMod:
		handlePlayerDeployMod(hub, client, msg)
	case MessagePlayerChargePortal:
		handlePlayerChargePortal(hub, client, msg)
	case MessagePlayerCreateLink:
		handlePlayerCreateLink(hub, client, msg)
	case MessagePlayerAttack:
		handlePlayerAttack(hub, client, msg)
	case MessagePlayerLocalUpdate:
		handlePlayerLocalUpdateDeprecated(client, msg)
	default:
		slog.Info("WS message received", "client_id", client.ID(), "type", msg.Type)
	}
}
