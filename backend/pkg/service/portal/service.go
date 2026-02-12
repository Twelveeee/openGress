package portal

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/service/common"
	"github.com/Twelveeee/openGress/backend/pkg/service/errors"
	"github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/google/uuid"
)

var baseLinkRangeMeters = map[int]float64{
	1: 160.0,
	2: 810.0,
	3: 6250.0,
	4: 24000.0,
	5: 40900.0,
	6: 81400.0,
	7: 110300.0,
	8: 160100.0,
}

var rareLinkMultiplier = map[int]float64{1: 2.0, 2: 2.5, 3: 2.75, 4: 3.0}
var veryRareLinkMultiplier = map[int]float64{1: 7.0, 2: 8.75, 3: 9.625, 4: 10.5}
var softbankLinkMultiplier = map[int]float64{1: 5.0, 2: 6.25, 3: 6.875, 4: 7.5}

var resonatorMaxEnergy = map[int]int{1: 1000, 2: 1500, 3: 2000, 4: 2500, 5: 3000, 6: 4000, 7: 5000, 8: 6000}

type Service struct {
	gameState *state.GameState
	cfg       config.GameplayConfig
	players   *player.Service
}

type DeployResonatorCmd struct {
	PlayerID        string
	PortalID        string
	Slot            int
	Level           int
	ExpectedVersion int
}

type DeployResonatorResult struct {
	Portal        state.Portal
	Player        state.Player
	WasNeutral    bool
	PreviousLevel int
	NewVersion    int
	APGained      int
}

type DeployModCmd struct {
	PlayerID        string
	PortalID        string
	Slot            int
	ModType         string
	Rarity          string
	ExpectedVersion int
}

type DeployModResult struct {
	Portal     state.Portal
	Player     state.Player
	ModType    string
	Rarity     string
	NewVersion int
	APGained   int
	XMCost     int
}

type ChargeCmd struct {
	PlayerID string
	PortalID string
	Amount   int
}

type ChargeResult struct {
	Portal        state.Portal
	Player        state.Player
	AppliedAmount int
	XMCost        int
}

func New(gameState *state.GameState, cfg config.GameplayConfig, players *player.Service) *Service {
	return &Service{gameState: gameState, cfg: cfg, players: players}
}

func (s *Service) DeployResonator(cmd DeployResonatorCmd, now time.Time) (DeployResonatorResult, error) {
	if cmd.PlayerID == "" || cmd.PortalID == "" {
		return DeployResonatorResult{}, errors.New(errors.KindBadRequest, "missing playerId or portalId")
	}
	if cmd.Slot < 1 || cmd.Slot > 8 || cmd.Level < 1 || cmd.Level > 8 {
		return DeployResonatorResult{}, errors.New(errors.KindBadRequest, "invalid slot or level")
	}
	if s.gameState == nil {
		return DeployResonatorResult{}, errors.New(errors.KindInternal, "state unavailable")
	}
	p := s.players.GetOrCreate(cmd.PlayerID)
	common.SyncPlayerProgression(&p)
	if p.Level < cmd.Level {
		return DeployResonatorResult{}, errors.New(errors.KindBadRequest, "player level too low")
	}
	portal, ok := s.gameState.Portals.Get(cmd.PortalID)
	if !ok {
		return DeployResonatorResult{}, errors.New(errors.KindNotFound, "portal not found")
	}
	if !s.players.WithinInteractRange(p.Position, portal.Position) {
		return DeployResonatorResult{}, errors.New(errors.KindBadRequest, "out of interact range")
	}
	if portal.Faction != "" && portal.Faction != "NEUTRAL" && portal.Faction != p.Faction {
		return DeployResonatorResult{}, errors.New(errors.KindBadRequest, "portal belongs to other faction")
	}
	itemID := gameplay.ItemIDResonator(cmd.Level)
	if p.Inventory == nil || p.Inventory[itemID] <= 0 {
		return DeployResonatorResult{}, errors.New(errors.KindBadRequest, "insufficient resonator")
	}
	cost := gameplay.DeployResonatorCost(s.cfg, cmd.Level)
	if p.XM < cost {
		return DeployResonatorResult{}, errors.New(errors.KindBadRequest, "insufficient xm")
	}
	if portal.Resonators == nil {
		portal.Resonators = make(map[int]state.ResonatorSlot)
	}
	current := portal.Resonators[cmd.Slot]
	if current.Version != cmd.ExpectedVersion {
		return DeployResonatorResult{}, errors.New(errors.KindConflict, "resonator slot conflict")
	}
	wasNeutral := portal.Faction == "" || portal.Faction == "NEUTRAL"
	prevLevel := current.Level
	portal.Resonators[cmd.Slot] = state.ResonatorSlot{Slot: cmd.Slot, Level: cmd.Level, Energy: ResonatorEnergyForLevel(cmd.Level), PlayerID: cmd.PlayerID, Version: current.Version + 1, UpdatedAt: now}
	if portal.Faction == "" || portal.Faction == "NEUTRAL" {
		portal.Faction = p.Faction
	}
	portal.Level = ComputePortalLevel(portal)
	portal.Energy = ComputePortalEnergy(portal)
	portal.UpdatedAt = now
	s.gameState.Portals.Upsert(portal)

	apGained := 0
	p.Inventory[itemID]--
	p.XM -= cost
	if prevLevel == 0 {
		// 在空槽部署谐振器获得基础 AP。
		apGained += gameplay.APRewardDeployResonator(s.cfg)
	}
	if wasNeutral {
		// 中立点首占奖励 AP。
		p.PortalCaptures++
		apGained += gameplay.APRewardCapturePortal(s.cfg)
		s.appendLog(state.LogEntry{Type: "CAPTURE", PlayerID: p.ID, PortalID: portal.ID, Faction: p.Faction, Message: "portal captured"})
	} else if prevLevel > 0 && cmd.Level > prevLevel {
		// 升级已有谐振器奖励 AP。
		apGained += gameplay.APRewardUpgradeResonator(s.cfg)
	}
	if apGained > 0 {
		common.ApplyAPGain(&p, apGained)
	}
	p.UpdatedAt = now
	s.players.Upsert(p)
	return DeployResonatorResult{
		Portal:        portal,
		Player:        p,
		WasNeutral:    wasNeutral,
		PreviousLevel: prevLevel,
		NewVersion:    current.Version + 1,
		APGained:      apGained,
	}, nil
}

func (s *Service) DeployMod(cmd DeployModCmd, now time.Time) (DeployModResult, error) {
	if cmd.PlayerID == "" || cmd.PortalID == "" || cmd.ModType == "" {
		return DeployModResult{}, errors.New(errors.KindBadRequest, "missing playerId, portalId, or modType")
	}
	if cmd.Slot < 1 || cmd.Slot > 4 {
		return DeployModResult{}, errors.New(errors.KindBadRequest, "invalid slot")
	}
	if s.gameState == nil {
		return DeployModResult{}, errors.New(errors.KindInternal, "state unavailable")
	}
	portal, ok := s.gameState.Portals.Get(cmd.PortalID)
	if !ok {
		return DeployModResult{}, errors.New(errors.KindNotFound, "portal not found")
	}
	p := s.players.GetOrCreate(cmd.PlayerID)
	common.SyncPlayerProgression(&p)
	if portal.Faction != "" && portal.Faction != "NEUTRAL" && portal.Faction != p.Faction {
		return DeployModResult{}, errors.New(errors.KindBadRequest, "portal belongs to other faction")
	}
	if !s.players.WithinInteractRange(p.Position, portal.Position) {
		return DeployModResult{}, errors.New(errors.KindBadRequest, "out of interact range")
	}
	modType := normalizeModType(cmd.ModType)
	rarity := normalizeModRarity(cmd.Rarity)
	if modType == "" || rarity == "" {
		return DeployModResult{}, errors.New(errors.KindBadRequest, "invalid modType or rarity")
	}
	itemID := gameplay.ItemIDMod(modType, rarity)
	if p.Inventory == nil || p.Inventory[itemID] <= 0 {
		return DeployModResult{}, errors.New(errors.KindBadRequest, "insufficient mod")
	}
	cost := gameplay.DeployModCostWith(s.cfg, modType, rarity)
	if p.XM < cost {
		return DeployModResult{}, errors.New(errors.KindBadRequest, "insufficient xm")
	}
	if portal.Mods == nil {
		portal.Mods = make(map[int]state.ModSlot)
	}
	current := portal.Mods[cmd.Slot]
	if current.Version != cmd.ExpectedVersion {
		return DeployModResult{}, errors.New(errors.KindConflict, "mod slot conflict")
	}
	portal.Mods[cmd.Slot] = state.ModSlot{Slot: cmd.Slot, ModType: modType, Rarity: rarity, PlayerID: cmd.PlayerID, Version: current.Version + 1, UpdatedAt: now}
	portal.UpdatedAt = now
	s.gameState.Portals.Upsert(portal)

	p.Inventory[itemID]--
	p.XM -= cost
	apGained := gameplay.APRewardDeployMod(s.cfg)
	if apGained > 0 {
		common.ApplyAPGain(&p, apGained)
	}
	p.UpdatedAt = now
	s.players.Upsert(p)
	return DeployModResult{
		Portal:     portal,
		Player:     p,
		ModType:    modType,
		Rarity:     rarity,
		NewVersion: current.Version + 1,
		APGained:   apGained,
		XMCost:     cost,
	}, nil
}

func normalizeModType(raw string) string {
	modType := strings.ToUpper(strings.TrimSpace(raw))
	switch modType {
	case "PORTAL_SHIELD":
		return "SHIELD"
	case "HEATSINK":
		return "HEAT_SINK"
	case "MULTI":
		return "MULTI_HACK"
	case "LINK":
		return "LINK_AMP"
	case "FORCE":
		return "FORCE_AMP"
	case "SBUL":
		return "SOFTBANK_ULTRA_LINK"
	default:
		return modType
	}
}

func normalizeModRarity(raw string) string {
	rarity := strings.ToUpper(strings.TrimSpace(raw))
	switch rarity {
	case "C":
		return "COMMON"
	case "R":
		return "RARE"
	case "VR":
		return "VERY_RARE"
	default:
		return rarity
	}
}

func (s *Service) Charge(cmd ChargeCmd, now time.Time) (ChargeResult, error) {
	if cmd.PlayerID == "" || cmd.PortalID == "" {
		return ChargeResult{}, errors.New(errors.KindBadRequest, "missing playerId or portalId")
	}
	if cmd.Amount <= 0 {
		return ChargeResult{}, errors.New(errors.KindBadRequest, "invalid charge amount")
	}
	if s.gameState == nil {
		return ChargeResult{}, errors.New(errors.KindInternal, "state unavailable")
	}
	portal, ok := s.gameState.Portals.Get(cmd.PortalID)
	if !ok {
		return ChargeResult{}, errors.New(errors.KindNotFound, "portal not found")
	}
	p := s.players.GetOrCreate(cmd.PlayerID)
	common.SyncPlayerProgression(&p)
	if portal.Faction != "" && portal.Faction != "NEUTRAL" && portal.Faction != p.Faction {
		return ChargeResult{}, errors.New(errors.KindBadRequest, "portal belongs to other faction")
	}
	if !s.players.WithinInteractRange(p.Position, portal.Position) {
		return ChargeResult{}, errors.New(errors.KindBadRequest, "out of interact range")
	}
	remainingCapacity := PortalRemainingChargeCapacity(portal)
	if remainingCapacity <= 0 {
		return ChargeResult{Portal: portal, Player: p, AppliedAmount: 0, XMCost: 0}, nil
	}
	applied := cmd.Amount
	if applied > remainingCapacity {
		applied = remainingCapacity
	}
	cost := gameplay.ChargeCost(s.cfg, applied)
	if p.XM < cost {
		return ChargeResult{}, errors.New(errors.KindBadRequest, "insufficient xm")
	}

	actualApplied := ApplyPortalCharge(&portal, applied)
	if actualApplied <= 0 {
		return ChargeResult{Portal: portal, Player: p, AppliedAmount: 0, XMCost: 0}, nil
	}
	if actualApplied < applied {
		cost = gameplay.ChargeCost(s.cfg, actualApplied)
	}
	portal.UpdatedAt = now
	s.gameState.Portals.Upsert(portal)

	p.XM -= cost
	p.UpdatedAt = now
	s.players.Upsert(p)
	return ChargeResult{
		Portal:        portal,
		Player:        p,
		AppliedAmount: actualApplied,
		XMCost:        cost,
	}, nil
}

func ComputePortalLevel(portal state.Portal) int {
	if len(portal.Resonators) == 0 {
		return 0
	}
	sum := 0
	count := 0
	for _, slot := range portal.Resonators {
		if slot.Level <= 0 {
			continue
		}
		sum += slot.Level
		count++
	}
	if count == 0 {
		return 0
	}
	level := int(math.Round(float64(sum) / float64(count)))
	if level < 1 {
		return 1
	}
	if level > 8 {
		return 8
	}
	return level
}

func ComputePortalEnergy(portal state.Portal) int {
	total := 0
	for _, slot := range portal.Resonators {
		if slot.Energy > 0 {
			total += slot.Energy
		}
	}
	return total
}

func ComputeLinkRangeMeters(portal state.Portal) float64 {
	base := ComputeBaseLinkRangeMeters(portal)
	if base <= 0 {
		return 0
	}
	mult := 1.0
	if portal.Mods != nil {
		var rareCount, veryRareCount, softbankCount int
		for _, mod := range portal.Mods {
			switch mod.ModType {
			case "LINK_AMP":
				switch mod.Rarity {
				case "VERY_RARE":
					veryRareCount++
				case "RARE":
					rareCount++
				}
			case "SOFTBANK_ULTRA_LINK":
				softbankCount++
			}
		}
		if veryRareCount > 0 {
			if veryRareCount > 4 {
				veryRareCount = 4
			}
			mult = veryRareLinkMultiplier[veryRareCount]
		} else if softbankCount > 0 {
			if softbankCount > 4 {
				softbankCount = 4
			}
			mult = softbankLinkMultiplier[softbankCount]
		} else if rareCount > 0 {
			if rareCount > 4 {
				rareCount = 4
			}
			mult = rareLinkMultiplier[rareCount]
		}
	}
	return base * mult
}

func ComputeBaseLinkRangeMeters(portal state.Portal) float64 {
	counts := [9]int{}
	total := 0
	for _, slot := range portal.Resonators {
		if slot.Level < 1 || slot.Level > 8 {
			continue
		}
		counts[slot.Level]++
		total++
	}
	if total == 8 {
		switch {
		case counts[1] == 8:
			return 160.0
		case counts[2] == 4 && counts[1] == 4:
			return 810.0
		case counts[3] == 4 && counts[2] == 4:
			return 6250.0
		case counts[4] == 4 && counts[3] == 4:
			return 24000.0
		case counts[5] == 2 && counts[4] == 3 && counts[3] == 3:
			return 40900.0
		case counts[6] == 2 && counts[5] == 2 && counts[4] == 4:
			return 81400.0
		case counts[7] == 1 && counts[6] == 2 && counts[5] == 2 && counts[4] == 3:
			return 110300.0
		case counts[8] == 1 && counts[7] == 1 && counts[6] == 2 && counts[5] == 2 && counts[4] == 2:
			return 160100.0
		case counts[8] == 2 && counts[7] == 2 && counts[6] == 4:
			return 332100.0
		case counts[8] == 3 && counts[7] == 3 && counts[6] == 2:
			return 412300.0
		case counts[8] == 4 && counts[7] == 4:
			return 506200.0
		case counts[8] == 5 && counts[7] == 3:
			return 540800.0
		case counts[8] == 6 && counts[7] == 2:
			return 577200.0
		case counts[8] == 7 && counts[7] == 1:
			return 615300.0
		case counts[8] == 8:
			return 655300.0
		}
	}
	level := portal.Level
	if level == 0 {
		level = ComputePortalLevel(portal)
	}
	base, ok := baseLinkRangeMeters[level]
	if !ok {
		return 0
	}
	return base
}

func ResonatorEnergyForLevel(level int) int {
	if energy, ok := resonatorMaxEnergy[level]; ok {
		return energy
	}
	return 0
}

func ApplyPortalCharge(portal *state.Portal, amount int) int {
	if amount <= 0 {
		return 0
	}
	if len(portal.Resonators) == 0 {
		return 0
	}
	remaining := amount
	for remaining > 0 {
		changed := false
		for slot, resonator := range portal.Resonators {
			maxEnergy := ResonatorEnergyForLevel(resonator.Level)
			if resonator.Energy >= maxEnergy {
				continue
			}
			resonator.Energy++
			portal.Resonators[slot] = resonator
			remaining--
			changed = true
			if remaining == 0 {
				break
			}
		}
		if !changed {
			break
		}
	}
	portal.Energy = ComputePortalEnergy(*portal)
	return amount - remaining
}

func PortalRemainingChargeCapacity(portal state.Portal) int {
	if len(portal.Resonators) == 0 {
		return 0
	}
	remaining := 0
	for _, resonator := range portal.Resonators {
		maxEnergy := ResonatorEnergyForLevel(resonator.Level)
		if maxEnergy <= 0 {
			continue
		}
		if resonator.Energy >= maxEnergy {
			continue
		}
		remaining += maxEnergy - resonator.Energy
	}
	return remaining
}

func SortPortalIDs(a, b, c string) [3]string {
	ids := []string{a, b, c}
	sort.Strings(ids)
	return [3]string{ids[0], ids[1], ids[2]}
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
