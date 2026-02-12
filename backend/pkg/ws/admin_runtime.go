package ws

// AdminRuntime 用于将管理操作映射到 Hub。
type AdminRuntime struct {
	hub *Hub
}

func NewAdminRuntime(hub *Hub) *AdminRuntime {
	return &AdminRuntime{hub: hub}
}

func (a *AdminRuntime) KickPlayer(playerID string) bool {
	if a == nil || a.hub == nil {
		return false
	}
	return a.hub.KickPlayer(playerID)
}
