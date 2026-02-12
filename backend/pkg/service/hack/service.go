package hack

import (
	"math/rand"
	"strings"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/service/common"
	"github.com/Twelveeee/openGress/backend/pkg/service/errors"
	"github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/service/portal"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

const (
	PortalHackRangeM   = 40.0
	PortalHackCooldown = 300 * time.Second

	baseHackRolls               = 3
	modDropRateCommon           = 0.090
	modDropRateRare             = 0.024
	modDropRateVeryRare         = 0.006
	portalKeyDropRateNoKey      = 0.78
	portalKeyDropRateWithHasKey = 0.02
	hackBiasFriendly            = "friendly"
	hackBiasEnemy               = "enemy"
	hackBiasNeutral             = "neutral"
	modRarityCommon             = "COMMON"
	modRarityRare               = "RARE"
	modRarityVeryRare           = "VERY_RARE"
)

type Service struct {
	gameState *state.GameState
	players   *player.Service
	cfg       config.GameplayConfig
	randFloat func() float64
}

type Result struct {
	PortalID        string
	Success         bool
	ItemsGained     []string
	APGained        int
	CooldownSeconds int
}

func New(gameState *state.GameState, players *player.Service, cfg config.GameplayConfig) *Service {
	return &Service{
		gameState: gameState,
		players:   players,
		cfg:       cfg,
		randFloat: rand.Float64,
	}
}

func newWithRand(gameState *state.GameState, players *player.Service, cfg config.GameplayConfig, randFloat func() float64) *Service {
	svc := New(gameState, players, cfg)
	if randFloat != nil {
		svc.randFloat = randFloat
	}
	return svc
}

func (s *Service) ToggleAutoHack(playerID string, enabled bool, now time.Time) error {
	if playerID == "" {
		return errors.New(errors.KindBadRequest, "missing playerId")
	}
	p := s.players.GetOrCreate(playerID)
	common.SyncPlayerProgression(&p)
	p.AutoHack = enabled
	p.UpdatedAt = now
	s.players.Upsert(p)
	return nil
}

func (s *Service) HackPortal(playerID, portalID string, now time.Time) (Result, error) {
	if playerID == "" || portalID == "" {
		return Result{}, errors.New(errors.KindBadRequest, "missing playerId or portalId")
	}
	if s.gameState == nil {
		return Result{}, errors.New(errors.KindInternal, "state unavailable")
	}
	p := s.players.GetOrCreate(playerID)
	portalState, ok := s.gameState.Portals.Get(portalID)
	if !ok {
		return Result{}, errors.New(errors.KindNotFound, "portal not found")
	}
	if player.HaversineMeters(p.Position, portalState.Position) > PortalHackRangeM {
		return Result{}, errors.New(errors.KindBadRequest, "out of hack range")
	}
	if isInventoryOverCapacity(p) {
		return Result{}, errors.New(errors.KindBadRequest, "inventory capacity exceeded")
	}
	updated, result := s.executeHack(&p, portalState, now)
	s.players.Upsert(updated)
	return result, nil
}

func (s *Service) TryAutoHack(playerState state.Player, portals []state.Portal, now time.Time) (state.Player, *Result) {
	if !playerState.AutoHack {
		return playerState, nil
	}
	if isInventoryOverCapacity(playerState) {
		return playerState, nil
	}
	for _, p := range portals {
		if player.HaversineMeters(playerState.Position, p.Position) > PortalHackRangeM {
			continue
		}
		if cooldownRemaining(playerState, p.ID, now) > 0 {
			continue
		}
		next, result := s.executeHack(&playerState, p, now)
		if !result.Success {
			continue
		}
		return next, &result
	}
	return playerState, nil
}

func isInventoryOverCapacity(playerState state.Player) bool {
	copyState := playerState
	common.SyncPlayerProgression(&copyState)
	generalUsed, keyUsed := gameplay.InventoryCounts(copyState.Inventory)
	generalCap, keyCap := gameplay.CapacityForLevel(copyState.Level)
	return generalUsed > generalCap || keyUsed > keyCap
}

func cooldownRemaining(playerState state.Player, portalID string, now time.Time) int {
	if portalID == "" || playerState.HackCooldowns == nil {
		return 0
	}
	last, ok := playerState.HackCooldowns[portalID]
	if !ok {
		return 0
	}
	remaining := PortalHackCooldown - now.Sub(last)
	if remaining <= 0 {
		return 0
	}
	seconds := int(remaining.Seconds())
	if remaining%time.Second != 0 {
		seconds++
	}
	return seconds
}

func (s *Service) executeHack(playerState *state.Player, portalState state.Portal, now time.Time) (state.Player, Result) {
	if playerState == nil {
		return state.Player{}, Result{PortalID: portalState.ID, Success: false, CooldownSeconds: int(PortalHackCooldown.Seconds())}
	}
	playerCopy := *playerState
	if playerCopy.HackCooldowns == nil {
		playerCopy.HackCooldowns = make(map[string]time.Time)
	}
	if remaining := cooldownRemaining(playerCopy, portalState.ID, now); remaining > 0 {
		return playerCopy, Result{PortalID: portalState.ID, Success: false, CooldownSeconds: remaining}
	}

	playerCopy.HackCooldowns[portalState.ID] = now
	items := s.grantHackItems(&playerCopy, portalState)
	hackAPReward := gameplay.APRewardHackPortal(s.cfg)
	common.ApplyAPGain(&playerCopy, hackAPReward)
	playerCopy.UpdatedAt = now
	return playerCopy, Result{
		PortalID:        portalState.ID,
		Success:         true,
		ItemsGained:     items,
		APGained:        hackAPReward,
		CooldownSeconds: int(PortalHackCooldown.Seconds()),
	}
}

func (s *Service) grantHackItems(playerState *state.Player, portalState state.Portal) []string {
	if playerState == nil {
		return nil
	}
	if playerState.Inventory == nil {
		playerState.Inventory = make(map[string]int)
	}

	level := resolveHackLevel(*playerState, portalState)
	bias := hackDropBias(playerState.Faction, portalState.Faction)

	drops := make([]string, 0, baseHackRolls+2)
	for i := 0; i < baseHackRolls; i++ {
		itemType := s.weightedStringChoice(baseTypeWeightsByBias(bias))
		itemLevel := s.rollDropLevel(level)
		itemID := buildHackItemID(itemType, itemLevel)
		if itemID == "" {
			continue
		}
		playerState.Inventory[itemID]++
		drops = append(drops, itemID)
	}

	if modItemID := s.rollModItemID(); modItemID != "" {
		playerState.Inventory[modItemID]++
		drops = append(drops, modItemID)
	}

	keyID := gameplay.ItemIDKey(portalState.ID)
	if s.shouldDropKey(playerState.Inventory, portalState.ID) {
		playerState.Inventory[keyID]++
		drops = append(drops, keyID)
	}

	for itemID, count := range playerState.Inventory {
		if count <= 0 {
			delete(playerState.Inventory, itemID)
		}
	}
	return normalizeDrops(drops)
}

func resolveHackLevel(playerState state.Player, portalState state.Portal) int {
	level := portalState.Level
	if level <= 0 {
		level = portal.ComputePortalLevel(portalState)
	}
	level = gameplay.NormalizeLevel(level)
	if level > 8 {
		level = 8
	}
	if playerState.Level > 0 && level > playerState.Level {
		level = playerState.Level
	}
	if level <= 0 {
		level = 1
	}
	return level
}

func hackDropBias(playerFaction, portalFaction string) string {
	playerFaction = strings.ToUpper(strings.TrimSpace(playerFaction))
	portalFaction = strings.ToUpper(strings.TrimSpace(portalFaction))
	if playerFaction == "" || playerFaction == "NEUTRAL" || portalFaction == "" || portalFaction == "NEUTRAL" {
		return hackBiasNeutral
	}
	if playerFaction == portalFaction {
		return hackBiasFriendly
	}
	return hackBiasEnemy
}

func baseTypeWeightsByBias(bias string) []weightedStringOption {
	switch bias {
	case hackBiasFriendly:
		return []weightedStringOption{
			{Value: "RESO", Weight: 44},
			{Value: "XMP", Weight: 28},
			{Value: "US", Weight: 8},
			{Value: "CUBE", Weight: 20},
		}
	case hackBiasEnemy:
		return []weightedStringOption{
			{Value: "RESO", Weight: 20},
			{Value: "XMP", Weight: 48},
			{Value: "US", Weight: 12},
			{Value: "CUBE", Weight: 20},
		}
	default:
		return []weightedStringOption{
			{Value: "RESO", Weight: 34},
			{Value: "XMP", Weight: 36},
			{Value: "US", Weight: 10},
			{Value: "CUBE", Weight: 20},
		}
	}
}

func buildHackItemID(itemType string, level int) string {
	switch itemType {
	case "RESO":
		return gameplay.ItemIDResonator(level)
	case "XMP":
		return gameplay.ItemIDXMP(level)
	case "US":
		return gameplay.ItemIDUS(level)
	case "CUBE":
		return gameplay.ItemIDCube(level)
	default:
		return ""
	}
}

func (s *Service) rollDropLevel(baseLevel int) int {
	offsetWeights := []weightedIntOption{
		{Value: -2, Weight: 8},
		{Value: -1, Weight: 17},
		{Value: 0, Weight: 50},
		{Value: 1, Weight: 17},
		{Value: 2, Weight: 8},
	}
	candidates := make([]weightedIntOption, 0, len(offsetWeights))
	for _, option := range offsetWeights {
		level := baseLevel + option.Value
		if level < 1 || level > 8 {
			continue
		}
		candidates = append(candidates, weightedIntOption{
			Value:  level,
			Weight: option.Weight,
		})
	}
	if len(candidates) == 0 {
		return gameplay.NormalizeLevel(baseLevel)
	}
	return s.weightedIntChoice(candidates)
}

func (s *Service) rollModItemID() string {
	roll := s.nextRand()
	rarity := ""
	switch {
	case roll < modDropRateVeryRare:
		rarity = modRarityVeryRare
	case roll < modDropRateVeryRare+modDropRateRare:
		rarity = modRarityRare
	case roll < modDropRateVeryRare+modDropRateRare+modDropRateCommon:
		rarity = modRarityCommon
	default:
		return ""
	}
	modType := s.weightedStringChoice(modTypeWeightsByRarity(rarity))
	if modType == "" {
		return ""
	}
	return gameplay.ItemIDMod(modType, rarity)
}

func modTypeWeightsByRarity(rarity string) []weightedStringOption {
	switch rarity {
	case modRarityCommon:
		return []weightedStringOption{
			{Value: "SHIELD", Weight: 45},
			{Value: "HEAT_SINK", Weight: 15},
			{Value: "MULTI_HACK", Weight: 15},
			{Value: "FORCE_AMP", Weight: 13},
			{Value: "TURRET", Weight: 12},
		}
	case modRarityRare:
		return []weightedStringOption{
			{Value: "SHIELD", Weight: 40},
			{Value: "HEAT_SINK", Weight: 18},
			{Value: "MULTI_HACK", Weight: 18},
			{Value: "LINK_AMP", Weight: 12},
			{Value: "FORCE_AMP", Weight: 7},
			{Value: "TURRET", Weight: 5},
		}
	case modRarityVeryRare:
		return []weightedStringOption{
			{Value: "SHIELD", Weight: 40},
			{Value: "HEAT_SINK", Weight: 30},
			{Value: "MULTI_HACK", Weight: 30},
		}
	default:
		return nil
	}
}

func (s *Service) shouldDropKey(inventory map[string]int, portalID string) bool {
	if portalID == "" {
		return false
	}
	current := gameplay.KeyCount(inventory, portalID)
	if current >= gameplay.MaxKeyPerPortal {
		return false
	}
	probability := portalKeyDropRateWithHasKey
	if current == 0 {
		probability = portalKeyDropRateNoKey
	}
	return s.nextRand() < probability
}

func (s *Service) nextRand() float64 {
	if s == nil || s.randFloat == nil {
		return rand.Float64()
	}
	value := s.randFloat()
	switch {
	case value < 0:
		return 0
	case value >= 1:
		return 0.999999
	default:
		return value
	}
}

type weightedStringOption struct {
	Value  string
	Weight int
}

type weightedIntOption struct {
	Value  int
	Weight int
}

func (s *Service) weightedStringChoice(options []weightedStringOption) string {
	if len(options) == 0 {
		return ""
	}
	totalWeight := 0
	for _, option := range options {
		if option.Weight > 0 {
			totalWeight += option.Weight
		}
	}
	if totalWeight <= 0 {
		return ""
	}
	threshold := s.nextRand() * float64(totalWeight)
	acc := 0.0
	for _, option := range options {
		if option.Weight <= 0 {
			continue
		}
		acc += float64(option.Weight)
		if threshold < acc {
			return option.Value
		}
	}
	return options[len(options)-1].Value
}

func (s *Service) weightedIntChoice(options []weightedIntOption) int {
	if len(options) == 0 {
		return 1
	}
	totalWeight := 0
	for _, option := range options {
		if option.Weight > 0 {
			totalWeight += option.Weight
		}
	}
	if totalWeight <= 0 {
		return options[len(options)-1].Value
	}
	threshold := s.nextRand() * float64(totalWeight)
	acc := 0.0
	for _, option := range options {
		if option.Weight <= 0 {
			continue
		}
		acc += float64(option.Weight)
		if threshold < acc {
			return option.Value
		}
	}
	return options[len(options)-1].Value
}

func normalizeDrops(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		normalized = append(normalized, v)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
