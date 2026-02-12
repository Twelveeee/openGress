package state

import (
	"encoding/json"
	"os"
	"time"
)

type Snapshot struct {
	// Snapshot 用于导入/导出状态。
	Version   int         `json:"version"`
	Timestamp time.Time   `json:"timestamp"`
	Players   []Player    `json:"players"`
	Portals   []Portal    `json:"portals"`
	Links     []Link      `json:"links,omitempty"`
	Fields    []Field     `json:"fields,omitempty"`
	Logs      []LogEntry  `json:"logs,omitempty"`
	Admin     *AdminState `json:"admin,omitempty"`
}

func (s *GameState) Snapshot() Snapshot {
	// 生成当前内存快照。
	var logs []LogEntry
	if s.Logs != nil {
		logs = s.Logs.List()
	}
	return Snapshot{
		Version:   2,
		Timestamp: time.Now(),
		Players:   s.Players.List(),
		Portals:   s.Portals.List(),
		Links:     s.Links.List(),
		Fields:    s.Fields.List(),
		Logs:      logs,
		Admin:     cloneAdminState(s.Admin),
	}
}

func SaveToFile(path string, snapshot Snapshot) error {
	// 保存快照到文件。
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadFromFile(path string) (Snapshot, error) {
	// 从文件读取快照。
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func NewGameStateFromSnapshot(snapshot Snapshot) *GameState {
	// 从快照构建新 GameState。
	gameState := NewGameState()
	for _, player := range snapshot.Players {
		gameState.Players.Upsert(player)
	}
	for _, portal := range snapshot.Portals {
		gameState.Portals.Upsert(portal)
	}
	for _, link := range snapshot.Links {
		gameState.Links.Upsert(link)
	}
	for _, field := range snapshot.Fields {
		gameState.Fields.Upsert(field)
	}
	if snapshot.Logs != nil {
		gameState.Logs = NewLogStore(len(snapshot.Logs))
		for _, entry := range snapshot.Logs {
			gameState.Logs.Add(entry)
		}
	}
	if snapshot.Admin != nil {
		gameState.Admin = cloneAdminState(snapshot.Admin)
	}
	return gameState
}

func cloneAdminState(admin *AdminState) *AdminState {
	if admin == nil {
		return nil
	}
	cloned := NewAdminState()
	for playerID, reason := range admin.Banned {
		cloned.Banned[playerID] = reason
	}
	return cloned
}
