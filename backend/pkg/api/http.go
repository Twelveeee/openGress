package api

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/gin-gonic/gin"
)

// HTTPServer 承载所有 HTTP API 路由与中间件。
type HTTPServer struct {
	engine       *gin.Engine
	port         int
	server       *http.Server
	state        *state.GameState
	auth         *AuthStore
	spawnRand    *rand.Rand
	spawnRandMu  sync.Mutex
	gameplay     config.GameplayConfig
	wsRuntime    config.WSRuntimeConfig
	adminToken   string
	adminRuntime AdminRuntime
}

// NewHTTPServer 创建 HTTP 服务并注册路由。
func NewHTTPServer(port int, gameState *state.GameState, auth *AuthStore, gameplay config.GameplayConfig, wsRuntime config.WSRuntimeConfig, adminToken string, adminRuntime AdminRuntime) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	server := &HTTPServer{
		engine:       engine,
		port:         port,
		state:        gameState,
		auth:         auth,
		spawnRand:    rand.New(rand.NewSource(time.Now().UnixNano())),
		gameplay:     gameplay,
		wsRuntime:    wsRuntime.Normalize(),
		adminToken:   adminToken,
		adminRuntime: adminRuntime,
	}

	engine.Use(CORSMiddleware())
	engine.Use(TraceIDMiddleware())
	engine.Use(SlogMiddleware())
	engine.Use(gin.Recovery())

	server.setupRoutes()
	return server
}

// Start 启动 HTTP 服务。
func (s *HTTPServer) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}
	if err := s.server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
	return nil
}

// Shutdown 关闭 HTTP 服务。
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// Addr 返回监听地址。
func (s *HTTPServer) Addr() string {
	return fmt.Sprintf(":%d", s.port)
}

// setupRoutes 注册所有 API 路由。
func (s *HTTPServer) setupRoutes() {
	apiGroup := s.engine.Group("/api")
	v1 := apiGroup.Group("/v1")

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
	})

	auth := v1.Group("/auth")
	auth.POST("/register", s.handleRegister)
	auth.POST("/login", s.handleLogin)
	auth.POST("/refresh", s.handleRefresh)
	auth.POST("/logout", s.handleLogout)

	players := v1.Group("/players")
	players.Use(s.authMiddleware())
	players.GET("/me", s.handleGetMe)
	players.PATCH("/me", s.handlePatchMe)
	players.GET("/:id", s.handleGetPlayer)
	players.GET("/:id/stats", s.handleGetPlayerStats)
	players.GET("/:id/visibility", s.handleGetPlayerVisibility)

	game := v1.Group("")
	game.Use(s.authMiddleware())
	game.GET("/map/entities", s.handleGetMapEntities)
	game.GET("/portals/:id", s.handleGetPortal)
	game.GET("/inventory", s.handleGetInventory)
	game.POST("/inventory/use", s.handleInventoryUse)
	game.POST("/inventory/recycle", s.handleInventoryRecycle)

	admin := v1.Group("/admin")
	admin.Use(s.adminMiddleware())
	admin.GET("/status", s.handleAdminStatus)
	admin.POST("/announce", s.handleAdminAnnounce)
	admin.POST("/kick", s.handleAdminKick)
	admin.POST("/ban", s.handleAdminBan)
	admin.POST("/unban", s.handleAdminUnban)
	admin.POST("/players/:id/faction", s.handleAdminSetFaction)
	admin.POST("/players/:id/level", s.handleAdminSetLevel)
	admin.POST("/players/:id/grant-item", s.handleAdminGrantItem)
	admin.POST("/portals", s.handleAdminAddPortal)
	admin.PATCH("/portals/:id", s.handleAdminUpdatePortal)
	admin.DELETE("/portals/:id", s.handleAdminRemovePortal)
	admin.DELETE("/links/:id", s.handleAdminRemoveLink)
	admin.DELETE("/fields/:id", s.handleAdminRemoveField)
	admin.GET("/logs", s.handleAdminLogs)
	admin.POST("/gc-logs", s.handleAdminGCLogs)
	admin.POST("/recalc-stats", s.handleAdminRecalcStats)
	admin.POST("/reload-config", s.handleAdminReloadConfig)
	admin.POST("/rotate-logs", s.handleAdminRotateLogs)

	v1.GET("/leaderboard", s.handleGetLeaderboard)
	v1.GET("/logs/global", s.handleGetGlobalLogs)
	v1.GET("/config", s.handleGetConfig)

	s.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
	})
}
