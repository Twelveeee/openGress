package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *HTTPServer) handleAdminStatus(c *gin.Context) {
	logCount := 0
	if s.state.Logs != nil {
		logCount = len(s.state.Logs.List())
	}
	banned := 0
	if s.state.Admin != nil {
		banned = len(s.state.Admin.Banned)
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"players": len(s.state.Players.List()),
		"portals": len(s.state.Portals.List()),
		"links":   len(s.state.Links.List()),
		"fields":  len(s.state.Fields.List()),
		"logs":    logCount,
		"banned":  banned,
		"time":    time.Now().UnixMilli(),
	}))
}

func (s *HTTPServer) handleAdminAnnounce(c *gin.Context) {
	var req adminAnnounceRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	appendLog(s.state, state.LogEntry{
		ID:      uuid.New().String(),
		Type:    "ANNOUNCE",
		Message: req.Message,
	})
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func (s *HTTPServer) handleAdminKick(c *gin.Context) {
	var req adminPlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	kicked := false
	if s.adminRuntime != nil {
		kicked = s.adminRuntime.KickPlayer(req.PlayerID)
	}
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "KICK",
		PlayerID: req.PlayerID,
		Message:  "player kicked",
	})
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"kicked": kicked}))
}

func (s *HTTPServer) handleAdminBan(c *gin.Context) {
	var req adminBanRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	if s.state.Admin == nil {
		s.state.Admin = state.NewAdminState()
	}
	s.state.Admin.Banned[req.PlayerID] = req.Reason
	if s.adminRuntime != nil {
		_ = s.adminRuntime.KickPlayer(req.PlayerID)
	}
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "BAN",
		PlayerID: req.PlayerID,
		Message:  "player banned",
	})
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func (s *HTTPServer) handleAdminUnban(c *gin.Context) {
	var req adminPlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	if s.state.Admin != nil {
		delete(s.state.Admin.Banned, req.PlayerID)
	}
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "UNBAN",
		PlayerID: req.PlayerID,
		Message:  "player unbanned",
	})
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func (s *HTTPServer) handleAdminLogs(c *gin.Context) {
	logs := []state.LogEntry{}
	if s.state.Logs != nil {
		logs = s.state.Logs.List()
	}
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	if limit > 0 && len(logs) > limit {
		logs = logs[len(logs)-limit:]
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"logs": logs}))
}

func (s *HTTPServer) handleAdminGCLogs(c *gin.Context) {
	var req adminGCLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Keep <= 0 {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	logs := []state.LogEntry{}
	if s.state.Logs != nil {
		logs = s.state.Logs.List()
	}
	if len(logs) > req.Keep {
		logs = logs[len(logs)-req.Keep:]
	}
	s.state.Logs = state.NewLogStore(len(logs))
	for _, entry := range logs {
		s.state.Logs.Add(entry)
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func (s *HTTPServer) handleAdminRecalcStats(c *gin.Context) {
	players := s.state.Players.List()
	for _, player := range players {
		player.PortalCaptures = 0
		player.LinksCreated = 0
		player.FieldsCreated = 0
		player.MUTotal = 0
		player.AP = 0
		player.Level = 1
		player.MaxXM = gameplay.MaxXMForLevel(1)
		if player.XM > player.MaxXM {
			player.XM = player.MaxXM
		}
		s.state.Players.Upsert(player)
	}
	logs := []state.LogEntry{}
	if s.state.Logs != nil {
		logs = s.state.Logs.List()
	}
	for _, entry := range logs {
		player, ok := s.state.Players.Get(entry.PlayerID)
		if !ok {
			continue
		}
		switch entry.Type {
		case "CAPTURE":
			player.PortalCaptures++
			player.AP += gameplay.APRewardCapturePortal(s.gameplay)
		case "LINK":
			player.LinksCreated++
			player.AP += gameplay.APRewardCreateLink(s.gameplay)
		case "FIELD":
			player.FieldsCreated++
			player.MUTotal += entry.MU
			player.AP += gameplay.APRewardCreateField(s.gameplay)
		}
		player.Level = gameplay.LevelForAP(player.AP)
		player.MaxXM = gameplay.MaxXMForLevel(player.Level)
		if player.XM > player.MaxXM {
			player.XM = player.MaxXM
		}
		s.state.Players.Upsert(player)
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func (s *HTTPServer) handleAdminReloadConfig(c *gin.Context) {
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func (s *HTTPServer) handleAdminRotateLogs(c *gin.Context) {
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}
