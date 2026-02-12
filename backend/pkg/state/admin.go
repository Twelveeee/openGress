package state

// AdminState 保存管理侧的额外状态（不直接暴露给玩家）。
type AdminState struct {
	Banned map[string]string `json:"banned"`
}

func NewAdminState() *AdminState {
	return &AdminState{Banned: make(map[string]string)}
}
