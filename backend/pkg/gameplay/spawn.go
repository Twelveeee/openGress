package gameplay

import (
	"log/slog"
	"math/rand"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

// PositionWithinBounds 判断坐标是否在地图边界内。
// 当边界未配置（全 0）时返回 true，保持兼容行为。
func PositionWithinBounds(pos state.Position, bounds config.MapBoundsConfig) bool {
	if !mapBoundsConfigured(bounds) {
		return true
	}
	return pos.Latitude >= bounds.MinLat &&
		pos.Latitude <= bounds.MaxLat &&
		pos.Longitude >= bounds.MinLon &&
		pos.Longitude <= bounds.MaxLon
}

// RandomPositionInBounds 在地图边界内随机生成出生点。
// 当边界未配置或非法时回退到 (0,0)。
func RandomPositionInBounds(bounds config.MapBoundsConfig, rnd *rand.Rand) state.Position {
	if !mapBoundsConfigured(bounds) {
		return state.Position{Latitude: 0, Longitude: 0}
	}
	if bounds.MinLat > bounds.MaxLat || bounds.MinLon > bounds.MaxLon {
		slog.Warn("invalid map bounds for random spawn, fallback to zero",
			"reason", "invalid_bounds",
			"minLat", bounds.MinLat,
			"maxLat", bounds.MaxLat,
			"minLon", bounds.MinLon,
			"maxLon", bounds.MaxLon,
		)
		return state.Position{Latitude: 0, Longitude: 0}
	}
	if rnd == nil {
		rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	lat := bounds.MinLat
	if bounds.MaxLat > bounds.MinLat {
		lat += rnd.Float64() * (bounds.MaxLat - bounds.MinLat)
	}
	lon := bounds.MinLon
	if bounds.MaxLon > bounds.MinLon {
		lon += rnd.Float64() * (bounds.MaxLon - bounds.MinLon)
	}
	return state.Position{Latitude: lat, Longitude: lon}
}

// EnsurePositionInBounds 当坐标越界时返回边界内新坐标，并标记 changed=true。
func EnsurePositionInBounds(pos state.Position, bounds config.MapBoundsConfig, rnd *rand.Rand) (state.Position, bool) {
	if PositionWithinBounds(pos, bounds) {
		return pos, false
	}
	return RandomPositionInBounds(bounds, rnd), true
}

func mapBoundsConfigured(bounds config.MapBoundsConfig) bool {
	return !(bounds.MinLat == 0 && bounds.MaxLat == 0 && bounds.MinLon == 0 && bounds.MaxLon == 0)
}
