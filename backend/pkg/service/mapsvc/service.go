package mapsvc

import (
	"github.com/Twelveeee/openGress/backend/pkg/geo"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

type Service struct {
	gameState *state.GameState
}

func New(gameState *state.GameState) *Service {
	return &Service{gameState: gameState}
}

func (s *Service) CollectPortals(bounds state.Bounds) []state.Portal {
	if s.gameState == nil {
		return nil
	}
	portals := s.gameState.Portals.List()
	result := make([]state.Portal, 0, len(portals))
	for _, portal := range portals {
		if geo.PointInBounds(portal.Position, bounds) {
			result = append(result, portal)
		}
	}
	return result
}

func (s *Service) CollectLinks(bounds state.Bounds) []state.Link {
	if s.gameState == nil {
		return nil
	}
	links := s.gameState.Links.List()
	result := make([]state.Link, 0, len(links))
	for _, link := range links {
		if s.LinkVisible(link.FromPosition, link.ToPosition, bounds) {
			result = append(result, link)
		}
	}
	return result
}

func (s *Service) CollectFields(bounds state.Bounds) []state.Field {
	if s.gameState == nil {
		return nil
	}
	fields := s.gameState.Fields.List()
	result := make([]state.Field, 0, len(fields))
	for _, field := range fields {
		p1, ok1 := s.gameState.Portals.Get(field.PortalIDs[0])
		p2, ok2 := s.gameState.Portals.Get(field.PortalIDs[1])
		p3, ok3 := s.gameState.Portals.Get(field.PortalIDs[2])
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		if s.FieldVisible(p1.Position, p2.Position, p3.Position, bounds) {
			result = append(result, field)
		}
	}
	return result
}

func (s *Service) LinkVisible(from, to state.Position, bounds state.Bounds) bool {
	return geo.LinkVisibleInBounds(from, to, bounds)
}

func (s *Service) FieldVisible(a, b, c state.Position, bounds state.Bounds) bool {
	if geo.PointInBounds(a, bounds) || geo.PointInBounds(b, bounds) || geo.PointInBounds(c, bounds) {
		return true
	}
	if geo.LinkVisibleInBounds(a, b, bounds) || geo.LinkVisibleInBounds(b, c, bounds) || geo.LinkVisibleInBounds(c, a, bounds) {
		return true
	}
	return false
}
