package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
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

func testWSRuntimeConfig() config.WSRuntimeConfig {
	return config.WSRuntimeConfig{
		MovementTickMS:      120,
		MapTickMS:           1400,
		PlayerStatePushMS:   350,
		NearbyPlayersPushMS: 800,
	}
}

func newTestServer() (*HTTPServer, *AuthStore, *state.GameState) {
	gameState := state.NewGameState()
	auth := NewAuthStore("test-secret", time.Hour)
	server := NewHTTPServer(0, gameState, auth, testGameplayConfig(), testWSRuntimeConfig(), "test-admin", nil)
	return server, auth, gameState
}

func doRequest(t *testing.T, server *HTTPServer, method, path string, body interface{}, token string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	server.engine.ServeHTTP(rec, req)
	return rec
}

func decodeData(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data, _ := resp.Data.(map[string]interface{})
	return data
}

func TestAuthEndpoints(t *testing.T) {
	server, _, _ := newTestServer()

	rec := doRequest(t, server, http.MethodPost, "/api/v1/auth/register", map[string]interface{}{
		"username": "u1",
		"password": "p1",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register expected 200, got %d", rec.Code)
	}
	data := decodeData(t, rec)
	access := data["access_token"].(string)
	refresh := data["refresh_token"].(string)

	rec = doRequest(t, server, http.MethodPost, "/api/v1/auth/login", map[string]interface{}{
		"username": "u1",
		"password": "p1",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodPost, "/api/v1/auth/refresh", map[string]interface{}{
		"refresh_token": refresh,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodPost, "/api/v1/auth/logout", nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout expected 200, got %d", rec.Code)
	}
}

func TestRegisterAssignsRandomPositionWithinMapBounds(t *testing.T) {
	gameState := state.NewGameState()
	auth := NewAuthStore("test-secret", time.Hour)
	cfg := testGameplayConfig()
	cfg.MapBounds = config.MapBoundsConfig{
		MinLat: 10,
		MaxLat: 10.1,
		MinLon: 20,
		MaxLon: 20.1,
	}
	server := NewHTTPServer(0, gameState, auth, cfg, testWSRuntimeConfig(), "test-admin", nil)

	rec := doRequest(t, server, http.MethodPost, "/api/v1/auth/register", map[string]interface{}{
		"username": "spawn-u1",
		"password": "p1",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register expected 200, got %d", rec.Code)
	}
	data := decodeData(t, rec)
	playerID := data["player_id"].(string)
	player, ok := gameState.Players.Get(playerID)
	if !ok {
		t.Fatalf("expected player in state")
	}
	if !gameplay.PositionWithinBounds(player.Position, cfg.MapBounds) {
		t.Fatalf("expected spawn within bounds, got %+v", player.Position)
	}
	if player.Position.Latitude == 0 && player.Position.Longitude == 0 {
		t.Fatalf("expected non-zero spawn position")
	}
}

func TestLoginRepairsOutOfBoundsPosition(t *testing.T) {
	gameState := state.NewGameState()
	auth := NewAuthStore("test-secret", time.Hour)
	cfg := testGameplayConfig()
	cfg.MapBounds = config.MapBoundsConfig{
		MinLat: 10,
		MaxLat: 10,
		MinLon: 20,
		MaxLon: 20,
	}
	server := NewHTTPServer(0, gameState, auth, cfg, testWSRuntimeConfig(), "test-admin", nil)

	rec := doRequest(t, server, http.MethodPost, "/api/v1/auth/register", map[string]interface{}{
		"username": "spawn-u2",
		"password": "p2",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodPost, "/api/v1/auth/login", map[string]interface{}{
		"username": "spawn-u2",
		"password": "p2",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", rec.Code)
	}
	playerID := decodeData(t, rec)["player_id"].(string)

	player, _ := gameState.Players.Get(playerID)
	player.Position = state.Position{Latitude: 99, Longitude: 199}
	gameState.Players.Upsert(player)

	rec = doRequest(t, server, http.MethodPost, "/api/v1/auth/login", map[string]interface{}{
		"username": "spawn-u2",
		"password": "p2",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", rec.Code)
	}
	player, _ = gameState.Players.Get(playerID)
	if !gameplay.PositionWithinBounds(player.Position, cfg.MapBounds) {
		t.Fatalf("expected repaired position within bounds, got %+v", player.Position)
	}
}

func TestLoginRepairsZeroPositionWhenBoundsConfigured(t *testing.T) {
	gameState := state.NewGameState()
	auth := NewAuthStore("test-secret", time.Hour)
	cfg := testGameplayConfig()
	cfg.MapBounds = config.MapBoundsConfig{
		MinLat: -1,
		MaxLat: 1,
		MinLon: -1,
		MaxLon: 1,
	}
	server := NewHTTPServer(0, gameState, auth, cfg, testWSRuntimeConfig(), "test-admin", nil)

	rec := doRequest(t, server, http.MethodPost, "/api/v1/auth/register", map[string]interface{}{
		"username": "spawn-u3",
		"password": "p3",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodPost, "/api/v1/auth/login", map[string]interface{}{
		"username": "spawn-u3",
		"password": "p3",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", rec.Code)
	}
	playerID := decodeData(t, rec)["player_id"].(string)

	player, _ := gameState.Players.Get(playerID)
	player.Position = state.Position{Latitude: 0, Longitude: 0}
	gameState.Players.Upsert(player)

	rec = doRequest(t, server, http.MethodPost, "/api/v1/auth/login", map[string]interface{}{
		"username": "spawn-u3",
		"password": "p3",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", rec.Code)
	}
	player, _ = gameState.Players.Get(playerID)
	if !gameplay.PositionWithinBounds(player.Position, cfg.MapBounds) {
		t.Fatalf("expected repaired position within bounds, got %+v", player.Position)
	}
	if player.Position.Latitude == 0 && player.Position.Longitude == 0 {
		t.Fatalf("expected zero position to be repaired")
	}
}

func TestPlayerEndpoints(t *testing.T) {
	server, _, gameState := newTestServer()

	rec := doRequest(t, server, http.MethodPost, "/api/v1/auth/register", map[string]interface{}{
		"username": "u2",
		"password": "p2",
	}, "")
	data := decodeData(t, rec)
	access := data["access_token"].(string)
	playerID := data["player_id"].(string)

	rec = doRequest(t, server, http.MethodGet, "/api/v1/players/me", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me without token expected 401, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/players/me", nil, "bad-token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("me invalid token expected 401, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/players/me", nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("me expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodPatch, "/api/v1/players/me", map[string]interface{}{
		"autoHack": true,
	}, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch me expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/players/"+playerID, nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("player expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/players/"+playerID+"/stats", nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("stats expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/players/"+playerID+"/visibility", nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("visibility expected 200, got %d", rec.Code)
	}

	if _, ok := gameState.Players.Get(playerID); !ok {
		t.Fatalf("expected player in state")
	}
}

func TestMapPortalInventoryEndpoints(t *testing.T) {
	server, _, gameState := newTestServer()

	rec := doRequest(t, server, http.MethodPost, "/api/v1/auth/register", map[string]interface{}{
		"username": "u3",
		"password": "p3",
	}, "")
	data := decodeData(t, rec)
	access := data["access_token"].(string)
	playerID := data["player_id"].(string)

	gameState.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Links.Upsert(state.Link{
		ID:           "link-1",
		FromPortalID: "portal-1",
		ToPortalID:   "portal-1",
		FromPosition: state.Position{Latitude: 0, Longitude: 0},
		ToPosition:   state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Fields.Upsert(state.Field{
		ID:        "field-1",
		PortalIDs: [3]string{"portal-1", "portal-1", "portal-1"},
		MU:        1,
		Layer:     1,
		Faction:   "RESISTANCE",
	})

	rec = doRequest(t, server, http.MethodGet, "/api/v1/map/entities?minLat=-1&maxLat=1&minLon=-1&maxLon=1", nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("map entities expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/portals/portal-1", nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("portal expected 200, got %d", rec.Code)
	}
	rec = doRequest(t, server, http.MethodGet, "/api/v1/portals/portal-1?format=legacy", nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("portal legacy expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/inventory", nil, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("inventory expected 200, got %d", rec.Code)
	}

	player, _ := gameState.Players.Get(playerID)
	player.Inventory[gameplay.ItemIDCube(1)] = 2
	gameState.Players.Upsert(player)

	rec = doRequest(t, server, http.MethodPost, "/api/v1/inventory/use", map[string]interface{}{
		"itemType": gameplay.ItemIDCube(1),
		"amount":   1,
	}, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("inventory use expected 200, got %d", rec.Code)
	}
	useData := decodeData(t, rec)
	if got := int(useData["apGained"].(float64)); got != 0 {
		t.Fatalf("expected apGained=0 for cube use, got %d", got)
	}
	if got := int(useData["xmDelta"].(float64)); got < 0 {
		t.Fatalf("expected non-negative xmDelta for cube use, got %d", got)
	}

	rec = doRequest(t, server, http.MethodPost, "/api/v1/inventory/recycle", map[string]interface{}{
		"itemType": gameplay.ItemIDCube(1),
		"amount":   1,
	}, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("inventory recycle expected 200, got %d", rec.Code)
	}

	keyPortalID := "1a42c71b53904625ab2142a79a0c4544.16"
	keyItemID := gameplay.ItemIDKey(keyPortalID)
	player, _ = gameState.Players.Get(playerID)
	player.Inventory[keyItemID] = 1
	gameState.Players.Upsert(player)

	rec = doRequest(t, server, http.MethodPost, "/api/v1/inventory/recycle", map[string]interface{}{
		"itemType": keyItemID,
		"amount":   1,
	}, access)
	if rec.Code != http.StatusOK {
		t.Fatalf("inventory recycle key expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	player, _ = gameState.Players.Get(playerID)
	if got := player.Inventory[keyItemID]; got != 0 {
		t.Fatalf("expected key item recycled, remaining=%d", got)
	}
}

func TestLeaderboardLogsConfigEndpoints(t *testing.T) {
	server, _, gameState := newTestServer()

	rec := doRequest(t, server, http.MethodPost, "/api/v1/auth/register", map[string]interface{}{
		"username": "u4",
		"password": "p4",
	}, "")
	data := decodeData(t, rec)
	playerID := data["player_id"].(string)
	player, _ := gameState.Players.Get(playerID)
	player.AP = 100
	gameState.Players.Upsert(player)

	rec = doRequest(t, server, http.MethodGet, "/api/v1/leaderboard", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("leaderboard expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/logs/global", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("logs expected 200, got %d", rec.Code)
	}

	rec = doRequest(t, server, http.MethodGet, "/api/v1/config", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("config expected 200, got %d", rec.Code)
	}
	configData := decodeData(t, rec)
	if got := int(configData["mapTickMs"].(float64)); got != 1400 {
		t.Fatalf("expected mapTickMs 1400, got %d", got)
	}
	wsCadence, ok := configData["wsCadence"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected wsCadence in config response")
	}
	if got := int(wsCadence["movementTickMs"].(float64)); got != 120 {
		t.Fatalf("expected movementTickMs 120, got %d", got)
	}
	if got := int(wsCadence["mapTickMs"].(float64)); got != 1400 {
		t.Fatalf("expected wsCadence.mapTickMs 1400, got %d", got)
	}
	if got := int(wsCadence["playerStatePushMs"].(float64)); got != 350 {
		t.Fatalf("expected playerStatePushMs 350, got %d", got)
	}
	if got := int(wsCadence["nearbyPlayersPushMs"].(float64)); got != 800 {
		t.Fatalf("expected nearbyPlayersPushMs 800, got %d", got)
	}
	attackSpecs, ok := configData["attackSpecs"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected attackSpecs in config response")
	}
	xmp, ok := attackSpecs["XMP"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected attackSpecs.XMP in config response")
	}
	l2, ok := xmp["2"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected attackSpecs.XMP.2 in config response")
	}
	if got := int(l2["costXm"].(float64)); got != 100 {
		t.Fatalf("expected attackSpecs.XMP.2.costXm 100, got %d", got)
	}
	if got := l2["radiusM"].(float64); got <= 0 {
		t.Fatalf("expected positive attackSpecs.XMP.2.radiusM, got %f", got)
	}
}

func TestAdminEndpoints(t *testing.T) {
	server, _, gameState := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/status", nil)
	rec := httptest.NewRecorder()
	server.engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin status without token expected 401, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/status", nil)
	req.Header.Set("X-Admin-Token", "test-admin")
	rec = httptest.NewRecorder()
	server.engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin status expected 200, got %d", rec.Code)
	}

	gameState.Players.Upsert(state.Player{ID: "p1"})
	payload := map[string]interface{}{"playerId": "p1", "reason": "test"}
	rec = doRequest(t, server, http.MethodPost, "/api/v1/admin/ban", payload, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin ban without token expected 401, got %d", rec.Code)
	}

	buf, _ := json.Marshal(payload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/ban", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "test-admin")
	rec = httptest.NewRecorder()
	server.engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin ban expected 200, got %d", rec.Code)
	}
}

func TestCORSAllowsLocalhostOrigin(t *testing.T) {
	server, _, _ := newTestServer()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	server.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("options expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected allow-origin localhost, got %q", got)
	}
}

func TestCORSDeniesNonLocalhostOrigin(t *testing.T) {
	server, _, _ := newTestServer()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()
	server.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("options expected 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow-origin for non-localhost, got %q", got)
	}
}
