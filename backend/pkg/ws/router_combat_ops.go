package ws

import (
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	combatsvc "github.com/Twelveeee/openGress/backend/pkg/service/combat"
)

func handlePlayerAttack(hub *Hub, client *Client, msg Message) {
	var payload AttackPortalPayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid attack payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId"}))
		return
	}
	if hub.services == nil || hub.services.Combat == nil || hub.services.Link == nil || hub.services.Field == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	result, err := hub.services.Combat.Attack(combatsvc.AttackCmd{
		PlayerID:    playerID,
		PortalID:    payload.PortalID,
		WeaponType:  payload.WeaponType,
		WeaponLevel: payload.WeaponLevel,
		ChargeBonus: payload.ChargeBonus,
	}, time.Now())
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}

	removedLinks, err := hub.services.Link.RemoveByPortalIDs(result.NeutralPortalIDs)
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
	removedFields, err := hub.services.Field.RemoveByPortalIDs(result.NeutralPortalIDs)
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}

	client.Send(NewMessage(MessageAttackResult, msg.ID, AttackResultPayload{
		PortalID:               result.Payload.PortalID,
		WeaponType:             result.Payload.WeaponType,
		WeaponLevel:            result.Payload.WeaponLevel,
		ChargeBonus:            result.Payload.ChargeBonus,
		DamageDealt:            result.Payload.DamageDealt,
		ResonatorsDestroyed:    result.Payload.ResonatorsDestroyed,
		PortalEnergy:           result.Payload.PortalEnergy,
		PortalNeutral:          result.Payload.PortalNeutral,
		MitigationApplied:      result.Payload.MitigationApplied,
		CounterattackTriggered: result.Payload.Counterattack,
		CounterattackDamage:    result.Payload.CounterattackDamage,
		Counterattacks:         toWSCounterattacks(result.Payload.Counterattacks),
		ItemsDropped:           result.Payload.ItemsDropped,
		ModsDestroyed:          toWSDestroyedMods(result.Payload.ModsDestroyed),
		PortalDamages:          toWSPortalDamages(result.Payload.PortalDamages),
	}))
	client.Send(NewMessage(MessagePlayerResourceUpdate, msg.ID, PlayerResourceUpdatePayload{
		PlayerID:       result.PlayerDelta.PlayerID,
		APGained:       result.PlayerDelta.APGained,
		XMDelta:        result.PlayerDelta.XMDelta,
		InventoryDelta: result.PlayerDelta.InventoryDelta,
	}))
	broadcastPortalChanges(hub, result.ChangedPortals)
	broadcastLinkRemovals(hub, removedLinks)
	broadcastFieldRemovals(hub, removedFields)
	if len(removedLinks) > 0 || len(removedFields) > 0 {
		broadcastTopologySnapshot(hub)
	}
}

func toWSDestroyedMods(mods []combatsvc.DestroyedMod) []AttackDestroyedMod {
	if len(mods) == 0 {
		return nil
	}
	result := make([]AttackDestroyedMod, 0, len(mods))
	for _, mod := range mods {
		result = append(result, AttackDestroyedMod{
			PortalID: mod.PortalID,
			Slot:     mod.Slot,
			ModType:  mod.ModType,
			Rarity:   mod.Rarity,
		})
	}
	return result
}

func toWSPortalDamages(items []combatsvc.PortalDamage) []AttackPortalDamage {
	if len(items) == 0 {
		return nil
	}
	result := make([]AttackPortalDamage, 0, len(items))
	for _, item := range items {
		result = append(result, AttackPortalDamage{
			PortalID:            item.PortalID,
			Damage:              item.Damage,
			ResonatorsDestroyed: item.ResonatorsDestroyed,
			PortalNeutral:       item.PortalNeutral,
		})
	}
	return result
}

func toWSCounterattacks(items []combatsvc.CounterattackDetail) []AttackCounterattack {
	if len(items) == 0 {
		return nil
	}
	result := make([]AttackCounterattack, 0, len(items))
	for _, item := range items {
		result = append(result, AttackCounterattack{
			PortalID:  item.PortalID,
			Triggered: item.Triggered,
			Damage:    item.Damage,
			Critical:  item.Critical,
		})
	}
	return result
}
