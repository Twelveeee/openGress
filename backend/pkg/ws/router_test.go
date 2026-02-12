package ws

import (
	"context"
	"testing"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	playersvc "github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func testGameplayConfig() config.GameplayConfig {
	return config.GameplayConfig{
		MapBounds: config.MapBoundsConfig{
			MinLat: -90,
			MaxLat: 90,
			MinLon: -180,
			MaxLon: 180,
		},
		ViewRadiusM:    400,
		InteractRangeM: 40,
		LogCapacity:    50,
		XMCost: config.XMCostConfig{
			DeployResonatorBase:     50,
			DeployResonatorPerLevel: 50,
			AttackXMPBase:           50,
			AttackXMPPerLevel:       50,
			AttackUSBase:            50,
			AttackUSPerLevel:        50,
			DeployModByRarity: map[string]int{
				"COMMON":    400,
				"RARE":      800,
				"VERY_RARE": 1000,
			},
			DeployModByType: map[string]int{
				"FORCE_AMP": 800,
				"TURRET":    800,
				"LINK_AMP":  800,
			},
			DeployModFlat: 400,
			LinkFlat:      250,
			ChargePerXM:   1,
		},
		APReward: config.APRewardConfig{
			CapturePortal:    500,
			DeployResonator:  125,
			UpgradeResonator: 65,
			DeployMod:        150,
			DestroyResonator: 75,
			CreateLink:       313,
			CreateField:      1250,
			HackPortal:       50,
		},
	}
}

func newTestHub() (*Hub, *api.AuthStore) {
	return newTestHubWithRuntime(config.DefaultWSRuntimeConfig())
}

func newTestHubWithRuntime(runtime config.WSRuntimeConfig) (*Hub, *api.AuthStore) {
	auth := api.NewAuthStore("test-secret", time.Hour)
	return NewHub(state.NewGameState(), auth, testGameplayConfig(), runtime), auth
}

func collectClientMessages(t *testing.T, client *Client, max int) []Message {
	t.Helper()
	messages := make([]Message, 0, max)
	for i := 0; i < max; i++ {
		select {
		case raw := <-client.send:
			msg, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode message: %v", err)
			}
			messages = append(messages, msg)
		default:
			return messages
		}
	}
	return messages
}

func containsMessageType(messages []Message, messageType string) bool {
	for _, msg := range messages {
		if msg.Type == messageType {
			return true
		}
	}
	return false
}

func TestConnectMissingAuthToken(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 1)}
	msg := Message{
		Type: MessageConnect,
		ID:   "1",
		Data: map[string]interface{}{
			"playerId": "p1",
		},
	}

	routeMessage(hub, client, msg)

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessageError {
			t.Fatalf("expected ERROR, got %s", out.Type)
		}
		var payload ErrorPayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode error payload: %v", err)
		}
		if payload.Code != api.ErrCodeUnauthorized {
			t.Fatalf("expected code %d, got %d", api.ErrCodeUnauthorized, payload.Code)
		}
	default:
		t.Fatal("expected response message")
	}
}

func TestConnectInvalidAuthToken(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 1)}
	msg := Message{
		Type: MessageConnect,
		ID:   "1a",
		Data: map[string]interface{}{
			"authToken": "bad-token",
		},
	}

	routeMessage(hub, client, msg)

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessageError {
			t.Fatalf("expected ERROR, got %s", out.Type)
		}
	default:
		t.Fatal("expected response message")
	}
}

func TestConnectSuccessCreatesSession(t *testing.T) {
	hub, auth := newTestHub()
	user, _ := auth.Register("p1", "pw")
	token, _, err := auth.IssueTokens("p1")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	client := &Client{send: make(chan []byte, 1)}
	msg := Message{
		Type: MessageConnect,
		ID:   "1",
		Data: map[string]interface{}{
			"playerId":  "p1",
			"authToken": token,
		},
	}

	routeMessage(hub, client, msg)

	if client.PlayerID() != user.PlayerID {
		t.Fatalf("expected player id to be set")
	}
	if client.SessionID() == "" {
		t.Fatalf("expected session id to be set")
	}
	if _, ok := hub.state.Players.Get(user.PlayerID); !ok {
		t.Fatalf("expected player to be upserted")
	}

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessageConnected {
			t.Fatalf("expected CONNECTED, got %s", out.Type)
		}
	default:
		t.Fatal("expected response message")
	}
}

func TestConnectSendsImmediatePlayerState(t *testing.T) {
	hub, auth := newTestHub()
	user, _ := auth.Register("connect-state", "pw")
	token, _, err := auth.IssueTokens("connect-state")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	client := &Client{send: make(chan []byte, 4)}
	msg := Message{
		Type: MessageConnect,
		ID:   "connect-state-1",
		Data: map[string]interface{}{
			"authToken": token,
		},
	}

	routeMessage(hub, client, msg)
	messages := collectClientMessages(t, client, 4)
	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(messages))
	}
	if messages[0].Type != MessageConnected {
		t.Fatalf("expected first message CONNECTED, got %s", messages[0].Type)
	}
	if messages[1].Type != MessagePlayerState {
		t.Fatalf("expected second message PLAYER_STATE, got %s", messages[1].Type)
	}
	var payload PlayerStatePayload
	if err := decodePayload(messages[1].Data, &payload); err != nil {
		t.Fatalf("decode player state payload: %v", err)
	}
	if payload.PlayerID != user.PlayerID {
		t.Fatalf("expected playerId %s, got %s", user.PlayerID, payload.PlayerID)
	}
	if payload.RenderLatitude != payload.Latitude || payload.RenderLongitude != payload.Longitude {
		t.Fatalf("expected render position equal to authoritative position")
	}
}

func TestHandleConnectRepairsOutOfBoundsPlayerPosition(t *testing.T) {
	cfg := testGameplayConfig()
	cfg.MapBounds = config.MapBoundsConfig{
		MinLat: 10,
		MaxLat: 10,
		MinLon: 20,
		MaxLon: 20,
	}
	auth := api.NewAuthStore("test-secret", time.Hour)
	hub := NewHub(state.NewGameState(), auth, cfg, config.DefaultWSRuntimeConfig())
	user, _ := auth.Register("repair-out", "pw")
	token, _, err := auth.IssueTokens("repair-out")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	hub.state.Players.Upsert(state.Player{
		ID:       user.PlayerID,
		Username: "repair-out",
		Position: state.Position{Latitude: 99, Longitude: 199},
	})

	client := &Client{send: make(chan []byte, 1)}
	msg := Message{
		Type: MessageConnect,
		ID:   "repair-1",
		Data: map[string]interface{}{
			"authToken": token,
		},
	}
	routeMessage(hub, client, msg)

	player, ok := hub.state.Players.Get(user.PlayerID)
	if !ok {
		t.Fatalf("expected player in state")
	}
	if !gameplay.PositionWithinBounds(player.Position, cfg.MapBounds) {
		t.Fatalf("expected repaired position in bounds, got %+v", player.Position)
	}
}

func TestHandleConnectRepairsZeroPositionWhenBoundsConfigured(t *testing.T) {
	cfg := testGameplayConfig()
	cfg.MapBounds = config.MapBoundsConfig{
		MinLat: -1,
		MaxLat: 1,
		MinLon: -1,
		MaxLon: 1,
	}
	auth := api.NewAuthStore("test-secret", time.Hour)
	hub := NewHub(state.NewGameState(), auth, cfg, config.DefaultWSRuntimeConfig())
	user, _ := auth.Register("repair-zero", "pw")
	token, _, err := auth.IssueTokens("repair-zero")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	hub.state.Players.Upsert(state.Player{
		ID:       user.PlayerID,
		Username: "repair-zero",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	client := &Client{send: make(chan []byte, 1)}
	msg := Message{
		Type: MessageConnect,
		ID:   "repair-2",
		Data: map[string]interface{}{
			"authToken": token,
		},
	}
	routeMessage(hub, client, msg)

	player, ok := hub.state.Players.Get(user.PlayerID)
	if !ok {
		t.Fatalf("expected player in state")
	}
	if !gameplay.PositionWithinBounds(player.Position, cfg.MapBounds) {
		t.Fatalf("expected repaired position in bounds, got %+v", player.Position)
	}
	if player.Position.Latitude == 0 && player.Position.Longitude == 0 {
		t.Fatalf("expected zero position to be repaired")
	}
}

func TestPlayerTargetUpdateStoresTarget(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 1), playerID: "p1"}
	msg := Message{
		Type: MessagePlayerTargetUpdate,
		ID:   "2",
		Data: map[string]interface{}{
			"latitude":  10.5,
			"longitude": 20.5,
		},
	}

	routeMessage(hub, client, msg)

	player, ok := hub.state.Players.Get("p1")
	if !ok {
		t.Fatalf("expected player to be created")
	}
	if player.Target == nil {
		t.Fatalf("expected target to be set")
	}
	if player.Target.Latitude != 10.5 || player.Target.Longitude != 20.5 {
		t.Fatalf("unexpected target coordinates")
	}
}

func TestPlayerViewUpdateClampsBounds(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 1), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	msg := Message{
		Type: MessagePlayerViewUpdate,
		ID:   "3",
		Data: map[string]interface{}{
			"bounds": map[string]interface{}{
				"minLat": -1.0,
				"maxLat": 1.0,
				"minLon": -1.0,
				"maxLon": 1.0,
			},
		},
	}

	routeMessage(hub, client, msg)

	player, ok := hub.state.Players.Get("p1")
	if !ok || player.View == nil {
		t.Fatalf("expected view bounds to be set")
	}
	halfLatMeters := (player.View.MaxLat - player.View.MinLat) * playersvc.MetersPerDegLat / 2
	halfLonMeters := (player.View.MaxLon - player.View.MinLon) * playersvc.MetersPerDegLon(0) / 2
	if halfLatMeters > 400.0+1 || halfLonMeters > 400.0+1 {
		t.Fatalf("expected bounds to be clamped within 400m")
	}
}

func TestPlayerViewUpdateSendsMapUpdate(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 4), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	msg := Message{
		Type: MessagePlayerViewUpdate,
		ID:   "4",
		Data: map[string]interface{}{
			"bounds": map[string]interface{}{
				"minLat": -0.001,
				"maxLat": 0.001,
				"minLon": -0.001,
				"maxLon": 0.001,
			},
		},
	}

	routeMessage(hub, client, msg)

	var gotMapUpdate bool
	for i := 0; i < 2; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type == MessageMapUpdate {
				gotMapUpdate = true
			}
		default:
		}
	}
	if !gotMapUpdate {
		t.Fatalf("expected MAP_UPDATE to be sent")
	}
}

func TestPlayerViewUpdateSendsLinkAndField(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 8), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{ID: "a", Position: state.Position{Latitude: 0, Longitude: 0}})
	hub.state.Portals.Upsert(state.Portal{ID: "b", Position: state.Position{Latitude: 0, Longitude: 0.001}})
	hub.state.Portals.Upsert(state.Portal{ID: "c", Position: state.Position{Latitude: 0.001, Longitude: 0}})
	hub.state.Links.Upsert(state.Link{
		ID:           "link-1",
		FromPortalID: "a",
		ToPortalID:   "b",
		FromPosition: state.Position{Latitude: 0, Longitude: 0},
		ToPosition:   state.Position{Latitude: 0, Longitude: 0.001},
	})
	hub.state.Fields.Upsert(state.Field{
		ID:        "field-1",
		PortalIDs: [3]string{"a", "b", "c"},
		MU:        1,
		Layer:     1,
		Faction:   "RESISTANCE",
	})

	msg := Message{
		Type: MessagePlayerViewUpdate,
		ID:   "4b",
		Data: map[string]interface{}{
			"bounds": map[string]interface{}{
				"minLat": -0.01,
				"maxLat": 0.01,
				"minLon": -0.01,
				"maxLon": 0.01,
			},
		},
	}
	routeMessage(hub, client, msg)

	var gotLink bool
	var gotField bool
	for i := 0; i < 6; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type == MessageLinkUpdate {
				gotLink = true
			}
			if out.Type == MessageFieldCreated {
				gotField = true
			}
		default:
		}
	}
	if !gotLink || !gotField {
		t.Fatalf("expected LINK_UPDATE and FIELD_CREATED")
	}
}

func TestToggleAutoHackUpdatesState(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 1), playerID: "p1"}
	msg := Message{
		Type: MessagePlayerToggleAutoHack,
		ID:   "5",
		Data: map[string]interface{}{
			"enabled": true,
		},
	}

	routeMessage(hub, client, msg)

	player, ok := hub.state.Players.Get("p1")
	if !ok {
		t.Fatalf("expected player to be created")
	}
	if !player.AutoHack {
		t.Fatalf("expected auto hack to be enabled")
	}
}

func TestMovePlayerTowardsReachesTarget(t *testing.T) {
	player := state.Player{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Target:   &state.Position{Latitude: 0, Longitude: 0.0001},
	}
	updated, moved := playersvc.MovePlayerTowards(player, 10000)
	if !moved {
		t.Fatalf("expected moved to be true")
	}
	if updated.Target != nil {
		t.Fatalf("expected target to be cleared")
	}
}

func TestHandleTickSendsStateAndNearby(t *testing.T) {
	hub, _ := newTestHub()
	clientA := &Client{send: make(chan []byte, 4), playerID: "a"}
	clientB := &Client{send: make(chan []byte, 4), playerID: "b"}
	hub.clients[clientA] = struct{}{}
	hub.clients[clientB] = struct{}{}

	hub.state.Players.Upsert(state.Player{
		ID:       "a",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Players.Upsert(state.Player{
		ID:       "b",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
	})

	hub.handleTick(context.Background())

	var hasState bool
	var hasNearby bool
	for i := 0; i < 2; i++ {
		select {
		case raw := <-clientA.send:
			msg, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode message: %v", err)
			}
			if msg.Type == MessagePlayerState {
				hasState = true
			}
			if msg.Type == MessageNearbyPlayers {
				hasNearby = true
			}
		default:
		}
	}

	if !hasState || !hasNearby {
		t.Fatalf("expected PLAYER_STATE and NEARBY_PLAYERS")
	}
}

func TestHandleTickPlayerStateContainsRenderAndPreRenderPosition(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 8), playerID: "a"}
	hub.clients[client] = struct{}{}

	hub.state.Players.Upsert(state.Player{
		ID:       "a",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Target:   &state.Position{Latitude: 0, Longitude: 0.02},
	})

	hub.handleTick(context.Background())
	messages := collectClientMessages(t, client, 8)

	var payload PlayerStatePayload
	var found bool
	for _, msg := range messages {
		if msg.Type != MessagePlayerState {
			continue
		}
		if err := decodePayload(msg.Data, &payload); err != nil {
			t.Fatalf("decode player state payload: %v", err)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("expected PLAYER_STATE message")
	}
	if payload.RenderLatitude != payload.Latitude || payload.RenderLongitude != payload.Longitude {
		t.Fatalf("expected render position equal to authoritative position, got render=(%f,%f) pos=(%f,%f)",
			payload.RenderLatitude, payload.RenderLongitude, payload.Latitude, payload.Longitude)
	}
	if payload.PreRenderLongitude <= payload.RenderLongitude {
		t.Fatalf("expected pre-render longitude ahead of render longitude, got pre=%f render=%f",
			payload.PreRenderLongitude, payload.RenderLongitude)
	}
}

func TestHandleTickNearbyPlayersContainsRenderAndPreRenderPosition(t *testing.T) {
	hub, _ := newTestHub()
	clientA := &Client{send: make(chan []byte, 8), playerID: "a"}
	clientB := &Client{send: make(chan []byte, 8), playerID: "b"}
	hub.clients[clientA] = struct{}{}
	hub.clients[clientB] = struct{}{}

	hub.state.Players.Upsert(state.Player{
		ID:       "a",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Players.Upsert(state.Player{
		ID:       "b",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Target:   &state.Position{Latitude: 0, Longitude: 0.02},
	})

	hub.handleTick(context.Background())
	messages := collectClientMessages(t, clientA, 8)

	var payload NearbyPlayersPayload
	var found bool
	for _, msg := range messages {
		if msg.Type != MessageNearbyPlayers {
			continue
		}
		if err := decodePayload(msg.Data, &payload); err != nil {
			t.Fatalf("decode nearby payload: %v", err)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("expected NEARBY_PLAYERS message")
	}
	if len(payload.Players) != 1 {
		t.Fatalf("expected 1 nearby player, got %d", len(payload.Players))
	}
	nearby := payload.Players[0]
	if nearby.RenderLatitude != nearby.Latitude || nearby.RenderLongitude != nearby.Longitude {
		t.Fatalf("expected nearby render position equal to authoritative position")
	}
	if nearby.PreRenderLongitude <= nearby.RenderLongitude {
		t.Fatalf("expected nearby pre-render longitude ahead of render longitude, got pre=%f render=%f",
			nearby.PreRenderLongitude, nearby.RenderLongitude)
	}
}

func TestHandleTickThrottlesStateAndNearbyWhileMoving(t *testing.T) {
	runtime := config.DefaultWSRuntimeConfig()
	runtime.PlayerStatePushMS = 1000
	runtime.NearbyPlayersPushMS = 1000
	hub, _ := newTestHubWithRuntime(runtime)
	now := time.Unix(1700000000, 0)
	hub.nowFn = func() time.Time { return now }

	clientA := &Client{send: make(chan []byte, 16), playerID: "a"}
	clientB := &Client{send: make(chan []byte, 16), playerID: "b"}
	hub.clients[clientA] = struct{}{}
	hub.clients[clientB] = struct{}{}

	hub.state.Players.Upsert(state.Player{
		ID:       "a",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Target:   &state.Position{Latitude: 0, Longitude: 0.01},
	})
	hub.state.Players.Upsert(state.Player{
		ID:       "b",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Target:   &state.Position{Latitude: 0, Longitude: 0.02},
	})

	hub.handleTick(context.Background())
	first := collectClientMessages(t, clientA, 8)
	if !containsMessageType(first, MessagePlayerState) || !containsMessageType(first, MessageNearbyPlayers) {
		t.Fatalf("expected first tick to send PLAYER_STATE and NEARBY_PLAYERS")
	}

	now = now.Add(200 * time.Millisecond)
	hub.handleTick(context.Background())
	second := collectClientMessages(t, clientA, 8)
	if containsMessageType(second, MessagePlayerState) || containsMessageType(second, MessageNearbyPlayers) {
		t.Fatalf("expected second immediate tick to be throttled, got %+v", second)
	}

	now = now.Add(1500 * time.Millisecond)
	hub.handleTick(context.Background())
	third := collectClientMessages(t, clientA, 8)
	if !containsMessageType(third, MessagePlayerState) || !containsMessageType(third, MessageNearbyPlayers) {
		t.Fatalf("expected third tick after cadence to send PLAYER_STATE and NEARBY_PLAYERS")
	}
}

func TestHandleTickCanUseLowerCadenceFromConfig(t *testing.T) {
	runtime := config.DefaultWSRuntimeConfig()
	runtime.PlayerStatePushMS = 20
	runtime.NearbyPlayersPushMS = 20
	hub, _ := newTestHubWithRuntime(runtime)
	now := time.Unix(1700000000, 0)
	hub.nowFn = func() time.Time { return now }

	clientA := &Client{send: make(chan []byte, 16), playerID: "a"}
	clientB := &Client{send: make(chan []byte, 16), playerID: "b"}
	hub.clients[clientA] = struct{}{}
	hub.clients[clientB] = struct{}{}

	hub.state.Players.Upsert(state.Player{
		ID:       "a",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Target:   &state.Position{Latitude: 0, Longitude: 0.01},
	})
	hub.state.Players.Upsert(state.Player{
		ID:       "b",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Target:   &state.Position{Latitude: 0, Longitude: 0.02},
	})

	hub.handleTick(context.Background())
	first := collectClientMessages(t, clientA, 8)
	if !containsMessageType(first, MessagePlayerState) || !containsMessageType(first, MessageNearbyPlayers) {
		t.Fatalf("expected first tick to send PLAYER_STATE and NEARBY_PLAYERS")
	}

	now = now.Add(30 * time.Millisecond)
	hub.handleTick(context.Background())
	second := collectClientMessages(t, clientA, 8)
	if !containsMessageType(second, MessagePlayerState) || !containsMessageType(second, MessageNearbyPlayers) {
		t.Fatalf("expected second tick with lower cadence to send PLAYER_STATE and NEARBY_PLAYERS")
	}
}

func TestGetOrCreatePlayerUsesConfiguredSpawn(t *testing.T) {
	cfg := testGameplayConfig()
	cfg.MapBounds = config.MapBoundsConfig{
		MinLat: 10,
		MaxLat: 14,
		MinLon: 20,
		MaxLon: 24,
	}

	gameState := state.NewGameState()
	player := playersvc.New(gameState, cfg).GetOrCreate("new-player")
	if !gameplay.PositionWithinBounds(player.Position, cfg.MapBounds) {
		t.Fatalf("expected spawn position in bounds, got %+v", player.Position)
	}
}

func TestHandleMapTickSendsMapTick(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 4), playerID: "p1"}
	hub.clients[client] = struct{}{}
	hub.state.Players.Upsert(state.Player{
		ID: "p1",
		View: &state.Bounds{
			MinLat: -0.001,
			MaxLat: 0.001,
			MinLon: -0.001,
			MaxLon: 0.001,
		},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	hub.handleMapTick(context.Background())

	var gotMapTick bool
	for i := 0; i < 2; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type == MessageMapTick {
				gotMapTick = true
			}
		default:
		}
	}
	if !gotMapTick {
		t.Fatalf("expected MAP_TICK to be sent")
	}
}

func TestAutoHackSendsResult(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 8), playerID: "p1"}
	hub.clients[client] = struct{}{}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
		AutoHack: true,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
	})

	hub.handleTick(context.Background())

	var gotAutoHack bool
	for i := 0; i < 4; i++ {
		select {
		case raw := <-client.send:
			msg, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode message: %v", err)
			}
			if msg.Type == MessageAutoHackResult {
				gotAutoHack = true
			}
		default:
		}
	}
	if !gotAutoHack {
		t.Fatalf("expected AUTO_HACK_RESULT")
	}
}

func TestAutoHackCooldownSkips(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 16), playerID: "p1"}
	hub.clients[client] = struct{}{}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
		AutoHack: true,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
	})

	hub.handleTick(context.Background())
	hub.handleTick(context.Background())

	autoHackCount := 0
	for i := 0; i < 8; i++ {
		select {
		case raw := <-client.send:
			msg, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode message: %v", err)
			}
			if msg.Type == MessageAutoHackResult {
				autoHackCount++
			}
		default:
		}
	}
	if autoHackCount != 1 {
		t.Fatalf("expected 1 AUTO_HACK_RESULT, got %d", autoHackCount)
	}
}

func TestDeployResonatorSuccess(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDResonator(3): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	msg := Message{
		Type: MessagePlayerDeployRes,
		ID:   "6",
		Data: map[string]interface{}{
			"portalId":        "portal-1",
			"slot":            1,
			"level":           3,
			"expectedVersion": 0,
		},
	}

	routeMessage(hub, client, msg)

	portal, ok := hub.state.Portals.Get("portal-1")
	if !ok {
		t.Fatalf("expected portal to exist")
	}
	slot := portal.Resonators[1]
	if slot.Level != 3 || slot.Version != 1 {
		t.Fatalf("unexpected resonator slot state")
	}
	player, ok := hub.state.Players.Get("p1")
	if !ok {
		t.Fatalf("expected player to exist")
	}
	if player.AP != 625 {
		t.Fatalf("expected AP +625 for capture deploy, got %d", player.AP)
	}
	if player.XM != 850 {
		t.Fatalf("expected XM deducted to 850, got %d", player.XM)
	}
	if got := player.Inventory[gameplay.ItemIDResonator(3)]; got != 0 {
		t.Fatalf("expected RESO_L3 consumed, got %d", got)
	}

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessagePlayerResourceUpdate {
			t.Fatalf("expected %s, got %s", MessagePlayerResourceUpdate, out.Type)
		}
		if out.ID != "6" {
			t.Fatalf("expected resource update id=6, got %s", out.ID)
		}
		var payload struct {
			PlayerID       string         `json:"playerId"`
			APGained       int            `json:"apGained"`
			XMDelta        int            `json:"xmDelta"`
			InventoryDelta map[string]int `json:"inventoryDelta"`
		}
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode resource update payload: %v", err)
		}
		if payload.PlayerID != "p1" {
			t.Fatalf("unexpected player id in resource update: %+v", payload)
		}
		if payload.APGained != 625 {
			t.Fatalf("expected apGained=625, got %d", payload.APGained)
		}
		if payload.XMDelta != -150 {
			t.Fatalf("expected xmDelta=-150, got %d", payload.XMDelta)
		}
		if got := payload.InventoryDelta[gameplay.ItemIDResonator(3)]; got != -1 {
			t.Fatalf("expected inventoryDelta RESO_L3=-1, got %d", got)
		}
	default:
		t.Fatalf("expected player resource update response")
	}

	select {
	case out := <-hub.broadcast:
		if out.Type != MessagePortalUpdate {
			t.Fatalf("expected %s, got %s", MessagePortalUpdate, out.Type)
		}
		if out.ID != "" {
			t.Fatalf("expected portal update broadcast id to be empty, got %q", out.ID)
		}
		var payload struct {
			PortalID   string `json:"portalId"`
			PlayerID   string `json:"playerId"`
			Slot       int    `json:"slot"`
			Level      int    `json:"level"`
			SlotEnergy int    `json:"slotEnergy"`
		}
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode portal update payload: %v", err)
		}
		if payload.PortalID != "portal-1" || payload.PlayerID != "p1" {
			t.Fatalf("unexpected portal update identity: %+v", payload)
		}
		if payload.Slot != 1 || payload.Level != 3 {
			t.Fatalf("unexpected portal update resonator fields: %+v", payload)
		}
		if payload.SlotEnergy <= 0 {
			t.Fatalf("expected positive slotEnergy, got %d", payload.SlotEnergy)
		}
	default:
		t.Fatalf("expected portal update broadcast")
	}
}

func TestDeployResonatorConflict(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     2,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDResonator(2): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID: "portal-1",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Version: 1},
		},
	})
	msg := Message{
		Type: MessagePlayerDeployRes,
		ID:   "7",
		Data: map[string]interface{}{
			"portalId":        "portal-1",
			"slot":            1,
			"level":           2,
			"expectedVersion": 0,
		},
	}

	routeMessage(hub, client, msg)

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessageError {
			t.Fatalf("expected ERROR, got %s", out.Type)
		}
		var payload ErrorPayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode error payload: %v", err)
		}
		if payload.Code != api.ErrCodeConflict {
			t.Fatalf("expected conflict code, got %d", payload.Code)
		}
	default:
		t.Fatalf("expected conflict response")
	}
}

func TestDeployModSuccess(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDMod("SHIELD", "COMMON"): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	msg := Message{
		Type: MessagePlayerDeployMod,
		ID:   "8",
		Data: map[string]interface{}{
			"portalId":        "portal-1",
			"slot":            1,
			"modType":         "SHIELD",
			"rarity":          "COMMON",
			"expectedVersion": 0,
		},
	}

	routeMessage(hub, client, msg)

	portal, ok := hub.state.Portals.Get("portal-1")
	if !ok {
		t.Fatalf("expected portal to exist")
	}
	slot := portal.Mods[1]
	if slot.ModType != "SHIELD" || slot.Version != 1 {
		t.Fatalf("unexpected mod slot state")
	}
	player, ok := hub.state.Players.Get("p1")
	if !ok {
		t.Fatalf("expected player to exist")
	}
	if player.AP != 150 {
		t.Fatalf("expected AP +150 for deploy mod, got %d", player.AP)
	}
	if player.XM != 600 {
		t.Fatalf("expected XM deducted to 600, got %d", player.XM)
	}

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessagePlayerResourceUpdate {
			t.Fatalf("expected %s, got %s", MessagePlayerResourceUpdate, out.Type)
		}
		var payload struct {
			PlayerID       string         `json:"playerId"`
			APGained       int            `json:"apGained"`
			XMDelta        int            `json:"xmDelta"`
			InventoryDelta map[string]int `json:"inventoryDelta"`
		}
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode resource update payload: %v", err)
		}
		if payload.PlayerID != "p1" {
			t.Fatalf("unexpected player id in resource update: %+v", payload)
		}
		if payload.APGained != 150 {
			t.Fatalf("expected apGained=150, got %d", payload.APGained)
		}
		if payload.XMDelta != -400 {
			t.Fatalf("expected xmDelta=-400, got %d", payload.XMDelta)
		}
		if got := payload.InventoryDelta[gameplay.ItemIDMod("SHIELD", "COMMON")]; got != -1 {
			t.Fatalf("expected inventoryDelta MOD:SHIELD:COMMON=-1, got %d", got)
		}
	default:
		t.Fatalf("expected player resource update response")
	}

	select {
	case out := <-hub.broadcast:
		if out.Type != MessagePortalUpdate {
			t.Fatalf("expected %s, got %s", MessagePortalUpdate, out.Type)
		}
		var payload struct {
			PortalID string `json:"portalId"`
			PlayerID string `json:"playerId"`
			ModSlot  int    `json:"modSlot"`
			ModType  string `json:"modType"`
		}
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode portal update payload: %v", err)
		}
		if payload.PortalID != "portal-1" || payload.PlayerID != "p1" {
			t.Fatalf("unexpected portal update identity: %+v", payload)
		}
		if payload.ModSlot != 1 || payload.ModType != "SHIELD" {
			t.Fatalf("unexpected portal update mod fields: %+v", payload)
		}
	default:
		t.Fatalf("expected portal update broadcast")
	}
}

func TestDeployModPortalShieldAlias(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDMod("SHIELD", "COMMON"): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	msg := Message{
		Type: MessagePlayerDeployMod,
		ID:   "8-alias",
		Data: map[string]interface{}{
			"portalId":        "portal-1",
			"slot":            1,
			"modType":         "PORTAL_SHIELD",
			"rarity":          "COMMON",
			"expectedVersion": 0,
		},
	}

	routeMessage(hub, client, msg)

	portal, ok := hub.state.Portals.Get("portal-1")
	if !ok {
		t.Fatalf("expected portal to exist")
	}
	if got := portal.Mods[1].ModType; got != "SHIELD" {
		t.Fatalf("expected alias to normalize to SHIELD, got %s", got)
	}
}

func TestDeployModConflict(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDMod("SHIELD", "COMMON"): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID: "portal-1",
		Mods: map[int]state.ModSlot{
			1: {Slot: 1, ModType: "SHIELD", Version: 1},
		},
	})
	msg := Message{
		Type: MessagePlayerDeployMod,
		ID:   "9",
		Data: map[string]interface{}{
			"portalId":        "portal-1",
			"slot":            1,
			"modType":         "SHIELD",
			"rarity":          "COMMON",
			"expectedVersion": 0,
		},
	}

	routeMessage(hub, client, msg)

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessageError {
			t.Fatalf("expected ERROR, got %s", out.Type)
		}
		var payload ErrorPayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode error payload: %v", err)
		}
		if payload.Code != api.ErrCodeConflict {
			t.Fatalf("expected conflict code, got %d", payload.Code)
		}
	default:
		t.Fatalf("expected conflict response")
	}
}

func TestChargePortalSuccess(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 4), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     1,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:      "portal-1",
		Faction: "RESISTANCE",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 900},
		},
		Energy:   900,
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	msg := Message{
		Type: MessagePlayerChargePortal,
		ID:   "10",
		Data: map[string]interface{}{
			"portalId": "portal-1",
			"amount":   5,
		},
	}

	routeMessage(hub, client, msg)

	portal, ok := hub.state.Portals.Get("portal-1")
	if !ok {
		t.Fatalf("expected portal to exist")
	}
	if portal.Energy != 905 {
		t.Fatalf("expected energy to be updated to 905, got %d", portal.Energy)
	}
	player, ok := hub.state.Players.Get("p1")
	if !ok {
		t.Fatalf("expected player to exist")
	}
	if player.XM != 995 {
		t.Fatalf("expected player xm to be 995, got %d", player.XM)
	}

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessagePlayerResourceUpdate {
			t.Fatalf("expected %s, got %s", MessagePlayerResourceUpdate, out.Type)
		}
		var payload PlayerResourceUpdatePayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode player resource payload: %v", err)
		}
		if payload.PlayerID != "p1" {
			t.Fatalf("expected playerId p1, got %s", payload.PlayerID)
		}
		if payload.XMDelta != -5 {
			t.Fatalf("expected xmDelta -5, got %d", payload.XMDelta)
		}
	default:
		t.Fatalf("expected player resource update")
	}

	select {
	case out := <-hub.broadcast:
		if out.Type != MessagePortalUpdate {
			t.Fatalf("expected %s broadcast, got %s", MessagePortalUpdate, out.Type)
		}
		var payload PortalUpdatePayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode portal update payload: %v", err)
		}
		if payload.PortalID != "portal-1" {
			t.Fatalf("expected portalId portal-1, got %s", payload.PortalID)
		}
		if payload.Energy != 905 {
			t.Fatalf("expected portal energy 905 in broadcast, got %d", payload.Energy)
		}
	default:
		t.Fatalf("expected portal update broadcast")
	}
}

func TestChargePortalFullNoOpStillSendsResourceUpdate(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 4), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     1,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:      "portal-1",
		Faction: "RESISTANCE",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Energy:   1000,
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	msg := Message{
		Type: MessagePlayerChargePortal,
		ID:   "10-noop",
		Data: map[string]interface{}{
			"portalId": "portal-1",
			"amount":   500,
		},
	}

	routeMessage(hub, client, msg)

	player, ok := hub.state.Players.Get("p1")
	if !ok {
		t.Fatalf("expected player to exist")
	}
	if player.XM != 1000 {
		t.Fatalf("expected player xm unchanged, got %d", player.XM)
	}
	portal, ok := hub.state.Portals.Get("portal-1")
	if !ok {
		t.Fatalf("expected portal to exist")
	}
	if portal.Energy != 1000 {
		t.Fatalf("expected portal energy unchanged, got %d", portal.Energy)
	}

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessagePlayerResourceUpdate {
			t.Fatalf("expected %s, got %s", MessagePlayerResourceUpdate, out.Type)
		}
		var payload PlayerResourceUpdatePayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode player resource payload: %v", err)
		}
		if payload.XMDelta != 0 {
			t.Fatalf("expected xmDelta 0, got %d", payload.XMDelta)
		}
	default:
		t.Fatalf("expected player resource update")
	}

	select {
	case out := <-hub.broadcast:
		if out.Type != MessagePortalUpdate {
			t.Fatalf("expected %s broadcast, got %s", MessagePortalUpdate, out.Type)
		}
		var payload PortalUpdatePayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode portal update payload: %v", err)
		}
		if payload.Energy != 1000 {
			t.Fatalf("expected unchanged energy 1000, got %d", payload.Energy)
		}
	default:
		t.Fatalf("expected portal update broadcast")
	}
}

func TestCreateLinkSendsUpdateWhenVisible(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 4), playerID: "p1"}
	hub.clients[client] = struct{}{}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDKey("p1"): 1, gameplay.ItemIDKey("p2"): 1},
		Position:  state.Position{Latitude: 0, Longitude: -0.2},
		View: &state.Bounds{
			MinLat: -0.1,
			MaxLat: 0.1,
			MinLon: -0.1,
			MaxLon: 0.1,
		},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: -0.2},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p2",
		Position: state.Position{Latitude: 0, Longitude: 0.2},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	msg := Message{
		Type: MessagePlayerCreateLink,
		ID:   "11",
		Data: map[string]interface{}{
			"fromPortalId": "p1",
			"toPortalId":   "p2",
		},
	}

	routeMessage(hub, client, msg)

	links := hub.state.Links.List()
	if len(links) != 1 {
		t.Fatalf("expected link to be created")
	}

	var gotLink bool
	for i := 0; i < 3; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type == MessageLinkUpdate {
				gotLink = true
			}
		default:
		}
	}
	if !gotLink {
		t.Fatalf("expected LINK_UPDATE to be sent")
	}
}

func TestCreateLinkDistanceExceeded(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     1,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDKey("p1"): 1, gameplay.ItemIDKey("p2"): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Faction:  "RESISTANCE",
		Level:    1,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p2",
		Position: state.Position{Latitude: 0, Longitude: 1},
		Faction:  "RESISTANCE",
		Level:    1,
	})
	msg := Message{
		Type: MessagePlayerCreateLink,
		ID:   "11b",
		Data: map[string]interface{}{
			"fromPortalId": "p1",
			"toPortalId":   "p2",
		},
	}
	routeMessage(hub, client, msg)
	select {
	case raw := <-client.send:
		out, _ := DecodeMessage(raw)
		if out.Type != MessageError {
			t.Fatalf("expected ERROR for distance exceeded")
		}
	default:
		t.Fatalf("expected error response")
	}
}

func TestCreateLinkCrossing(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDKey("a"): 1, gameplay.ItemIDKey("b"): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "a",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "b",
		Position: state.Position{Latitude: 0.2, Longitude: 0.2},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "c",
		Position: state.Position{Latitude: 0.2, Longitude: 0},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "d",
		Position: state.Position{Latitude: 0, Longitude: 0.2},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	hub.state.Links.Upsert(state.Link{
		ID:           "link-1",
		FromPortalID: "c",
		ToPortalID:   "d",
		FromPosition: state.Position{Latitude: 0.2, Longitude: 0},
		ToPosition:   state.Position{Latitude: 0, Longitude: 0.2},
	})
	msg := Message{
		Type: MessagePlayerCreateLink,
		ID:   "11c",
		Data: map[string]interface{}{
			"fromPortalId": "a",
			"toPortalId":   "b",
		},
	}
	routeMessage(hub, client, msg)
	select {
	case raw := <-client.send:
		out, _ := DecodeMessage(raw)
		if out.Type != MessageError {
			t.Fatalf("expected ERROR for crossing")
		}
	default:
		t.Fatalf("expected error response")
	}
}

func TestCreateLinkOutLimit(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDKey("p1"): 1, gameplay.ItemIDKey("p9"): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	for i := 0; i < 8; i++ {
		id := "t" + string(rune('a'+i))
		pos := state.Position{Latitude: 0.01 * float64(i+1), Longitude: 0}
		hub.state.Portals.Upsert(state.Portal{
			ID:       id,
			Position: pos,
			Faction:  "RESISTANCE",
			Level:    8,
		})
		hub.state.Links.Upsert(state.Link{
			ID:           id,
			FromPortalID: "p1",
			ToPortalID:   id,
			FromPosition: state.Position{Latitude: 0, Longitude: 0},
			ToPosition:   pos,
		})
	}
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p9",
		Position: state.Position{Latitude: 0.3, Longitude: 0},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	msg := Message{
		Type: MessagePlayerCreateLink,
		ID:   "11d",
		Data: map[string]interface{}{
			"fromPortalId": "p1",
			"toPortalId":   "p9",
		},
	}
	routeMessage(hub, client, msg)
	select {
	case raw := <-client.send:
		out, _ := DecodeMessage(raw)
		if out.Type != MessageError {
			t.Fatalf("expected ERROR for out link limit")
		}
	default:
		t.Fatalf("expected error response")
	}
}

func TestFieldCreatedOnTriangle(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 4), playerID: "p1"}
	hub.clients[client] = struct{}{}
	hub.state.Players.Upsert(state.Player{
		ID:      "p1",
		Faction: "RESISTANCE",
		Level:   8,
		XM:      1000,
		MaxXM:   3000,
		Inventory: map[string]int{
			gameplay.ItemIDKey("a"): 2,
			gameplay.ItemIDKey("b"): 2,
			gameplay.ItemIDKey("c"): 2,
		},
		Position: state.Position{Latitude: 0, Longitude: 0},
		View: &state.Bounds{
			MinLat: -0.5,
			MaxLat: 0.5,
			MinLon: -0.5,
			MaxLon: 0.5,
		},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "a",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "b",
		Position: state.Position{Latitude: 0.1, Longitude: 0},
		Faction:  "RESISTANCE",
		Level:    8,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "c",
		Position: state.Position{Latitude: 0, Longitude: 0.1},
		Faction:  "RESISTANCE",
		Level:    8,
	})

	msg1 := Message{Type: MessagePlayerCreateLink, ID: "f1", Data: map[string]interface{}{"fromPortalId": "a", "toPortalId": "b"}}
	msg2 := Message{Type: MessagePlayerCreateLink, ID: "f2", Data: map[string]interface{}{"fromPortalId": "b", "toPortalId": "c"}}
	msg3 := Message{Type: MessagePlayerCreateLink, ID: "f3", Data: map[string]interface{}{"fromPortalId": "a", "toPortalId": "c"}}
	player, _ := hub.state.Players.Get("p1")
	player.Position = state.Position{Latitude: 0, Longitude: 0}
	hub.state.Players.Upsert(player)
	routeMessage(hub, client, msg1)
	player.Position = state.Position{Latitude: 0.1, Longitude: 0}
	hub.state.Players.Upsert(player)
	routeMessage(hub, client, msg2)
	player.Position = state.Position{Latitude: 0, Longitude: 0}
	hub.state.Players.Upsert(player)
	routeMessage(hub, client, msg3)

	var gotField bool
	for i := 0; i < 6; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type == MessageFieldCreated {
				gotField = true
			}
		default:
		}
	}
	if !gotField {
		t.Fatalf("expected FIELD_CREATED")
	}
}

func TestAttackPortalSuccess(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 8), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDXMP(2): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Faction:  "ENLIGHTENED",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 50},
		},
		Level:  1,
		Energy: 50,
	})
	msg := Message{
		Type: MessagePlayerAttack,
		ID:   "12",
		Data: map[string]interface{}{
			"portalId":    "portal-1",
			"weaponType":  "XMP",
			"weaponLevel": 2,
		},
	}

	routeMessage(hub, client, msg)

	portal, ok := hub.state.Portals.Get("portal-1")
	if !ok {
		t.Fatalf("expected portal to exist")
	}
	if portal.Energy >= 1000 {
		t.Fatalf("expected portal energy to decrease")
	}

	var gotAttackResult bool
	var gotResourceUpdate bool
	for i := 0; i < 6; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type == MessagePlayerResourceUpdate {
				var payload PlayerResourceUpdatePayload
				if err := decodePayload(out.Data, &payload); err != nil {
					t.Fatalf("decode resource payload: %v", err)
				}
				if payload.InventoryDelta[gameplay.ItemIDXMP(2)] != -1 {
					t.Fatalf("expected inventory delta -1 for XMP, got %+v", payload.InventoryDelta)
				}
				if payload.XMDelta >= 0 {
					t.Fatalf("expected negative xm delta, got %d", payload.XMDelta)
				}
				if payload.APGained != 75 {
					t.Fatalf("expected apGained=75 for destroyed resonator, got %d", payload.APGained)
				}
				gotResourceUpdate = true
			}
			if out.Type == MessageAttackResult {
				var payload AttackResultPayload
				if err := decodePayload(out.Data, &payload); err != nil {
					t.Fatalf("decode attack result payload: %v", err)
				}
				if payload.WeaponType != "XMP" || payload.WeaponLevel != 2 {
					t.Fatalf("unexpected attack payload: %+v", payload)
				}
				if payload.MitigationApplied < 0 || payload.MitigationApplied > 1 {
					t.Fatalf("invalid mitigationApplied: %f", payload.MitigationApplied)
				}
				gotAttackResult = true
			}
		default:
		}
	}
	if !gotResourceUpdate {
		t.Fatalf("expected PLAYER_RESOURCE_UPDATE")
	}
	if !gotAttackResult {
		t.Fatalf("expected ATTACK_RESULT")
	}
}

func TestAttackPortalAllowsEmptyFire(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 6), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDXMP(2): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})

	msg := Message{
		Type: MessagePlayerAttack,
		ID:   "empty-fire",
		Data: map[string]interface{}{
			"portalId":    "",
			"weaponType":  "XMP",
			"weaponLevel": 2,
		},
	}
	routeMessage(hub, client, msg)

	var gotAttackResult bool
	var gotResourceUpdate bool
	for i := 0; i < 6; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type == MessageAttackResult {
				var payload AttackResultPayload
				if err := decodePayload(out.Data, &payload); err != nil {
					t.Fatalf("decode attack result payload: %v", err)
				}
				if payload.PortalID != "" {
					t.Fatalf("expected empty portalId, got %q", payload.PortalID)
				}
				if payload.DamageDealt != 0 {
					t.Fatalf("expected empty-fire damage 0, got %d", payload.DamageDealt)
				}
				gotAttackResult = true
			}
			if out.Type == MessagePlayerResourceUpdate {
				var payload PlayerResourceUpdatePayload
				if err := decodePayload(out.Data, &payload); err != nil {
					t.Fatalf("decode resource payload: %v", err)
				}
				if payload.InventoryDelta[gameplay.ItemIDXMP(2)] != -1 {
					t.Fatalf("expected inventory delta -1 for XMP, got %+v", payload.InventoryDelta)
				}
				if payload.XMDelta >= 0 {
					t.Fatalf("expected negative xm delta, got %d", payload.XMDelta)
				}
				gotResourceUpdate = true
			}
		default:
		}
	}
	if !gotAttackResult {
		t.Fatalf("expected ATTACK_RESULT")
	}
	if !gotResourceUpdate {
		t.Fatalf("expected PLAYER_RESOURCE_UPDATE")
	}
}

func TestAttackBroadcastPortalUpdateIncludesSnapshot(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 12), playerID: "p1"}
	hub.clients[client] = struct{}{}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDXMP(2): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
		View:      &state.Bounds{MinLat: -1, MaxLat: 1, MinLon: -1, MaxLon: 1},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0.00005},
		Faction:  "ENLIGHTENED",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Mods: map[int]state.ModSlot{
			1: {Slot: 1, ModType: "SHIELD", Rarity: "RARE", PlayerID: "defender"},
		},
		Level:  1,
		Energy: 1000,
	})

	msg := Message{
		Type: MessagePlayerAttack,
		ID:   "snapshot",
		Data: map[string]interface{}{
			"portalId":    "portal-1",
			"weaponType":  "XMP",
			"weaponLevel": 2,
		},
	}
	routeMessage(hub, client, msg)

	for i := 0; i < 12; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type != MessagePortalUpdate {
				continue
			}
			var payload PortalUpdatePayload
			if err := decodePayload(out.Data, &payload); err != nil {
				t.Fatalf("decode portal update payload: %v", err)
			}
			if payload.PortalID != "portal-1" {
				t.Fatalf("unexpected portalId: %s", payload.PortalID)
			}
			if len(payload.Resonators) == 0 {
				t.Fatalf("expected resonators snapshot in portal update")
			}
			if len(payload.Mods) == 0 {
				t.Fatalf("expected mods snapshot in portal update")
			}
			return
		default:
		}
	}
	t.Fatalf("expected PORTAL_UPDATE with snapshot")
}

func TestAttackChargeBonusClamp(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 6), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        5000,
		MaxXM:     6000,
		Inventory: map[string]int{gameplay.ItemIDXMP(8): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Faction:  "ENLIGHTENED",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Level:  1,
		Energy: 1000,
	})

	msg := Message{
		Type: MessagePlayerAttack,
		ID:   "charge-bonus",
		Data: map[string]interface{}{
			"portalId":    "portal-1",
			"weaponType":  "XMP",
			"weaponLevel": 8,
			"chargeBonus": 1.8,
		},
	}
	routeMessage(hub, client, msg)

	for i := 0; i < 6; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type != MessageAttackResult {
				continue
			}
			var payload AttackResultPayload
			if err := decodePayload(out.Data, &payload); err != nil {
				t.Fatalf("decode attack payload: %v", err)
			}
			if payload.ChargeBonus != 0.2 {
				t.Fatalf("expected clamped charge bonus 0.2, got %f", payload.ChargeBonus)
			}
			return
		default:
		}
	}
	t.Fatalf("expected ATTACK_RESULT payload")
}

func TestAttackPortalBatchMapUpdate(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 8), playerID: "p1"}
	hub.clients[client] = struct{}{}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        5000,
		MaxXM:     6000,
		Inventory: map[string]int{gameplay.ItemIDXMP(8): 1},
		View:      &state.Bounds{MinLat: -1, MaxLat: 1, MinLon: -1, MaxLon: 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Faction:  "ENLIGHTENED",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Level:  1,
		Energy: 1000,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p2",
		Position: state.Position{Latitude: 0, Longitude: 0.0002},
		Faction:  "ENLIGHTENED",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Level:  1,
		Energy: 1000,
	})
	msg := Message{
		Type: MessagePlayerAttack,
		ID:   "12b",
		Data: map[string]interface{}{
			"portalId":    "p1",
			"weaponType":  "XMP",
			"weaponLevel": 8,
		},
	}

	routeMessage(hub, client, msg)

	var gotMapUpdate bool
	for i := 0; i < 6; i++ {
		select {
		case raw := <-client.send:
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type == MessageMapUpdate {
				gotMapUpdate = true
			}
		default:
		}
	}
	if !gotMapUpdate {
		t.Fatalf("expected MAP_UPDATE for multi-portal attack")
	}
}

func TestCreateLinkMissingKeys(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Faction:  "RESISTANCE",
		Level:    1,
		XM:       1000,
		MaxXM:    3000,
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
		Faction:  "RESISTANCE",
		Level:    1,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p2",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Faction:  "RESISTANCE",
		Level:    1,
	})
	msg := Message{
		Type: MessagePlayerCreateLink,
		ID:   "missing-keys",
		Data: map[string]interface{}{
			"fromPortalId": "p1",
			"toPortalId":   "p2",
		},
	}
	routeMessage(hub, client, msg)

	select {
	case raw := <-client.send:
		out, _ := DecodeMessage(raw)
		if out.Type != MessageError {
			t.Fatalf("expected ERROR for missing keys")
		}
	default:
		t.Fatalf("expected error response")
	}
}

func TestAttackPortalInsufficientXM(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        0,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDXMP(2): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Faction:  "ENLIGHTENED",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Level:  1,
		Energy: 1000,
	})
	msg := Message{
		Type: MessagePlayerAttack,
		ID:   "no-xm",
		Data: map[string]interface{}{
			"portalId":    "portal-1",
			"weaponType":  "XMP",
			"weaponLevel": 2,
		},
	}
	routeMessage(hub, client, msg)
	select {
	case raw := <-client.send:
		out, _ := DecodeMessage(raw)
		if out.Type != MessageError {
			t.Fatalf("expected ERROR for insufficient xm")
		}
	default:
		t.Fatalf("expected error response")
	}
}

func TestHackPortalSuccess(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Faction:  "RESISTANCE",
		Level:    3,
		AP:       0,
		XM:       3000,
		MaxXM:    3000,
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Faction:  "ENLIGHTENED",
		Level:    3,
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
	})

	msg := Message{
		Type: MessageHackPortal,
		ID:   "hack-1",
		Data: map[string]interface{}{
			"portalId": "portal-1",
		},
	}

	routeMessage(hub, client, msg)

	player, ok := hub.state.Players.Get("p1")
	if !ok {
		t.Fatalf("player not found")
	}
	if player.AP != 50 {
		t.Fatalf("expected AP +50, got %d", player.AP)
	}
	if len(player.Inventory) == 0 {
		t.Fatalf("expected items granted from hack")
	}
	if _, ok := player.HackCooldowns["portal-1"]; !ok {
		t.Fatalf("expected cooldown to be set")
	}

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessageHackResult {
			t.Fatalf("expected HACK_RESULT, got %s", out.Type)
		}
		var payload HackResultPayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if !payload.Success {
			t.Fatalf("expected successful hack")
		}
	default:
		t.Fatalf("expected hack result")
	}
}

func TestHackPortalCooldown(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 4), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Faction:  "RESISTANCE",
		Level:    3,
		AP:       0,
		XM:       3000,
		MaxXM:    3000,
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Faction:  "ENLIGHTENED",
		Level:    3,
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
	})

	msg := Message{
		Type: MessageHackPortal,
		ID:   "hack-2",
		Data: map[string]interface{}{
			"portalId": "portal-1",
		},
	}
	routeMessage(hub, client, msg)
	routeMessage(hub, client, msg)

	var second HackResultPayload
	got := 0
	for got < 2 {
		select {
		case raw := <-client.send:
			got++
			out, err := DecodeMessage(raw)
			if err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if out.Type != MessageHackResult {
				t.Fatalf("expected HACK_RESULT, got %s", out.Type)
			}
			if got == 2 {
				if err := decodePayload(out.Data, &second); err != nil {
					t.Fatalf("decode payload: %v", err)
				}
			}
		default:
			t.Fatalf("expected two hack results")
		}
	}
	if second.Success {
		t.Fatalf("expected cooldown failure on second hack")
	}
	if second.CooldownSeconds <= 0 {
		t.Fatalf("expected positive cooldown seconds")
	}
}

func TestHackPortalInventoryOverCapacityReturnsError(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 2), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Faction:  "RESISTANCE",
		Level:    1,
		AP:       0,
		XM:       3000,
		MaxXM:    3000,
		Position: state.Position{Latitude: 0, Longitude: 0},
		Inventory: map[string]int{
			gameplay.ItemIDXMP(1): 501,
		},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Faction:  "ENLIGHTENED",
		Level:    1,
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
	})

	msg := Message{
		Type: MessageHackPortal,
		ID:   "hack-over-cap",
		Data: map[string]interface{}{
			"portalId": "portal-1",
		},
	}
	routeMessage(hub, client, msg)

	select {
	case raw := <-client.send:
		out, err := DecodeMessage(raw)
		if err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Type != MessageError {
			t.Fatalf("expected ERROR, got %s", out.Type)
		}
		var payload ErrorPayload
		if err := decodePayload(out.Data, &payload); err != nil {
			t.Fatalf("decode error payload: %v", err)
		}
		if payload.Code != api.ErrCodeBadRequest {
			t.Fatalf("expected bad request code, got %d", payload.Code)
		}
		if payload.Message != "inventory capacity exceeded" {
			t.Fatalf("unexpected error message: %s", payload.Message)
		}
	default:
		t.Fatalf("expected ERROR response")
	}
}

func TestAutoHackSkipsWhenInventoryOverCapacity(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 16), playerID: "p1"}
	hub.clients[client] = struct{}{}
	hub.state.Players.Upsert(state.Player{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0},
		AutoHack: true,
		Level:    1,
		Inventory: map[string]int{
			gameplay.ItemIDXMP(1): 501,
		},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
	})

	hub.handleTick(context.Background())

	messages := collectClientMessages(t, client, 8)
	if containsMessageType(messages, MessageAutoHackResult) {
		t.Fatalf("expected no AUTO_HACK_RESULT when inventory over capacity")
	}

	player, ok := hub.state.Players.Get("p1")
	if !ok {
		t.Fatalf("player not found")
	}
	if _, exists := player.HackCooldowns["portal-1"]; exists {
		t.Fatalf("expected no hack cooldown when auto hack is skipped")
	}
}

func TestAttackNeutralPortalCleansLinksAndFields(t *testing.T) {
	hub, _ := newTestHub()
	client := &Client{send: make(chan []byte, 8), playerID: "p1"}
	hub.state.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        6000,
		MaxXM:     10000,
		Inventory: map[string]int{gameplay.ItemIDXMP(8): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p1",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Faction:  "ENLIGHTENED",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 10},
		},
		Level:  1,
		Energy: 10,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p2",
		Position: state.Position{Latitude: 0.0002, Longitude: 0.0001},
		Faction:  "ENLIGHTENED",
		Level:    1,
	})
	hub.state.Portals.Upsert(state.Portal{
		ID:       "p3",
		Position: state.Position{Latitude: 0.0001, Longitude: 0.0002},
		Faction:  "ENLIGHTENED",
		Level:    1,
	})
	hub.state.Links.Upsert(state.Link{
		ID:           "link-1",
		FromPortalID: "p1",
		ToPortalID:   "p2",
		FromPosition: state.Position{Latitude: 0, Longitude: 0.0001},
		ToPosition:   state.Position{Latitude: 0.0002, Longitude: 0.0001},
	})
	hub.state.Fields.Upsert(state.Field{
		ID:        "field-1",
		PortalIDs: [3]string{"p1", "p2", "p3"},
		Faction:   "ENLIGHTENED",
		MU:        10,
		Layer:     1,
	})

	msg := Message{
		Type: MessagePlayerAttack,
		ID:   "neutral-cleanup",
		Data: map[string]interface{}{
			"portalId":    "p1",
			"weaponType":  "XMP",
			"weaponLevel": 8,
		},
	}
	routeMessage(hub, client, msg)

	portal, ok := hub.state.Portals.Get("p1")
	if !ok {
		t.Fatalf("target portal missing")
	}
	if portal.Faction != "NEUTRAL" {
		t.Fatalf("expected neutral portal, got %s", portal.Faction)
	}
	if len(hub.state.Links.List()) != 0 {
		t.Fatalf("expected links to be cleaned after neutralize")
	}
	if len(hub.state.Fields.List()) != 0 {
		t.Fatalf("expected fields to be cleaned after neutralize")
	}
}
