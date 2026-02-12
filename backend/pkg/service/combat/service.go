package combat

import (
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/service/common"
	"github.com/Twelveeee/openGress/backend/pkg/service/errors"
	"github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/service/portal"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

const ResonatorRingRadiusM = 10.0

var slotBearingsDeg = []float64{0, 45, 90, 135, 180, 225, 270, 315}

type Service struct {
	gameState *state.GameState
	cfg       config.GameplayConfig
	attackCfg runtimeAttackConfig
	players   *player.Service
	rng       *rand.Rand
	rngMu     sync.Mutex
}

type AttackCmd struct {
	PlayerID    string
	PortalID    string
	WeaponType  string
	WeaponLevel int
	ChargeBonus float64
}

type Result struct {
	Payload          AttackResultPayload
	ChangedPortals   []state.Portal
	NeutralPortalIDs map[string]struct{}
	Player           state.Player
	PlayerDelta      PlayerResourceDelta
}

type AttackResultPayload struct {
	PortalID            string
	WeaponType          string
	WeaponLevel         int
	ChargeBonus         float64
	DamageDealt         int
	ResonatorsDestroyed int
	PortalEnergy        int
	PortalNeutral       bool
	MitigationApplied   float64
	ModsDestroyed       []DestroyedMod
	Counterattack       bool
	CounterattackDamage int
	Counterattacks      []CounterattackDetail
	PortalDamages       []PortalDamage
	ItemsDropped        []string
}

type PortalDamage struct {
	PortalID            string
	Damage              int
	ResonatorsDestroyed int
	PortalNeutral       bool
}

type CounterattackDetail struct {
	PortalID  string
	Triggered bool
	Damage    int
	Critical  bool
}

type PlayerResourceDelta struct {
	PlayerID       string
	APGained       int
	XMDelta        int
	InventoryDelta map[string]int
}

func New(gameState *state.GameState, cfg config.GameplayConfig, players *player.Service) *Service {
	return &Service{
		gameState: gameState,
		cfg:       cfg,
		attackCfg: buildRuntimeAttackConfig(cfg.Attack),
		players:   players,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Service) setRandSource(source rand.Source) {
	if source == nil {
		return
	}
	s.rngMu.Lock()
	defer s.rngMu.Unlock()
	s.rng = rand.New(source)
}

func (s *Service) Attack(cmd AttackCmd, now time.Time) (Result, error) {
	if cmd.PlayerID == "" {
		return Result{}, errors.New(errors.KindBadRequest, "missing playerId")
	}
	if cmd.WeaponLevel < 1 || cmd.WeaponLevel > 8 {
		return Result{}, errors.New(errors.KindBadRequest, "invalid weapon level")
	}
	if s.gameState == nil {
		return Result{}, errors.New(errors.KindInternal, "state unavailable")
	}
	weaponType := strings.ToUpper(cmd.WeaponType)
	if weaponType != "XMP" && weaponType != "US" {
		return Result{}, errors.New(errors.KindBadRequest, "invalid weapon type")
	}
	p := s.players.GetOrCreate(cmd.PlayerID)
	common.SyncPlayerProgression(&p)
	if p.Level < cmd.WeaponLevel {
		return Result{}, errors.New(errors.KindBadRequest, "player level too low")
	}
	var (
		target    state.Portal
		hasTarget bool
	)
	if cmd.PortalID != "" {
		if value, ok := s.gameState.Portals.Get(cmd.PortalID); ok {
			if value.Faction != "" && value.Faction != "NEUTRAL" && value.Faction != p.Faction {
				target = value
				hasTarget = true
			}
		}
	}
	spec, ok := s.attackCfg.weaponSpecs[weaponType][cmd.WeaponLevel]
	if !ok {
		return Result{}, errors.New(errors.KindBadRequest, "invalid weapon spec")
	}
	chargeBonus := cmd.ChargeBonus
	if chargeBonus < 0 {
		chargeBonus = 0
	}
	if chargeBonus > 0.2 {
		chargeBonus = 0.2
	}
	itemID := gameplay.ItemIDWeapon(weaponType, cmd.WeaponLevel)
	if p.Inventory == nil || p.Inventory[itemID] <= 0 {
		return Result{}, errors.New(errors.KindBadRequest, "insufficient weapon")
	}
	cost := spec.CostXM
	if cost <= 0 {
		cost = gameplay.AttackCost(s.cfg, weaponType, cmd.WeaponLevel)
	}
	if p.XM < cost {
		return Result{}, errors.New(errors.KindBadRequest, "insufficient xm")
	}
	if p.Inventory == nil {
		p.Inventory = make(map[string]int)
	}

	preAP := p.AP
	preXM := p.XM
	p.Inventory[itemID]--
	p.XM -= cost

	linkCounts := buildLinkCounts(s.gameState.Links.List())
	fieldCounts := buildFieldCounts(s.gameState.Fields.List())
	targetMitigation := 0.0
	crit := s.attackCfg.critByWeapon[weaponType]

	changedPortals := make([]state.Portal, 0)
	modsDestroyed := make([]DestroyedMod, 0)
	portalDamages := make([]PortalDamage, 0)
	counterattacks := make([]CounterattackDetail, 0)
	totalDamage := 0
	totalDestroyed := 0
	totalCounterDamage := 0
	counterattackTriggered := false
	targetEnergy := 0
	targetNeutral := false
	if hasTarget {
		targetMitigation = calculateMitigationPercent(target, linkCounts[target.ID], fieldCounts[target.ID], s.attackCfg.mitigation)
		targetEnergy = target.Energy
		targetNeutral = target.Level == 0 || target.Faction == "NEUTRAL"
	}

	for _, portalState := range s.gameState.Portals.List() {
		if portalState.Faction == "" || portalState.Faction == "NEUTRAL" || portalState.Faction == p.Faction {
			continue
		}
		if len(portalState.Resonators) == 0 {
			continue
		}
		counterSource := state.Portal{
			Faction:    portalState.Faction,
			Level:      portalState.Level,
			Resonators: cloneResonatorSlots(portalState.Resonators),
			Mods:       cloneModSlots(portalState.Mods),
		}

		mitigation := calculateMitigationPercent(portalState, linkCounts[portalState.ID], fieldCounts[portalState.ID], s.attackCfg.mitigation)
		portalChanged := false
		portalHit := false
		portalDamage := 0
		portalDestroyed := 0
		portalCrit := false

		for slot, resonator := range portalState.Resonators {
			pos := resonatorPosition(portalState.Position, slot)
			distance := player.HaversineMeters(p.Position, pos)
			rawDamage := computeRawDamage(spec.MaxDamage, distance, spec.RadiusM, chargeBonus, s.attackCfg.falloffTiers)
			if rawDamage <= 0 {
				continue
			}
			portalHit = true
			if crit.chance > 0 && s.rollChance(crit.chance) {
				rawDamage = rawDamage * crit.multiplier
				portalCrit = true
			}
			damage := applyMitigation(rawDamage, mitigation)
			if damage <= 0 {
				continue
			}

			damageDealt := damage
			if damageDealt > resonator.Energy {
				damageDealt = resonator.Energy
			}
			resonator.Energy -= damage
			totalDamage += damageDealt
			portalDamage += damageDealt
			portalChanged = true

			if resonator.Energy <= 0 {
				delete(portalState.Resonators, slot)
				portalDestroyed++
				totalDestroyed++
				continue
			}
			portalState.Resonators[slot] = resonator
		}

		if portalCrit {
			removedMods := destroyModsOnCrit(&portalState, s.attackCfg.modDestroy, s.rollIntn)
			if len(removedMods) > 0 {
				portalChanged = true
				modsDestroyed = append(modsDestroyed, removedMods...)
			}
		}

		if !portalChanged {
			continue
		}
		portalState.Energy = portal.ComputePortalEnergy(portalState)
		portalState.Level = portal.ComputePortalLevel(portalState)
		if portalState.Level == 0 {
			portalState.Faction = "NEUTRAL"
		}
		portalState.UpdatedAt = now
		changedPortals = append(changedPortals, portalState)
		portalDamages = append(portalDamages, PortalDamage{
			PortalID:            portalState.ID,
			Damage:              portalDamage,
			ResonatorsDestroyed: portalDestroyed,
			PortalNeutral:       portalState.Level == 0 || portalState.Faction == "NEUTRAL",
		})
		if portalHit {
			counter := calculateCounterattack(counterSource, s.attackCfg.counterattack, s.rollChance)
			detail := CounterattackDetail{
				PortalID:  portalState.ID,
				Triggered: counter.triggered,
				Damage:    counter.damage,
				Critical:  counter.critical,
			}
			if counter.triggered {
				counterattackTriggered = true
			}
			if counter.damage > 0 {
				damage := counter.damage
				if damage > p.XM {
					damage = p.XM
				}
				if damage > 0 {
					p.XM -= damage
					totalCounterDamage += damage
				}
				detail.Damage = damage
			}
			counterattacks = append(counterattacks, detail)
		}
		if portalState.ID == cmd.PortalID {
			targetEnergy = portalState.Energy
			targetNeutral = portalState.Level == 0 || portalState.Faction == "NEUTRAL"
		}
	}

	apGained := gameplay.APRewardDestroyResonator(s.cfg, totalDestroyed)
	if apGained > 0 {
		common.ApplyAPGain(&p, apGained)
	}

	inventoryDelta := map[string]int{itemID: -1}
	itemsDropped := make([]string, 0)
	if cmd.PortalID != "" && targetNeutral {
		itemsDropped = s.maybeDropKey(&p, cmd.PortalID)
		for _, item := range itemsDropped {
			inventoryDelta[item]++
		}
	}
	p.UpdatedAt = now
	s.players.Upsert(p)

	for _, portalState := range changedPortals {
		s.gameState.Portals.Upsert(portalState)
	}

	payload := AttackResultPayload{
		PortalID:            cmd.PortalID,
		WeaponType:          weaponType,
		WeaponLevel:         cmd.WeaponLevel,
		ChargeBonus:         chargeBonus,
		DamageDealt:         totalDamage,
		ResonatorsDestroyed: totalDestroyed,
		PortalEnergy:        targetEnergy,
		PortalNeutral:       targetNeutral,
		MitigationApplied:   targetMitigation / 100.0,
		ModsDestroyed:       modsDestroyed,
		Counterattack:       counterattackTriggered,
		CounterattackDamage: totalCounterDamage,
		Counterattacks:      counterattacks,
		PortalDamages:       portalDamages,
		ItemsDropped:        itemsDropped,
	}

	neutralIDs := collectNeutralPortalIDs(changedPortals)
	return Result{
		Payload:          payload,
		ChangedPortals:   changedPortals,
		NeutralPortalIDs: neutralIDs,
		Player:           p,
		PlayerDelta: PlayerResourceDelta{
			PlayerID:       p.ID,
			APGained:       p.AP - preAP,
			XMDelta:        p.XM - preXM,
			InventoryDelta: inventoryDelta,
		},
	}, nil
}

func (s *Service) maybeDropKey(playerState *state.Player, portalID string) []string {
	if playerState == nil {
		return nil
	}
	if s.attackCfg.keyDropChance <= 0 || !s.rollChance(s.attackCfg.keyDropChance) {
		return nil
	}
	if playerState.Inventory == nil {
		playerState.Inventory = make(map[string]int)
	}
	if !gameplay.CanAddKey(playerState.Inventory, portalID, 1, playerState.Level) {
		return nil
	}
	key := gameplay.ItemIDKey(portalID)
	playerState.Inventory[key]++
	return []string{key}
}

func (s *Service) rollChance(chance float64) bool {
	if chance <= 0 {
		return false
	}
	if chance >= 1 {
		return true
	}
	s.rngMu.Lock()
	defer s.rngMu.Unlock()
	return s.rng.Float64() < chance
}

func (s *Service) rollIntn(limit int) int {
	if limit <= 1 {
		return 0
	}
	s.rngMu.Lock()
	defer s.rngMu.Unlock()
	return s.rng.Intn(limit)
}

func cloneResonatorSlots(source map[int]state.ResonatorSlot) map[int]state.ResonatorSlot {
	if len(source) == 0 {
		return nil
	}
	next := make(map[int]state.ResonatorSlot, len(source))
	for slot, value := range source {
		next[slot] = value
	}
	return next
}

func cloneModSlots(source map[int]state.ModSlot) map[int]state.ModSlot {
	if len(source) == 0 {
		return nil
	}
	next := make(map[int]state.ModSlot, len(source))
	for slot, value := range source {
		next[slot] = value
	}
	return next
}

func collectNeutralPortalIDs(portals []state.Portal) map[string]struct{} {
	ids := make(map[string]struct{})
	for _, portalState := range portals {
		if portalState.Level > 0 && portalState.Faction != "" && portalState.Faction != "NEUTRAL" {
			continue
		}
		ids[portalState.ID] = struct{}{}
	}
	return ids
}

func resonatorPosition(portalPos state.Position, slot int) state.Position {
	if slot < 1 || slot > 8 {
		return portalPos
	}
	bearing := slotBearingsDeg[slot-1]
	return player.MoveByBearing(portalPos, bearing, ResonatorRingRadiusM)
}
