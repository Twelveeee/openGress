package api

import (
	"net/http"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/gin-gonic/gin"
)

type inventoryRequest struct {
	ItemType string `json:"itemType"`
	Amount   int    `json:"amount"`
}

type inventoryUseResponse struct {
	APGained int `json:"apGained"`
	XMDelta  int `json:"xmDelta"`
}

// handleGetInventory 返回玩家背包。
func (s *HTTPServer) handleGetInventory(c *gin.Context) {
	user := c.MustGet(contextUserKey).(User)
	player, ok := s.state.Players.Get(user.PlayerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	c.JSON(http.StatusOK, SuccessResponse(player.Inventory))
}

// handleInventoryUse 使用背包道具。
func (s *HTTPServer) handleInventoryUse(c *gin.Context) {
	user := c.MustGet(contextUserKey).(User)
	player, ok := s.state.Players.Get(user.PlayerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	var req inventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemType == "" || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	itemID := gameplay.NormalizeCubeID(req.ItemType)
	kind, level, ok := gameplay.ParseLevelItem(itemID)
	if !ok || kind != "CUBE" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("unsupported item"))
		return
	}
	player.Level = gameplay.LevelForAP(player.AP)
	player.MaxXM = gameplay.MaxXMForLevel(player.Level)
	if player.Level < level {
		c.JSON(http.StatusBadRequest, BadRequestResponse("player level too low"))
		return
	}
	if player.Inventory[itemID] < req.Amount {
		c.JSON(http.StatusBadRequest, BadRequestResponse("insufficient items"))
		return
	}
	prevXM := player.XM
	player.Inventory[itemID] -= req.Amount
	player.XM += gameplay.CubeXM(level) * req.Amount
	if player.XM > player.MaxXM {
		player.XM = player.MaxXM
	}
	player.UpdatedAt = time.Now()
	s.state.Players.Upsert(player)
	c.JSON(http.StatusOK, SuccessResponse(inventoryUseResponse{
		APGained: 0,
		XMDelta:  player.XM - prevXM,
	}))
}

// handleInventoryRecycle 回收道具。
func (s *HTTPServer) handleInventoryRecycle(c *gin.Context) {
	user := c.MustGet(contextUserKey).(User)
	player, ok := s.state.Players.Get(user.PlayerID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "player not found"))
		return
	}
	var req inventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemType == "" || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	itemID := gameplay.NormalizeInventoryItemID(req.ItemType)
	if player.Inventory[itemID] < req.Amount {
		c.JSON(http.StatusBadRequest, BadRequestResponse("insufficient items"))
		return
	}
	player.Inventory[itemID] -= req.Amount
	player.XM += 100 * req.Amount
	if player.XM > player.MaxXM {
		player.XM = player.MaxXM
	}
	player.UpdatedAt = time.Now()
	s.state.Players.Upsert(player)
	c.JSON(http.StatusOK, SuccessResponse(player.Inventory))
}
