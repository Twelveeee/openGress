package api

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/gin-gonic/gin"
)

// handleGetLeaderboard 返回排行榜（占位）。
func (s *HTTPServer) handleGetLeaderboard(c *gin.Context) {
	players := s.state.Players.List()
	sort.Slice(players, func(i, j int) bool {
		return players[i].AP > players[j].AP
	})
	c.JSON(http.StatusOK, SuccessResponse(players))
}

// handleGetGlobalLogs 返回全局日志（占位）。
func (s *HTTPServer) handleGetGlobalLogs(c *gin.Context) {
	logs := []state.LogEntry{}
	if s.state.Logs != nil {
		logs = s.state.Logs.List()
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"logs": logs,
	}))
}

// handleGetConfig 返回客户端配置（占位）。
func (s *HTTPServer) handleGetConfig(c *gin.Context) {
	viewRadius := s.gameplay.ViewRadiusM
	if viewRadius <= 0 {
		viewRadius = 400
	}
	c.JSON(http.StatusOK, SuccessResponse(gin.H{
		"viewRadiusMeters": viewRadius,
		"mapTickMs":        s.wsRuntime.MapTickMS,
		"attackSpecs":      buildAttackSpecsPayload(s.gameplay),
		"wsCadence": gin.H{
			"movementTickMs":      s.wsRuntime.MovementTickMS,
			"mapTickMs":           s.wsRuntime.MapTickMS,
			"playerStatePushMs":   s.wsRuntime.PlayerStatePushMS,
			"nearbyPlayersPushMs": s.wsRuntime.NearbyPlayersPushMS,
		},
	}))
}

func buildAttackSpecsPayload(cfg config.GameplayConfig) gin.H {
	result := gin.H{
		"XMP": gin.H{},
		"US":  gin.H{},
	}
	defaultRadiusXMP := map[int]float64{1: 42, 2: 48, 3: 58, 4: 72, 5: 90, 6: 112, 7: 138, 8: 168}
	defaultRadiusUS := map[int]float64{1: 10, 2: 13, 3: 16, 4: 18, 5: 21, 6: 24, 7: 27, 8: 30}

	types := []string{"XMP", "US"}
	for _, weaponType := range types {
		levelMap := result[weaponType].(gin.H)
		specsByLevel := cfg.Attack.WeaponSpecs[weaponType]
		for level := 1; level <= 8; level++ {
			spec := specsByLevel[level]
			radius := spec.RadiusM
			if radius <= 0 {
				if weaponType == "XMP" {
					radius = defaultRadiusXMP[level]
				} else {
					radius = defaultRadiusUS[level]
				}
			}
			cost := spec.CostXM
			if cost <= 0 {
				cost = gameplay.AttackCost(cfg, weaponType, level)
			}
			levelMap[strconv.Itoa(level)] = gin.H{
				"radiusM": radius,
				"costXm":  cost,
			}
		}
	}
	return result
}
