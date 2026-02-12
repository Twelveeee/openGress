package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *HTTPServer) handleAdminAddPortal(c *gin.Context) {
	var req adminPortalRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	portal := state.Portal{
		ID:        req.ID,
		Title:     strings.TrimSpace(req.Title),
		CoverURL:  strings.TrimSpace(req.CoverURL),
		Position:  state.Position{Latitude: req.Lat, Longitude: req.Lon},
		Faction:   strings.ToUpper(req.Faction),
		UpdatedAt: time.Now(),
	}
	if portal.Title == "" {
		portal.Title = portal.ID
	}
	s.state.Portals.Upsert(portal)
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PORTAL",
		PortalID: portal.ID,
		Message:  "portal added",
	})
	c.JSON(http.StatusOK, SuccessResponse(portal))
}

func (s *HTTPServer) handleAdminUpdatePortal(c *gin.Context) {
	portalID := c.Param("id")
	var req adminPortalPatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || portalID == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	portal, ok := s.state.Portals.Get(portalID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "portal not found"))
		return
	}
	if req.Lat != nil && req.Lon != nil {
		portal.Position = state.Position{Latitude: *req.Lat, Longitude: *req.Lon}
	}
	if req.Title != nil {
		portal.Title = strings.TrimSpace(*req.Title)
		if portal.Title == "" {
			portal.Title = portal.ID
		}
	}
	if req.CoverURL != nil {
		portal.CoverURL = strings.TrimSpace(*req.CoverURL)
	}
	if req.Faction != nil {
		portal.Faction = strings.ToUpper(*req.Faction)
	}
	if req.Level != nil {
		portal.Level = *req.Level
	}
	if req.Energy != nil {
		portal.Energy = *req.Energy
	}
	portal.UpdatedAt = time.Now()
	s.state.Portals.Upsert(portal)
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PORTAL",
		PortalID: portal.ID,
		Message:  "portal updated",
	})
	c.JSON(http.StatusOK, SuccessResponse(portal))
}

func (s *HTTPServer) handleAdminRemovePortal(c *gin.Context) {
	portalID := c.Param("id")
	if portalID == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	s.state.Portals.Remove(portalID)
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PORTAL",
		PortalID: portalID,
		Message:  "portal removed",
	})
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func (s *HTTPServer) handleAdminRemoveLink(c *gin.Context) {
	linkID := c.Param("id")
	if linkID == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	s.state.Links.Remove(linkID)
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "LINK",
		PortalID: linkID,
		Message:  "link removed",
	})
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}

func (s *HTTPServer) handleAdminRemoveField(c *gin.Context) {
	fieldID := c.Param("id")
	if fieldID == "" {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid request"))
		return
	}
	s.state.Fields.Remove(fieldID)
	appendLog(s.state, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "FIELD",
		PortalID: fieldID,
		Message:  "field removed",
	})
	c.JSON(http.StatusOK, SuccessResponse(gin.H{"status": "ok"}))
}
