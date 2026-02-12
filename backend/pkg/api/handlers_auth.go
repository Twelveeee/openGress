package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/gin-gonic/gin"
)

// gin context 中保存鉴权用户的 key。
const contextUserKey = "auth_user"

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Faction  string `json:"faction"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *HTTPServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 统一的 JWT 鉴权入口（MVP）。
		token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer"))
		if token == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse(ErrCodeUnauthorized, "missing token"))
			c.Abort()
			return
		}
		user, ok := s.auth.UserByAccessToken(token)
		if !ok {
			c.JSON(http.StatusUnauthorized, ErrorResponse(ErrCodeUnauthorized, "invalid token"))
			c.Abort()
			return
		}
		if s.state.Admin != nil {
			if _, banned := s.state.Admin.Banned[user.PlayerID]; banned {
				c.JSON(http.StatusUnauthorized, ErrorResponse(ErrCodeUnauthorized, "banned"))
				c.Abort()
				return
			}
		}
		c.Set(contextUserKey, user)
		c.Next()
	}
}

// handleRegister 注册并签发 JWT。
func (s *HTTPServer) handleRegister(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	user, err := s.auth.Register(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserExists):
			c.JSON(http.StatusBadRequest, BadRequestResponse("user exists"))
		case errors.Is(err, ErrInvalidCredentials):
			c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		default:
			c.JSON(http.StatusInternalServerError, InternalErrorResponse("register failed"))
		}
		return
	}
	player := state.Player{
		ID:        user.PlayerID,
		Username:  user.Username,
		Faction:   strings.ToUpper(req.Faction),
		Level:     1,
		XM:        gameplay.MaxXMForLevel(1),
		MaxXM:     gameplay.MaxXMForLevel(1),
		Inventory: starterInventory(),
		Position:  s.randomSpawn(),
		UpdatedAt: time.Now(),
	}
	s.state.Players.Upsert(player)
	access, refresh, err := s.auth.IssueTokens(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, InternalErrorResponse("token error"))
		return
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"player_id":     user.PlayerID,
	}))
}

// handleLogin 登录并签发 JWT。
func (s *HTTPServer) handleLogin(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	user, err := s.auth.Authenticate(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse(ErrCodeUnauthorized, "invalid credentials"))
		return
	}
	player, exists := s.state.Players.Get(user.PlayerID)
	if !exists {
		player = state.Player{
			ID:        user.PlayerID,
			Username:  user.Username,
			Faction:   "NEUTRAL",
			Level:     1,
			XM:        gameplay.MaxXMForLevel(1),
			MaxXM:     gameplay.MaxXMForLevel(1),
			Inventory: starterInventory(),
			Position:  s.randomSpawn(),
			UpdatedAt: time.Now(),
		}
		s.state.Players.Upsert(player)
	} else {
		reason := ""
		switch {
		case mapBoundsConfigured(s.gameplay.MapBounds) && isZeroPosition(player.Position):
			reason = "zero_position"
		case !gameplay.PositionWithinBounds(player.Position, s.gameplay.MapBounds):
			reason = "out_of_bounds"
		}
		if reason != "" {
			repaired, changed := s.ensurePlayerSpawn(player, reason)
			if changed {
				s.state.Players.Upsert(repaired)
			}
		}
	}
	access, refresh, err := s.auth.IssueTokens(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, InternalErrorResponse("token error"))
		return
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"player_id":     user.PlayerID,
	}))
}

// handleRefresh 刷新 access_token。
func (s *HTTPServer) handleRefresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	access, ok := s.auth.Refresh(req.RefreshToken)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse(ErrCodeUnauthorized, "invalid refresh token"))
		return
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"access_token": access,
	}))
}

// handleLogout 退出登录（MVP 只处理 refresh）。
func (s *HTTPServer) handleLogout(c *gin.Context) {
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer"))
	if token != "" {
		s.auth.Revoke(token)
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func starterInventory() map[string]int {
	return map[string]int{
		gameplay.ItemIDResonator(1): 8,
		gameplay.ItemIDXMP(1):       8,
		gameplay.ItemIDCube(1):      3,
	}
}

func (s *HTTPServer) randomSpawn() state.Position {
	s.spawnRandMu.Lock()
	defer s.spawnRandMu.Unlock()
	return gameplay.RandomPositionInBounds(s.gameplay.MapBounds, s.spawnRand)
}

func (s *HTTPServer) ensurePlayerSpawn(player state.Player, reason string) (state.Player, bool) {
	if !mapBoundsConfigured(s.gameplay.MapBounds) {
		return player, false
	}
	oldPos := player.Position
	player.Position = s.randomSpawn()
	player.UpdatedAt = time.Now()
	slog.Info("repair player position",
		"playerId", player.ID,
		"oldPos", oldPos,
		"newPos", player.Position,
		"reason", reason,
	)
	return player, true
}

func mapBoundsConfigured(bounds config.MapBoundsConfig) bool {
	return !(bounds.MinLat == 0 && bounds.MaxLat == 0 && bounds.MinLon == 0 && bounds.MaxLon == 0)
}

func isZeroPosition(pos state.Position) bool {
	return pos.Latitude == 0 && pos.Longitude == 0
}
