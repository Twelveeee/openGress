package service

import (
	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/service/combat"
	"github.com/Twelveeee/openGress/backend/pkg/service/field"
	"github.com/Twelveeee/openGress/backend/pkg/service/hack"
	"github.com/Twelveeee/openGress/backend/pkg/service/link"
	"github.com/Twelveeee/openGress/backend/pkg/service/mapsvc"
	"github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/service/portal"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

type Container struct {
	Player *player.Service
	Map    *mapsvc.Service
	Portal *portal.Service
	Hack   *hack.Service
	Link   *link.Service
	Field  *field.Service
	Combat *combat.Service
}

func NewContainer(gameState *state.GameState, gameplay config.GameplayConfig) *Container {
	playerSvc := player.New(gameState, gameplay)
	mapSvc := mapsvc.New(gameState)
	portalSvc := portal.New(gameState, gameplay, playerSvc)
	hackSvc := hack.New(gameState, playerSvc, gameplay)
	linkSvc := link.New(gameState, gameplay, playerSvc)
	fieldSvc := field.New(gameState, gameplay, playerSvc)
	combatSvc := combat.New(gameState, gameplay, playerSvc)

	return &Container{
		Player: playerSvc,
		Map:    mapSvc,
		Portal: portalSvc,
		Hack:   hackSvc,
		Link:   linkSvc,
		Field:  fieldSvc,
		Combat: combatSvc,
	}
}
