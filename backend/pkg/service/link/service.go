package link

import (
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/geo"
	"github.com/Twelveeee/openGress/backend/pkg/service/common"
	"github.com/Twelveeee/openGress/backend/pkg/service/errors"
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

type CreateLinkCmd struct {
	PlayerID     string
	FromPortalID string
	ToPortalID   string
}

type CreateLinkResult struct {
	Link   state.Link
	Player state.Player
}

func New(gameState *state.GameState, cfg config.GameplayConfig, players *player.Service) *Service {
	return &Service{gameState: gameState, cfg: cfg, players: players}
}

func (s *Service) CreateLink(cmd CreateLinkCmd, now time.Time) (CreateLinkResult, error) {
	if cmd.PlayerID == "" || cmd.FromPortalID == "" || cmd.ToPortalID == "" {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "missing playerId or portal ids")
	}
	if cmd.FromPortalID == cmd.ToPortalID {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "portal ids must differ")
	}
	if s.gameState == nil {
		return CreateLinkResult{}, errors.New(errors.KindInternal, "state unavailable")
	}
	fromPortal, ok := s.gameState.Portals.Get(cmd.FromPortalID)
	if !ok {
		return CreateLinkResult{}, errors.New(errors.KindNotFound, "from portal not found")
	}
	toPortal, ok := s.gameState.Portals.Get(cmd.ToPortalID)
	if !ok {
		return CreateLinkResult{}, errors.New(errors.KindNotFound, "to portal not found")
	}
	if fromPortal.Faction == "" || fromPortal.Faction == "NEUTRAL" || toPortal.Faction == "" || toPortal.Faction == "NEUTRAL" {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "portal must be owned")
	}
	if fromPortal.Faction != toPortal.Faction {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "portal factions differ")
	}
	p := s.players.GetOrCreate(cmd.PlayerID)
	common.SyncPlayerProgression(&p)
	if p.Faction != "" && p.Faction != "NEUTRAL" && p.Faction != fromPortal.Faction {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "player faction mismatch")
	}
	if !s.players.WithinInteractRange(p.Position, fromPortal.Position) {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "out of interact range")
	}
	keyFrom := gameplay.ItemIDKey(cmd.FromPortalID)
	keyTo := gameplay.ItemIDKey(cmd.ToPortalID)
	if p.Inventory == nil || p.Inventory[keyFrom] <= 0 || p.Inventory[keyTo] <= 0 {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "missing portal keys")
	}
	cost := gameplay.LinkCost(s.cfg)
	if p.XM < cost {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "insufficient xm")
	}

	existingLinks := s.gameState.Links.List()
	outCount := 0
	for _, link := range existingLinks {
		if link.FromPortalID == cmd.FromPortalID {
			outCount++
		}
		if (link.FromPortalID == cmd.FromPortalID && link.ToPortalID == cmd.ToPortalID) ||
			(link.FromPortalID == cmd.ToPortalID && link.ToPortalID == cmd.FromPortalID) {
			return CreateLinkResult{}, errors.New(errors.KindConflict, "link already exists")
		}
	}
	if outCount >= 8 {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "max outgoing links")
	}

	linkDistance := player.HaversineMeters(fromPortal.Position, toPortal.Position)
	maxDistance := portal.ComputeLinkRangeMeters(fromPortal)
	if maxDistance <= 0 || linkDistance > maxDistance {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "link distance exceeded")
	}
	if LinkCrossesExisting(fromPortal.Position, toPortal.Position, cmd.FromPortalID, cmd.ToPortalID, existingLinks) {
		return CreateLinkResult{}, errors.New(errors.KindBadRequest, "link would cross")
	}

	linkState := state.Link{ID: uuid.New().String(), FromPortalID: fromPortal.ID, ToPortalID: toPortal.ID, FromPosition: fromPortal.Position, ToPosition: toPortal.Position, CreatedAt: now}
	s.gameState.Links.Upsert(linkState)

	p.Inventory[keyFrom]--
	p.Inventory[keyTo]--
	p.XM -= cost
	p.LinksCreated++
	common.ApplyAPGain(&p, gameplay.APRewardCreateLink(s.cfg))
	p.UpdatedAt = now
	s.players.Upsert(p)
	s.appendLog(state.LogEntry{Type: "LINK", PlayerID: p.ID, PortalID: fromPortal.ID, Faction: p.Faction, Message: "link created"})
	return CreateLinkResult{Link: linkState, Player: p}, nil
}

func (s *Service) RemoveByPortalIDs(ids map[string]struct{}) ([]state.Link, error) {
	if s.gameState == nil || len(ids) == 0 {
		return nil, nil
	}
	removed := make([]state.Link, 0)
	for _, linkState := range s.gameState.Links.List() {
		if _, ok := ids[linkState.FromPortalID]; !ok {
			if _, ok := ids[linkState.ToPortalID]; !ok {
				continue
			}
		}
		if s.gameState.Links.Remove(linkState.ID) {
			removed = append(removed, linkState)
		}
	}
	return removed, nil
}

func LinkCrossesExisting(from, to state.Position, fromID, toID string, links []state.Link) bool {
	for _, linkState := range links {
		if linkState.FromPortalID == fromID || linkState.ToPortalID == fromID || linkState.FromPortalID == toID || linkState.ToPortalID == toID {
			continue
		}
		if geo.SegmentsIntersect(
			geo.XY{X: from.Longitude, Y: from.Latitude},
			geo.XY{X: to.Longitude, Y: to.Latitude},
			geo.XY{X: linkState.FromPosition.Longitude, Y: linkState.FromPosition.Latitude},
			geo.XY{X: linkState.ToPosition.Longitude, Y: linkState.ToPosition.Latitude},
		) {
			return true
		}
	}
	return false
}

func LinkedPortals(portalID string, links []state.Link) map[string]bool {
	result := make(map[string]bool)
	for _, linkState := range links {
		if linkState.FromPortalID == portalID {
			result[linkState.ToPortalID] = true
		} else if linkState.ToPortalID == portalID {
			result[linkState.FromPortalID] = true
		}
	}
	return result
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
