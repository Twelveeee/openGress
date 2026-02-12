package api

import (
	"net/http"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/gin-gonic/gin"
)

// handleGetMe 返回当前登录玩家信息。
func (s *HTTPServer) handleGetMe(c *gin.Context) {
	user := c.MustGet(contextUserKey).(User)
	player, ok := s.state.Players.Get(user.PlayerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	player.Level = gameplay.LevelForAP(player.AP)
	player.MaxXM = gameplay.MaxXMForLevel(player.Level)
	if player.XM > player.MaxXM {
		player.XM = player.MaxXM
	}
	s.state.Players.Upsert(player)
	c.JSON(http.StatusOK, SuccessResponse(player))
}

// handlePatchMe 更新当前玩家配置（如自动 hack）。
func (s *HTTPServer) handlePatchMe(c *gin.Context) {
	user := c.MustGet(contextUserKey).(User)
	player, ok := s.state.Players.Get(user.PlayerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	var payload struct {
		AutoHack *bool `json:"autoHack"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	if payload.AutoHack != nil {
		player.AutoHack = *payload.AutoHack
	}
	player.UpdatedAt = time.Now()
	s.state.Players.Upsert(player)
	c.JSON(http.StatusOK, SuccessResponse(player))
}

// handleGetPlayer 查询指定玩家。
func (s *HTTPServer) handleGetPlayer(c *gin.Context) {
	playerID := c.Param("id")
	player, ok := s.state.Players.Get(playerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	c.JSON(http.StatusOK, SuccessResponse(player))
}

// handleGetPlayerStats 返回玩家统计（占位）。
func (s *HTTPServer) handleGetPlayerStats(c *gin.Context) {
	playerID := c.Param("id")
	player, ok := s.state.Players.Get(playerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"portalCaptures": player.PortalCaptures,
		"linksCreated":   player.LinksCreated,
		"fieldsCreated":  player.FieldsCreated,
		"muTotal":        player.MUTotal,
	}))
}

// handleGetPlayerVisibility 返回在线与最后活动信息。
func (s *HTTPServer) handleGetPlayerVisibility(c *gin.Context) {
	playerID := c.Param("id")
	player, ok := s.state.Players.Get(playerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"online":     true,
		"lastActive": player.UpdatedAt.UnixMilli(),
		"position":   player.Position,
	}))
}
