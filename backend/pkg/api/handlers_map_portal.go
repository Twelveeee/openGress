package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/geo"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/gin-gonic/gin"
)

// handleGetMapEntities 根据 bounds 返回可见实体。
func (s *HTTPServer) handleGetMapEntities(c *gin.Context) {
	bounds, err := parseBounds(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, BadRequestResponse("invalid bounds"))
		return
	}
	if !boundsWithinMap(bounds, s.gameplay.MapBounds) {
		c.JSON(http.StatusBadRequest, BadRequestResponse("bounds out of map"))
		return
	}
	portals := s.state.Portals.List()
	visiblePortals := make([]state.Portal, 0, len(portals))
	for _, portal := range portals {
		if pointInBounds(portal.Position, bounds) {
			visiblePortals = append(visiblePortals, portal)
		}
	}
	links := s.state.Links.List()
	visibleLinks := make([]state.Link, 0, len(links))
	for _, link := range links {
		if linkVisibleInBounds(link.FromPosition, link.ToPosition, bounds) {
			visibleLinks = append(visibleLinks, link)
		}
	}
	fields := s.state.Fields.List()
	visibleFields := make([]state.Field, 0, len(fields))
	for _, field := range fields {
		p1, ok1 := s.state.Portals.Get(field.PortalIDs[0])
		p2, ok2 := s.state.Portals.Get(field.PortalIDs[1])
		p3, ok3 := s.state.Portals.Get(field.PortalIDs[2])
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		if pointInBounds(p1.Position, bounds) || pointInBounds(p2.Position, bounds) || pointInBounds(p3.Position, bounds) ||
			linkVisibleInBounds(p1.Position, p2.Position, bounds) || linkVisibleInBounds(p2.Position, p3.Position, bounds) || linkVisibleInBounds(p3.Position, p1.Position, bounds) {
			visibleFields = append(visibleFields, field)
		}
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"portals": visiblePortals,
		"links":   visibleLinks,
		"fields":  visibleFields,
	}))
}

// handleGetPortal 返回 Portal 详情（支持 legacy 格式）。
func (s *HTTPServer) handleGetPortal(c *gin.Context) {
	portalID := c.Param("id")
	portal, ok := s.state.Portals.Get(portalID)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse(ErrCodeNotFound, "portal not found"))
		return
	}
	if c.Query("format") == "legacy" {
		c.JSON(http.StatusOK, SuccessResponse(gin.H{
			"result": buildLegacyPortalDetails(portal),
		}))
		return
	}
	c.JSON(http.StatusOK, SuccessResponse(portal))
}

// buildLegacyPortalDetails 兼容旧版门户详情格式。
func buildLegacyPortalDetails(portal state.Portal) []interface{} {
	latE6 := int64(portal.Position.Latitude * 1e6)
	lonE6 := int64(portal.Position.Longitude * 1e6)
	title := portal.Title
	if title == "" {
		title = portal.ID
	}
	ts := portal.UpdatedAt.UnixMilli()
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	return []interface{}{
		"p",
		"N",
		latE6,
		lonE6,
		1,
		0,
		0,
		portal.CoverURL,
		title,
		[]interface{}{},
		false,
		false,
		nil,
		ts,
		[]interface{}{nil, nil, nil, nil},
		[]interface{}{},
		"",
		[]interface{}{"", "", []interface{}{}},
	}
}

// parseBounds 解析视野 bounds 查询参数。
func parseBounds(c *gin.Context) (state.Bounds, error) {
	minLat, err := parseFloatQuery(c, "minLat")
	if err != nil {
		return state.Bounds{}, err
	}
	maxLat, err := parseFloatQuery(c, "maxLat")
	if err != nil {
		return state.Bounds{}, err
	}
	minLon, err := parseFloatQuery(c, "minLon")
	if err != nil {
		return state.Bounds{}, err
	}
	maxLon, err := parseFloatQuery(c, "maxLon")
	if err != nil {
		return state.Bounds{}, err
	}
	return state.Bounds{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLon: minLon,
		MaxLon: maxLon,
	}, nil
}

func boundsWithinMap(bounds state.Bounds, mapBounds config.MapBoundsConfig) bool {
	if mapBounds.MinLat == 0 && mapBounds.MaxLat == 0 && mapBounds.MinLon == 0 && mapBounds.MaxLon == 0 {
		return true
	}
	if bounds.MinLat < mapBounds.MinLat || bounds.MaxLat > mapBounds.MaxLat {
		return false
	}
	if bounds.MinLon < mapBounds.MinLon || bounds.MaxLon > mapBounds.MaxLon {
		return false
	}
	return true
}

// parseFloatQuery 解析浮点型 query 参数。
func parseFloatQuery(c *gin.Context, key string) (float64, error) {
	raw := c.Query(key)
	return strconv.ParseFloat(raw, 64)
}

// linkVisibleInBounds 判断连线是否与 bounds 相交。
func linkVisibleInBounds(from, to state.Position, bounds state.Bounds) bool {
	return geo.LinkVisibleInBounds(from, to, bounds)
}

// pointInBounds 判断点是否在 bounds 内。
func pointInBounds(pos state.Position, bounds state.Bounds) bool {
	return geo.PointInBounds(pos, bounds)
}
