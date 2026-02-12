package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/database"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/Twelveeee/openGress/backend/pkg/ws"
)

// App 负责组装 HTTP/WS 服务与全局状态。
type App struct {
	cfg           *config.Config
	httpServer    *api.HTTPServer
	wsServer      *ws.Server
	state         *state.GameState
	persistence   *PersistenceManager
	persistCancel context.CancelFunc
}

// New 初始化应用（含内存状态与鉴权）。
func New(cfg *config.Config) *App {
	gameState := state.NewGameState()
	if cfg.Persist.Enabled {
		snapshot, err := database.LoadSnapshotFromDB()
		if err != nil {
			slog.Error("load snapshot from db failed, fallback to empty state", "err", err)
		} else if snapshotHasData(snapshot) {
			gameState = state.NewGameStateFromSnapshot(snapshot)
			slog.Info("loaded game state snapshot from db",
				"players", len(snapshot.Players),
				"portals", len(snapshot.Portals),
				"links", len(snapshot.Links),
				"fields", len(snapshot.Fields),
			)
		} else {
			slog.Info("database snapshot empty, start with empty game state")
		}
	}
	normalizeLogStoreCapacity(gameState, cfg.Gameplay.LogCapacity)

	secret := cfg.Auth.JWTSecret
	if secret == "" {
		secret = "dev-secret"
	}
	var authStore *api.AuthStore
	if database.GormDB != nil {
		authRepo := database.NewAuthUserStore(database.GormDB)
		authStore = api.NewAuthStoreWithRepository(secret, time.Duration(cfg.Auth.AccessTTLSeconds)*time.Second, authRepo)
	} else {
		authStore = api.NewAuthStore(secret, time.Duration(cfg.Auth.AccessTTLSeconds)*time.Second)
	}
	wsServer := ws.NewServerWithAuth(cfg.Server.WSPort, gameState, authStore, cfg.Gameplay, cfg.Server.WS)
	adminRuntime := ws.NewAdminRuntime(wsServer.Hub())
	app := &App{
		cfg:        cfg,
		httpServer: api.NewHTTPServer(cfg.Server.HTTPPort, gameState, authStore, cfg.Gameplay, cfg.Server.WS, cfg.Admin.Token, adminRuntime),
		wsServer:   wsServer,
		state:      gameState,
	}
	if cfg.Persist.Enabled {
		deltaWriter := database.NewDeltaWriter(database.GormDB, 1000)
		app.persistence = NewPersistenceManager(deltaWriter.SaveSnapshot)
	}
	return app
}

// Start 启动 HTTP 与 WS 服务。
func (a *App) Start(ctx context.Context) {
	if a.persistence != nil {
		a.persistence.Start()
		intervalSeconds := a.cfg.Persist.FlushIntervalSeconds
		if intervalSeconds <= 0 {
			intervalSeconds = 3
		}
		persistCtx, cancel := context.WithCancel(ctx)
		a.persistCancel = cancel

		go func() {
			ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					a.persistence.EnqueueLatest(a.state.Snapshot())
				case <-persistCtx.Done():
					return
				}
			}
		}()
	}

	go func() {
		slog.InfoContext(ctx, "HTTP server starting", "addr", a.httpServer.Addr())
		if err := a.httpServer.Start(); err != nil {
			slog.ErrorContext(ctx, "HTTP server stopped", "err", err)
		}
	}()

	go func() {
		slog.InfoContext(ctx, "WebSocket server starting", "addr", a.wsServer.Addr())
		if err := a.wsServer.Start(ctx); err != nil {
			slog.ErrorContext(ctx, "WebSocket server stopped", "err", err)
		}
	}()
}

// Shutdown 优雅停止服务。
func (a *App) Shutdown(ctx context.Context) error {
	if a.persistCancel != nil {
		a.persistCancel()
	}
	if a.persistence != nil {
		timeoutSeconds := a.cfg.Persist.ShutdownFlushTimeoutSeconds
		if timeoutSeconds <= 0 {
			timeoutSeconds = 2
		}
		if err := a.persistence.FlushBestEffort(time.Duration(timeoutSeconds)*time.Second, a.state.Snapshot()); err != nil {
			slog.WarnContext(ctx, "best-effort persistence flush failed", "err", err)
		}
		a.persistence.Close()
	}
	if err := a.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return a.wsServer.Shutdown(ctx)
}

func snapshotHasData(snapshot state.Snapshot) bool {
	if len(snapshot.Players) > 0 || len(snapshot.Portals) > 0 || len(snapshot.Links) > 0 || len(snapshot.Fields) > 0 || len(snapshot.Logs) > 0 {
		return true
	}
	if snapshot.Admin != nil && len(snapshot.Admin.Banned) > 0 {
		return true
	}
	return false
}

func normalizeLogStoreCapacity(gameState *state.GameState, capacity int) {
	if gameState == nil {
		return
	}
	if capacity <= 0 {
		return
	}
	logs := []state.LogEntry{}
	if gameState.Logs != nil {
		logs = gameState.Logs.List()
	}
	store := state.NewLogStore(capacity)
	for _, entry := range logs {
		store.Add(entry)
	}
	gameState.Logs = store
}
