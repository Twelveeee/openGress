package ws

import (
	"context"
	"log/slog"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/service"
	playersvc "github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

type Hub struct {
	// Hub 负责连接管理与消息分发。
	register          chan *Client
	unregister        chan *Client
	broadcast         chan Message
	kick              chan string
	clients           map[*Client]struct{}
	state             *state.GameState
	auth              *api.AuthStore
	services          *service.Container
	gameplay          config.GameplayConfig
	wsRuntime         config.WSRuntimeConfig
	nowFn             func() time.Time
	lastPlayerStateAt map[string]time.Time
	lastPlayerState   map[string]PlayerStatePayload
	lastNearbyAt      map[string]time.Time
	lastNearbyPlayers map[string][]NearbyPlayer
}

func NewHub(gameState *state.GameState, auth *api.AuthStore, gameplay config.GameplayConfig, wsRuntime config.WSRuntimeConfig) *Hub {
	// 初始化 hub 与内部通道。
	normalizedWSRuntime := wsRuntime.Normalize()
	return &Hub{
		register:          make(chan *Client),
		unregister:        make(chan *Client),
		broadcast:         make(chan Message, 64),
		kick:              make(chan string, 16),
		clients:           make(map[*Client]struct{}),
		state:             gameState,
		auth:              auth,
		services:          service.NewContainer(gameState, gameplay),
		gameplay:          gameplay,
		wsRuntime:         normalizedWSRuntime,
		nowFn:             time.Now,
		lastPlayerStateAt: make(map[string]time.Time),
		lastPlayerState:   make(map[string]PlayerStatePayload),
		lastNearbyAt:      make(map[string]time.Time),
		lastNearbyPlayers: make(map[string][]NearbyPlayer),
	}
}

func (h *Hub) Run(ctx context.Context) {
	// 主循环：处理连接、广播、移动与地图 tick。
	movementTicker := time.NewTicker(h.movementTickInterval())
	defer movementTicker.Stop()
	mapTicker := time.NewTicker(h.mapTickInterval())
	defer mapTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			for client := range h.clients {
				client.Close()
			}
			return
		case <-movementTicker.C:
			h.handleTick(ctx)
		case <-mapTicker.C:
			h.handleMapTick(ctx)
		case client := <-h.register:
			h.clients[client] = struct{}{}
			slog.InfoContext(ctx, "WS client connected", "client_id", client.ID(), "player_id", client.PlayerID())
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
				slog.InfoContext(ctx, "WS client disconnected", "client_id", client.ID(), "player_id", client.PlayerID())
			}
		case msg := <-h.broadcast:
			for client := range h.clients {
				client.Send(msg)
			}
		case playerID := <-h.kick:
			for client := range h.clients {
				if client.PlayerID() == playerID {
					delete(h.clients, client)
					client.Close()
					slog.InfoContext(ctx, "WS client kicked", "player_id", playerID, "client_id", client.ID())
				}
			}
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(msg Message) {
	h.broadcast <- msg
}

func (h *Hub) RouteMessage(client *Client, msg Message) {
	routeMessage(h, client, msg)
}

// KickPlayer 触发踢人操作（运行中服务）。
func (h *Hub) KickPlayer(playerID string) bool {
	if playerID == "" {
		return false
	}
	select {
	case h.kick <- playerID:
		return true
	default:
		return false
	}
}

func (h *Hub) handleTick(ctx context.Context) {
	// 处理移动 tick、自动 hack 与附近玩家广播。
	if h.state == nil || h.services == nil || h.services.Player == nil || h.services.Hack == nil {
		return
	}
	players := h.state.Players.List()
	if len(players) == 0 {
		return
	}
	portals := h.state.Portals.List()
	now := h.nowFn()

	stepMeters := playersvc.SpeedMPS() * h.movementTickInterval().Seconds()
	updated, movedPlayers := h.services.Player.AdvanceMovement(stepMeters, now)

	for client := range h.clients {
		playerID := client.PlayerID()
		if playerID == "" {
			continue
		}
		player, ok := updated[playerID]
		if !ok {
			continue
		}

		player, autoHack := h.services.Hack.TryAutoHack(player, portals, now)
		if autoHack != nil {
			h.services.Player.Upsert(player)
			updated[player.ID] = player
			client.Send(NewMessage(MessageAutoHackResult, "", AutoHackResultPayload{
				PortalID:        autoHack.PortalID,
				Success:         autoHack.Success,
				CooldownSeconds: autoHack.CooldownSeconds,
				ItemsGained:     autoHack.ItemsGained,
				APGained:        autoHack.APGained,
			}))
		}

		playerState := h.services.Player.BuildPlayerState(player, h.playerStatePushInterval())
		if h.shouldSendPlayerState(player.ID, playerState, now, movedPlayers[player.ID]) {
			client.Send(NewMessage(MessagePlayerState, "", playerState))
			h.lastPlayerState[player.ID] = playerState
			h.lastPlayerStateAt[player.ID] = now
		}

		nearbyRadius := h.services.Player.ViewRadius()
		nearby := h.services.Player.BuildNearbyPlayers(player, updated, nearbyRadius, h.playerStatePushInterval())

		if h.shouldSendNearbyPlayers(player.ID, nearby, now) {
			client.Send(NewMessage(MessageNearbyPlayers, "", NearbyPlayersPayload{
				Players: nearby,
			}))
			h.lastNearbyPlayers[player.ID] = append([]NearbyPlayer(nil), nearby...)
			h.lastNearbyAt[player.ID] = now
		}
	}
}

func (h *Hub) shouldSendPlayerState(playerID string, payload PlayerStatePayload, now time.Time, moved bool) bool {
	lastPayload, ok := h.lastPlayerState[playerID]
	if !ok {
		return true
	}
	if playerStateEqual(lastPayload, payload) {
		return false
	}
	lastAt := h.lastPlayerStateAt[playerID]
	if moved && now.Sub(lastAt) < h.playerStatePushInterval() {
		return false
	}
	return true
}

func (h *Hub) shouldSendNearbyPlayers(playerID string, nearby []NearbyPlayer, now time.Time) bool {
	lastNearby, ok := h.lastNearbyPlayers[playerID]
	if !ok {
		return true
	}
	if nearbyPlayersEqual(lastNearby, nearby) {
		return false
	}
	lastAt := h.lastNearbyAt[playerID]
	if now.Sub(lastAt) < h.nearbyPlayersPushInterval() {
		return false
	}
	return true
}

func (h *Hub) movementTickInterval() time.Duration {
	return time.Duration(h.wsRuntime.MovementTickMS) * time.Millisecond
}

func (h *Hub) mapTickInterval() time.Duration {
	return time.Duration(h.wsRuntime.MapTickMS) * time.Millisecond
}

func (h *Hub) playerStatePushInterval() time.Duration {
	return time.Duration(h.wsRuntime.PlayerStatePushMS) * time.Millisecond
}

func (h *Hub) nearbyPlayersPushInterval() time.Duration {
	return time.Duration(h.wsRuntime.NearbyPlayersPushMS) * time.Millisecond
}

func playerStateEqual(a, b PlayerStatePayload) bool {
	return a.PlayerID == b.PlayerID &&
		a.Latitude == b.Latitude &&
		a.Longitude == b.Longitude &&
		a.RenderLatitude == b.RenderLatitude &&
		a.RenderLongitude == b.RenderLongitude &&
		a.PreRenderLatitude == b.PreRenderLatitude &&
		a.PreRenderLongitude == b.PreRenderLongitude &&
		a.SpeedMps == b.SpeedMps &&
		a.HeadingDeg == b.HeadingDeg
}

func nearbyPlayersEqual(a, b []NearbyPlayer) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			return false
		}
		if a[i].Latitude != b[i].Latitude || a[i].Longitude != b[i].Longitude {
			return false
		}
		if a[i].RenderLatitude != b[i].RenderLatitude || a[i].RenderLongitude != b[i].RenderLongitude {
			return false
		}
		if a[i].PreRenderLatitude != b[i].PreRenderLatitude || a[i].PreRenderLongitude != b[i].PreRenderLongitude {
			return false
		}
	}
	return true
}

func (h *Hub) handleMapTick(ctx context.Context) {
	// 定时下发视野内 Portal 更新。
	if h.state == nil || h.services == nil || h.services.Map == nil {
		return
	}
	players := h.state.Players.List()
	if len(players) == 0 {
		return
	}
	for client := range h.clients {
		playerID := client.PlayerID()
		if playerID == "" {
			continue
		}
		player, ok := h.state.Players.Get(playerID)
		if !ok || player.View == nil {
			continue
		}
		visible := toMapPortals(h.services.Map.CollectPortals(*player.View))
		sendMapUpdates(client, visible, false, MessageMapTick)
	}
}
