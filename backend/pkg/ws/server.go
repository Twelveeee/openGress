package ws

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/gorilla/websocket"
)

type Server struct {
	// WS 服务端封装。
	addr     string
	upgrader websocket.Upgrader
	server   *http.Server
	hub      *Hub
	mu       sync.Mutex
}

func NewServer(port int, gameState *state.GameState, gameplay config.GameplayConfig, wsRuntime config.WSRuntimeConfig) *Server {
	// 创建 WS 服务实例。
	return &Server{
		addr: fmt.Sprintf(":%d", port),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		hub: NewHub(gameState, nil, gameplay, wsRuntime),
	}
}

func NewServerWithAuth(port int, gameState *state.GameState, auth *api.AuthStore, gameplay config.GameplayConfig, wsRuntime config.WSRuntimeConfig) *Server {
	// 创建带鉴权的 WS 服务实例。
	server := NewServer(port, gameState, gameplay, wsRuntime)
	server.hub.auth = auth
	return server
}

func (s *Server) Start(ctx context.Context) error {
	// 启动 WS 服务并运行 hub。
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	go s.hub.Run(ctx)

	if err := s.server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	// 关闭 WS 服务。
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) Addr() string {
	return s.addr
}

// Hub 暴露内部 hub，用于管理接口。
func (s *Server) Hub() *Hub {
	return s.hub
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	// 升级连接并绑定到 hub。
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "err", err)
		return
	}

	client := NewClient(conn, s.hub)
	s.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
