package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *HTTPServer) handleAdminSetFaction(c *gin.Context) {
	playerID := c.Param("id")
	var req adminSetFactionRequest
	if err := c.ShouldBindJSON(&req); err != nil || playerID == "" || req.Faction == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	player, ok := s.state.Players.Get(playerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	player.Faction = strings.ToUpper(req.Faction)
	player.UpdatedAt = time.Now()
	s.state.Players.Upsert(player)
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PLAYER",
		PlayerID: player.ID,
		Message:  "faction updated",
	})
	c.JSON(http.StatusOK, SuccessResponse(player))
}

func (s *HTTPServer) handleAdminSetLevel(c *gin.Context) {
	playerID := c.Param("id")
	var req adminSetLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil || playerID == "" || req.Level <= 0 {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	player, ok := s.state.Players.Get(playerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	player.Level = gameplay.NormalizeLevel(req.Level)
	player.MaxXM = gameplay.MaxXMForLevel(player.Level)
	if player.XM > player.MaxXM {
		player.XM = player.MaxXM
	}
	player.UpdatedAt = time.Now()
	s.state.Players.Upsert(player)
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PLAYER",
		PlayerID: player.ID,
		Message:  "level updated",
	})
	c.JSON(http.StatusOK, SuccessResponse(player))
}

func (s *HTTPServer) handleAdminGrantItem(c *gin.Context) {
	playerID := c.Param("id")
	var req adminGrantItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || playerID == "" || req.ItemID == "" || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	player, ok := s.state.Players.Get(playerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	item := gameplay.NormalizeCubeID(req.ItemID)
	if portalID, ok := gameplay.IsKey(item); ok {
		if !gameplay.CanAddKey(player.Inventory, portalID, req.Amount, player.Level) {
			c.JSON(http.StatusBadRequest, BadRequestResponse("key capacity exceeded"))
			return
		}
	} else if !gameplay.CanAddItem(player.Inventory, item, req.Amount, player.Level) {
		c.JSON(http.StatusBadRequest, BadRequestResponse("inventory capacity exceeded"))
		return
	}
	if player.Inventory == nil {
		player.Inventory = make(map[string]int)
	}
	player.Inventory[item] += req.Amount
	player.UpdatedAt = time.Now()
	s.state.Players.Upsert(player)
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PLAYER",
		PlayerID: player.ID,
		Message:  "item granted",
	})
	c.JSON(http.StatusOK, SuccessResponse(player.Inventory))
}
