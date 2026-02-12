package state

type PlayerRepository interface {
	// 玩家读写仓库抽象。
	Get(id string) (Player, bool)
	Upsert(player Player)
	Remove(id string) bool
	List() []Player
}

type PortalRepository interface {
	// Portal 读写仓库抽象。
	Get(id string) (Portal, bool)
	Upsert(portal Portal)
	Remove(id string) bool
	List() []Portal
}

type LinkRepository interface {
	// Link 读写仓库抽象。
	Get(id string) (Link, bool)
	Upsert(link Link)
	Remove(id string) bool
	List() []Link
}

type FieldRepository interface {
	// Field 读写仓库抽象。
	Get(id string) (Field, bool)
	Upsert(field Field)
	Remove(id string) bool
	List() []Field
}

type GameState struct {
	// GameState 聚合所有核心状态仓库。
	Players PlayerRepository
	Portals PortalRepository
	Links   LinkRepository
	Fields  FieldRepository
	Logs    *LogStore
	Admin   *AdminState
}

func NewGameState() *GameState {
	return &GameState{
		Players: NewInMemoryPlayerRepo(),
		Portals: NewInMemoryPortalRepo(),
		Links:   NewInMemoryLinkRepo(),
		Fields:  NewInMemoryFieldRepo(),
		Logs:    NewLogStore(DefaultLogCapacity),
		Admin:   NewAdminState(),
	}
}
