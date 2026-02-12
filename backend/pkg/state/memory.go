package state

import (
	"sync"
	"time"
)

type InMemoryPlayerRepo struct {
	// 线程安全的内存玩家仓库。
	mu      sync.RWMutex
	players map[string]Player
}

func NewInMemoryPlayerRepo() *InMemoryPlayerRepo {
	return &InMemoryPlayerRepo{
		players: make(map[string]Player),
	}
}

func (r *InMemoryPlayerRepo) Get(id string) (Player, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	player, ok := r.players[id]
	if !ok {
		return Player{}, false
	}
	return clonePlayer(player), true
}

func (r *InMemoryPlayerRepo) Upsert(player Player) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.players[player.ID] = clonePlayer(player)
}

func (r *InMemoryPlayerRepo) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.players[id]; !ok {
		return false
	}
	delete(r.players, id)
	return true
}

func (r *InMemoryPlayerRepo) List() []Player {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Player, 0, len(r.players))
	for _, player := range r.players {
		result = append(result, clonePlayer(player))
	}
	return result
}

type InMemoryPortalRepo struct {
	// 线程安全的内存 Portal 仓库。
	mu      sync.RWMutex
	portals map[string]Portal
}

func NewInMemoryPortalRepo() *InMemoryPortalRepo {
	return &InMemoryPortalRepo{
		portals: make(map[string]Portal),
	}
}

func (r *InMemoryPortalRepo) Get(id string) (Portal, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	portal, ok := r.portals[id]
	if !ok {
		return Portal{}, false
	}
	return clonePortal(portal), true
}

func (r *InMemoryPortalRepo) Upsert(portal Portal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if portal.Title == "" {
		portal.Title = portal.ID
	}
	r.portals[portal.ID] = clonePortal(portal)
}

func (r *InMemoryPortalRepo) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.portals[id]; !ok {
		return false
	}
	delete(r.portals, id)
	return true
}

func (r *InMemoryPortalRepo) List() []Portal {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Portal, 0, len(r.portals))
	for _, portal := range r.portals {
		result = append(result, clonePortal(portal))
	}
	return result
}

type InMemoryLinkRepo struct {
	// 线程安全的内存 Link 仓库。
	mu    sync.RWMutex
	links map[string]Link
}

func NewInMemoryLinkRepo() *InMemoryLinkRepo {
	return &InMemoryLinkRepo{
		links: make(map[string]Link),
	}
}

func (r *InMemoryLinkRepo) Get(id string) (Link, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	link, ok := r.links[id]
	return link, ok
}

func (r *InMemoryLinkRepo) Upsert(link Link) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.links[link.ID] = link
}

func (r *InMemoryLinkRepo) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.links[id]; !ok {
		return false
	}
	delete(r.links, id)
	return true
}

func (r *InMemoryLinkRepo) List() []Link {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Link, 0, len(r.links))
	for _, link := range r.links {
		result = append(result, link)
	}
	return result
}

type InMemoryFieldRepo struct {
	// 线程安全的内存 Field 仓库。
	mu     sync.RWMutex
	fields map[string]Field
}

func NewInMemoryFieldRepo() *InMemoryFieldRepo {
	return &InMemoryFieldRepo{
		fields: make(map[string]Field),
	}
}

func (r *InMemoryFieldRepo) Get(id string) (Field, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	field, ok := r.fields[id]
	return field, ok
}

func (r *InMemoryFieldRepo) Upsert(field Field) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fields[field.ID] = field
}

func (r *InMemoryFieldRepo) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.fields[id]; !ok {
		return false
	}
	delete(r.fields, id)
	return true
}

func (r *InMemoryFieldRepo) List() []Field {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Field, 0, len(r.fields))
	for _, field := range r.fields {
		result = append(result, field)
	}
	return result
}

// clonePlayer 复制 Player，避免 map/slice/pointer 被仓库外共享修改。
func clonePlayer(player Player) Player {
	cloned := player
	if player.Inventory != nil {
		cloned.Inventory = make(map[string]int, len(player.Inventory))
		for itemID, amount := range player.Inventory {
			cloned.Inventory[itemID] = amount
		}
	}
	if player.HackCooldowns != nil {
		cloned.HackCooldowns = make(map[string]time.Time, len(player.HackCooldowns))
		for portalID, t := range player.HackCooldowns {
			cloned.HackCooldowns[portalID] = t
		}
	}
	if player.ViewTiles != nil {
		cloned.ViewTiles = append([]string(nil), player.ViewTiles...)
	}
	if player.Target != nil {
		target := *player.Target
		cloned.Target = &target
	}
	if player.View != nil {
		view := *player.View
		cloned.View = &view
	}
	return cloned
}

// clonePortal 复制 Portal，避免共用 Resonators/Mods map。
func clonePortal(portal Portal) Portal {
	cloned := portal
	if portal.Resonators != nil {
		cloned.Resonators = make(map[int]ResonatorSlot, len(portal.Resonators))
		for slot, resonator := range portal.Resonators {
			cloned.Resonators[slot] = resonator
		}
	}
	if portal.Mods != nil {
		cloned.Mods = make(map[int]ModSlot, len(portal.Mods))
		for slot, mod := range portal.Mods {
			cloned.Mods[slot] = mod
		}
	}
	return cloned
}
