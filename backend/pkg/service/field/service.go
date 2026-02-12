package field

import (
	"math"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/service/common"
	"github.com/Twelveeee/openGress/backend/pkg/service/link"
	"github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/service/portal"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/google/uuid"
)

type Service struct {
	gameState *state.GameState
	cfg       config.GameplayConfig
	players   *player.Service
}

func New(gameState *state.GameState, cfg config.GameplayConfig, players *player.Service) *Service {
	return &Service{gameState: gameState, cfg: cfg, players: players}
}

func (s *Service) CreateFromNewLink(linkState state.Link, playerID string, now time.Time) ([]state.Field, error) {
	if s.gameState == nil {
		return nil, nil
	}
	created := make([]state.Field, 0)
	links := s.gameState.Links.List()
	linkedFrom := link.LinkedPortals(linkState.FromPortalID, links)
	linkedTo := link.LinkedPortals(linkState.ToPortalID, links)

	for portalID := range linkedFrom {
		if portalID == linkState.ToPortalID {
			continue
		}
		if !linkedTo[portalID] {
			continue
		}
		p1, ok1 := s.gameState.Portals.Get(linkState.FromPortalID)
		p2, ok2 := s.gameState.Portals.Get(linkState.ToPortalID)
		p3, ok3 := s.gameState.Portals.Get(portalID)
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		if p1.Faction == "" || p1.Faction == "NEUTRAL" || p1.Faction != p2.Faction || p1.Faction != p3.Faction {
			continue
		}
		ids := portal.SortPortalIDs(p1.ID, p2.ID, p3.ID)
		if FieldExists(s.gameState.Fields.List(), ids) {
			continue
		}
		f := state.Field{ID: uuid.New().String(), PortalIDs: ids, Faction: p1.Faction, MU: ComputeFieldMU(p1.Position, p2.Position, p3.Position), Layer: 1, CreatedAt: now}
		s.gameState.Fields.Upsert(f)
		created = append(created, f)
		if p, ok := s.gameState.Players.Get(playerID); ok {
			p.FieldsCreated++
			p.MUTotal += f.MU
			common.ApplyAPGain(&p, gameplay.APRewardCreateField(s.cfg))
			p.UpdatedAt = now
			s.players.Upsert(p)
			s.appendLog(state.LogEntry{Type: "FIELD", PlayerID: p.ID, PortalID: linkState.FromPortalID, Faction: p.Faction, MU: f.MU, Message: "field created"})
		}
	}
	return created, nil
}

func (s *Service) RemoveByPortalIDs(ids map[string]struct{}) ([]state.Field, error) {
	if s.gameState == nil || len(ids) == 0 {
		return nil, nil
	}
	removed := make([]state.Field, 0)
	for _, fieldState := range s.gameState.Fields.List() {
		if _, ok := ids[fieldState.PortalIDs[0]]; !ok {
			if _, ok := ids[fieldState.PortalIDs[1]]; !ok {
				if _, ok := ids[fieldState.PortalIDs[2]]; !ok {
					continue
				}
			}
		}
		if s.gameState.Fields.Remove(fieldState.ID) {
			removed = append(removed, fieldState)
		}
	}
	return removed, nil
}

func FieldExists(fields []state.Field, ids [3]string) bool {
	for _, fieldState := range fields {
		if fieldState.PortalIDs == ids {
			return true
		}
	}
	return false
}

func ComputeFieldMU(p1, p2, p3 state.Position) int {
	refLat := (p1.Latitude + p2.Latitude + p3.Latitude) / 3.0
	x1 := p1.Longitude * player.MetersPerDegLon(refLat)
	y1 := p1.Latitude * player.MetersPerDegLat
	x2 := p2.Longitude * player.MetersPerDegLon(refLat)
	y2 := p2.Latitude * player.MetersPerDegLat
	x3 := p3.Longitude * player.MetersPerDegLon(refLat)
	y3 := p3.Latitude * player.MetersPerDegLat
	area := math.Abs((x1*(y2-y3) + x2*(y3-y1) + x3*(y1-y2)) / 2.0)
	return int(math.Floor(area))
}

func (s *Service) appendLog(entry state.LogEntry) {
	if s.gameState == nil || s.gameState.Logs == nil {
		return
	}
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	s.gameState.Logs.Add(entry)
}
