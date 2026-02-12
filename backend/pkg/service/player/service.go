package player

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	pbws "github.com/Twelveeee/openGress/backend/pkg/pb/ws"
	"github.com/Twelveeee/openGress/backend/pkg/service/errors"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

const (
	MoveSpeedKmH         = 120.0
	EarthRadiusM         = 6371000.0
	MetersPerDegLat      = 111_320.0
	DefaultNearbyRadiusM = 400.0
)

type StateSnapshot = pbws.PlayerStatePayload
type NearbySnapshot = pbws.NearbyPlayer

type Service struct {
	gameState *state.GameState
	cfg       config.GameplayConfig
}

func New(gameState *state.GameState, cfg config.GameplayConfig) *Service {
	return &Service{gameState: gameState, cfg: cfg}
}

func (s *Service) GetOrCreate(playerID string) state.Player {
	if s.gameState == nil {
		return state.Player{}
	}
	if player, ok := s.gameState.Players.Get(playerID); ok {
		return player
	}
	spawn := gameplay.RandomPositionInBounds(s.cfg.MapBounds, nil)
	return state.Player{
		ID:            playerID,
		Username:      playerID,
		Faction:       "NEUTRAL",
		Level:         1,
		XM:            gameplay.MaxXMForLevel(1),
		MaxXM:         gameplay.MaxXMForLevel(1),
		Inventory:     make(map[string]int),
		Position:      state.Position{Latitude: spawn.Latitude, Longitude: spawn.Longitude},
		HackCooldowns: make(map[string]time.Time),
		UpdatedAt:     time.Now(),
	}
}

func (s *Service) Upsert(player state.Player) {
	if s.gameState == nil {
		return
	}
	s.gameState.Players.Upsert(player)
}

func (s *Service) SetTarget(playerID string, target state.Position) error {
	if playerID == "" {
		return errors.New(errors.KindBadRequest, "missing playerId")
	}
	if !s.PositionWithinMap(target) {
		return errors.New(errors.KindBadRequest, "target out of map")
	}
	player := s.GetOrCreate(playerID)
	player.Target = &target
	player.UpdatedAt = time.Now()
	s.Upsert(player)
	return nil
}

func (s *Service) UpdateView(playerID string, bounds *state.Bounds, tiles []string) (*state.Bounds, error) {
	if playerID == "" {
		return nil, errors.New(errors.KindBadRequest, "missing playerId")
	}
	if bounds == nil && len(tiles) == 0 {
		return nil, errors.New(errors.KindBadRequest, "missing bounds or tiles")
	}

	player := s.GetOrCreate(playerID)
	var normalized *state.Bounds
	if bounds != nil {
		if !s.PositionWithinMap(player.Position) {
			return nil, errors.New(errors.KindBadRequest, "player out of map")
		}
		if !s.BoundsWithinMap(*bounds) {
			return nil, errors.New(errors.KindBadRequest, "view out of map")
		}
		maxRadius := s.ViewRadius()
		normalized = NormalizeViewBounds(player.Position, *bounds, maxRadius)
		clamped := s.ClampBoundsToMap(*normalized)
		normalized = &clamped
		player.View = normalized
	}
	if len(tiles) > 0 {
		player.ViewTiles = append([]string(nil), tiles...)
	}
	player.UpdatedAt = time.Now()
	s.Upsert(player)
	return normalized, nil
}

func (s *Service) AdvanceMovement(stepMeters float64, now time.Time) (map[string]state.Player, map[string]bool) {
	updated := make(map[string]state.Player)
	moved := make(map[string]bool)
	if s.gameState == nil {
		return updated, moved
	}
	players := s.gameState.Players.List()
	for _, p := range players {
		next, ok := MovePlayerTowards(p, stepMeters)
		if ok {
			next.UpdatedAt = now
			s.gameState.Players.Upsert(next)
			moved[next.ID] = true
		}
		updated[next.ID] = next
	}
	return updated, moved
}

func (s *Service) BuildPlayerState(player state.Player, horizon time.Duration) StateSnapshot {
	speed := 0.0
	heading := 0.0
	if player.Target != nil {
		speed = SpeedMPS()
		heading = ComputeHeadingDeg(player.Position, *player.Target)
	}
	renderPos := player.Position
	preRender := PreRenderPosition(player, horizon)
	return StateSnapshot{
		PlayerID:           player.ID,
		Latitude:           renderPos.Latitude,
		Longitude:          renderPos.Longitude,
		RenderLatitude:     renderPos.Latitude,
		RenderLongitude:    renderPos.Longitude,
		PreRenderLatitude:  preRender.Latitude,
		PreRenderLongitude: preRender.Longitude,
		SpeedMps:           speed,
		HeadingDeg:         heading,
	}
}

func (s *Service) BuildNearbyPlayers(self state.Player, all map[string]state.Player, radius float64, horizon time.Duration) []NearbySnapshot {
	if radius <= 0 {
		radius = DefaultNearbyRadiusM
	}
	nearby := make([]NearbySnapshot, 0, len(all))
	for _, other := range all {
		if other.ID == self.ID {
			continue
		}
		if HaversineMeters(self.Position, other.Position) > radius {
			continue
		}
		renderPos := other.Position
		preRenderPos := PreRenderPosition(other, horizon)
		nearby = append(nearby, NearbySnapshot{
			ID:                 other.ID,
			Latitude:           renderPos.Latitude,
			Longitude:          renderPos.Longitude,
			RenderLatitude:     renderPos.Latitude,
			RenderLongitude:    renderPos.Longitude,
			PreRenderLatitude:  preRenderPos.Latitude,
			PreRenderLongitude: preRenderPos.Longitude,
		})
	}
	sort.Slice(nearby, func(i, j int) bool { return nearby[i].ID < nearby[j].ID })
	return nearby
}

func (s *Service) ViewRadius() float64 {
	if s.cfg.ViewRadiusM > 0 {
		return s.cfg.ViewRadiusM
	}
	return DefaultNearbyRadiusM
}

func (s *Service) InteractRange() float64 {
	if s.cfg.InteractRangeM > 0 {
		return s.cfg.InteractRangeM
	}
	return 40
}

func (s *Service) WithinInteractRange(a, b state.Position) bool {
	return HaversineMeters(a, b) <= s.InteractRange()
}

func (s *Service) PositionWithinMap(pos state.Position) bool {
	return gameplay.PositionWithinBounds(pos, s.cfg.MapBounds)
}

func (s *Service) BoundsWithinMap(bounds state.Bounds) bool {
	if !MapBoundsConfigured(s.cfg.MapBounds) {
		return true
	}
	if bounds.MinLat < s.cfg.MapBounds.MinLat || bounds.MaxLat > s.cfg.MapBounds.MaxLat {
		return false
	}
	if bounds.MinLon < s.cfg.MapBounds.MinLon || bounds.MaxLon > s.cfg.MapBounds.MaxLon {
		return false
	}
	return true
}

func (s *Service) ClampBoundsToMap(bounds state.Bounds) state.Bounds {
	if !MapBoundsConfigured(s.cfg.MapBounds) {
		return bounds
	}
	if bounds.MinLat < s.cfg.MapBounds.MinLat {
		bounds.MinLat = s.cfg.MapBounds.MinLat
	}
	if bounds.MaxLat > s.cfg.MapBounds.MaxLat {
		bounds.MaxLat = s.cfg.MapBounds.MaxLat
	}
	if bounds.MinLon < s.cfg.MapBounds.MinLon {
		bounds.MinLon = s.cfg.MapBounds.MinLon
	}
	if bounds.MaxLon > s.cfg.MapBounds.MaxLon {
		bounds.MaxLon = s.cfg.MapBounds.MaxLon
	}
	return bounds
}

func MapBoundsConfigured(bounds config.MapBoundsConfig) bool {
	return !(bounds.MinLat == 0 && bounds.MaxLat == 0 && bounds.MinLon == 0 && bounds.MaxLon == 0)
}

func NormalizeViewBounds(center state.Position, bounds state.Bounds, maxRadiusMeters float64) *state.Bounds {
	centerLat := center.Latitude
	centerLon := center.Longitude
	halfLatMeters := math.Abs(bounds.MaxLat-bounds.MinLat) * MetersPerDegLat / 2
	halfLonMeters := math.Abs(bounds.MaxLon-bounds.MinLon) * MetersPerDegLon(centerLat) / 2
	maxHalf := math.Max(halfLatMeters, halfLonMeters)
	if maxHalf <= maxRadiusMeters {
		return &bounds
	}
	clampedHalfLat := maxRadiusMeters / MetersPerDegLat
	clampedHalfLon := maxRadiusMeters / MetersPerDegLon(centerLat)
	return &state.Bounds{
		MinLat: centerLat - clampedHalfLat,
		MaxLat: centerLat + clampedHalfLat,
		MinLon: centerLon - clampedHalfLon,
		MaxLon: centerLon + clampedHalfLon,
	}
}

func SpeedMPS() float64 {
	return MoveSpeedKmH * 1000.0 / 3600.0
}

func MovePlayerTowards(player state.Player, stepMeters float64) (state.Player, bool) {
	if player.Target == nil {
		return player, false
	}
	target := *player.Target
	dist := HaversineMeters(player.Position, target)
	if dist <= stepMeters {
		player.Position = target
		player.Target = nil
		return player, true
	}
	player.Position = MoveTowards(player.Position, target, stepMeters)
	return player, true
}

func PreRenderPosition(player state.Player, horizon time.Duration) state.Position {
	if player.Target == nil || horizon <= 0 {
		return player.Position
	}
	target := *player.Target
	dist := HaversineMeters(player.Position, target)
	step := SpeedMPS() * horizon.Seconds()
	if step <= 0 || dist <= step {
		return target
	}
	return MoveTowards(player.Position, target, step)
}

func ComputeHeadingDeg(from, to state.Position) float64 {
	lat1 := degToRad(from.Latitude)
	lat2 := degToRad(to.Latitude)
	dLon := degToRad(to.Longitude - from.Longitude)

	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	heading := math.Atan2(y, x)
	return math.Mod(radToDeg(heading)+360.0, 360.0)
}

func HaversineMeters(a, b state.Position) float64 {
	lat1 := degToRad(a.Latitude)
	lat2 := degToRad(b.Latitude)
	dLat := lat2 - lat1
	dLon := degToRad(b.Longitude - a.Longitude)

	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2 * EarthRadiusM * math.Asin(math.Sqrt(h))
}

func MoveTowards(from, to state.Position, distanceMeters float64) state.Position {
	lat1 := degToRad(from.Latitude)
	lon1 := degToRad(from.Longitude)
	lat2 := degToRad(to.Latitude)
	lon2 := degToRad(to.Longitude)

	heading := bearingRad(lat1, lon1, lat2, lon2)
	angular := distanceMeters / EarthRadiusM

	newLat := math.Asin(math.Sin(lat1)*math.Cos(angular) + math.Cos(lat1)*math.Sin(angular)*math.Cos(heading))
	newLon := lon1 + math.Atan2(
		math.Sin(heading)*math.Sin(angular)*math.Cos(lat1),
		math.Cos(angular)-math.Sin(lat1)*math.Sin(newLat),
	)

	return state.Position{Latitude: radToDeg(newLat), Longitude: radToDeg(newLon)}
}

func MoveByBearing(from state.Position, bearingDeg, distanceMeters float64) state.Position {
	lat1 := degToRad(from.Latitude)
	lon1 := degToRad(from.Longitude)
	heading := degToRad(bearingDeg)
	angular := distanceMeters / EarthRadiusM

	newLat := math.Asin(math.Sin(lat1)*math.Cos(angular) + math.Cos(lat1)*math.Sin(angular)*math.Cos(heading))
	newLon := lon1 + math.Atan2(
		math.Sin(heading)*math.Sin(angular)*math.Cos(lat1),
		math.Cos(angular)-math.Sin(lat1)*math.Sin(newLat),
	)
	return state.Position{Latitude: radToDeg(newLat), Longitude: radToDeg(newLon)}
}

func MetersPerDegLon(lat float64) float64 {
	return MetersPerDegLat * math.Cos(lat*math.Pi/180.0)
}

func DefaultSpawnPosition(mapBounds config.MapBoundsConfig) state.Position {
	if !MapBoundsConfigured(mapBounds) {
		return state.Position{Latitude: 0, Longitude: 0}
	}
	return state.Position{Latitude: (mapBounds.MinLat + mapBounds.MaxLat) / 2, Longitude: (mapBounds.MinLon + mapBounds.MaxLon) / 2}
}

func ValidateViewUpdateInput(playerID string, bounds *state.Bounds, tiles []string) error {
	if playerID == "" {
		return errors.New(errors.KindBadRequest, "missing playerId")
	}
	if bounds == nil && len(tiles) == 0 {
		return errors.New(errors.KindBadRequest, "missing bounds or tiles")
	}
	return nil
}

func MustParseBounds(bounds *pbws.ViewBoundsPayload) (*state.Bounds, error) {
	if bounds == nil {
		return nil, nil
	}
	if bounds.MinLat > bounds.MaxLat || bounds.MinLon > bounds.MaxLon {
		return nil, fmt.Errorf("invalid bounds")
	}
	return &state.Bounds{MinLat: bounds.MinLat, MaxLat: bounds.MaxLat, MinLon: bounds.MinLon, MaxLon: bounds.MaxLon}, nil
}

func bearingRad(lat1, lon1, lat2, lon2 float64) float64 {
	dLon := lon2 - lon1
	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	return math.Atan2(y, x)
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}

func radToDeg(rad float64) float64 {
	return rad * 180.0 / math.Pi
}
