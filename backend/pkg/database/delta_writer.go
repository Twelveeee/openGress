package database

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"sync"

	"github.com/Twelveeee/openGress/backend/pkg/state"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultDeltaBatchSize = 1000

type DeltaWriter struct {
	db        *gorm.DB
	batchSize int

	mu          sync.Mutex
	initialized bool
	baseline    modelFingerprintState
}

type modelFingerprintState struct {
	players     map[string]string
	inventories map[string]string
	portals     map[string]string
	links       map[string]string
	fields      map[string]string
	logs        map[string]string
	adminBans   map[string]string
}

type modelIndex struct {
	players     map[string]PlayerModel
	inventories map[string]PlayerInventoryModel
	portals     map[string]PortalModel
	links       map[string]LinkModel
	fields      map[string]FieldModel
	logs        map[string]LogModel
	adminBans   map[string]AdminBanModel
}

func NewDeltaWriter(db *gorm.DB, batchSize int) *DeltaWriter {
	if batchSize <= 0 {
		batchSize = defaultDeltaBatchSize
	}
	return &DeltaWriter{
		db:        db,
		batchSize: batchSize,
	}
}

func (w *DeltaWriter) SaveSnapshot(snapshot state.Snapshot) error {
	if w == nil || w.db == nil {
		return errors.New("database not initialized")
	}
	models, err := snapshotToModels(snapshot)
	if err != nil {
		return err
	}
	next, index, err := buildModelState(models)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.initialized {
		currentModels, err := loadModelSnapshotFromDB(w.db)
		if err != nil {
			return err
		}
		currentState, _, err := buildModelState(currentModels)
		if err != nil {
			return err
		}
		w.baseline = currentState
		w.initialized = true
	}

	if err := w.db.Transaction(func(tx *gorm.DB) error {
		if err := applyPlayerDelta(tx, w.baseline.players, next.players, index.players, w.batchSize); err != nil {
			return err
		}
		if err := applyInventoryDelta(tx, w.baseline.inventories, next.inventories, index.inventories, w.batchSize); err != nil {
			return err
		}
		if err := applyPortalDelta(tx, w.baseline.portals, next.portals, index.portals, w.batchSize); err != nil {
			return err
		}
		if err := applyLinkDelta(tx, w.baseline.links, next.links, index.links, w.batchSize); err != nil {
			return err
		}
		if err := applyFieldDelta(tx, w.baseline.fields, next.fields, index.fields, w.batchSize); err != nil {
			return err
		}
		if err := applyLogDelta(tx, w.baseline.logs, next.logs, index.logs, w.batchSize); err != nil {
			return err
		}
		if err := applyAdminBanDelta(tx, w.baseline.adminBans, next.adminBans, index.adminBans, w.batchSize); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	w.baseline = next
	return nil
}

func loadModelSnapshotFromDB(db *gorm.DB) (modelSnapshot, error) {
	models := modelSnapshot{}
	if err := db.Order("id asc").Find(&models.players).Error; err != nil {
		return modelSnapshot{}, err
	}
	if err := db.Order("player_id asc, item_id asc").Find(&models.inventories).Error; err != nil {
		return modelSnapshot{}, err
	}
	if err := db.Order("id asc").Find(&models.portals).Error; err != nil {
		return modelSnapshot{}, err
	}
	if err := db.Order("id asc").Find(&models.links).Error; err != nil {
		return modelSnapshot{}, err
	}
	if err := db.Order("id asc").Find(&models.fields).Error; err != nil {
		return modelSnapshot{}, err
	}
	if err := db.Order("timestamp asc").Find(&models.logs).Error; err != nil {
		return modelSnapshot{}, err
	}
	if err := db.Order("player_id asc").Find(&models.adminBans).Error; err != nil {
		return modelSnapshot{}, err
	}
	return models, nil
}

func buildModelState(models modelSnapshot) (modelFingerprintState, modelIndex, error) {
	index := modelIndex{
		players:     make(map[string]PlayerModel, len(models.players)),
		inventories: make(map[string]PlayerInventoryModel, len(models.inventories)),
		portals:     make(map[string]PortalModel, len(models.portals)),
		links:       make(map[string]LinkModel, len(models.links)),
		fields:      make(map[string]FieldModel, len(models.fields)),
		logs:        make(map[string]LogModel, len(models.logs)),
		adminBans:   make(map[string]AdminBanModel, len(models.adminBans)),
	}
	state := modelFingerprintState{
		players:     make(map[string]string, len(models.players)),
		inventories: make(map[string]string, len(models.inventories)),
		portals:     make(map[string]string, len(models.portals)),
		links:       make(map[string]string, len(models.links)),
		fields:      make(map[string]string, len(models.fields)),
		logs:        make(map[string]string, len(models.logs)),
		adminBans:   make(map[string]string, len(models.adminBans)),
	}

	for _, model := range models.players {
		index.players[model.ID] = model
		hash, err := modelFingerprint(model)
		if err != nil {
			return modelFingerprintState{}, modelIndex{}, err
		}
		state.players[model.ID] = hash
	}
	for _, model := range models.inventories {
		key := inventoryModelKey(model.PlayerID, model.ItemID)
		index.inventories[key] = model
		hash, err := modelFingerprint(model)
		if err != nil {
			return modelFingerprintState{}, modelIndex{}, err
		}
		state.inventories[key] = hash
	}
	for _, model := range models.portals {
		index.portals[model.ID] = model
		hash, err := modelFingerprint(model)
		if err != nil {
			return modelFingerprintState{}, modelIndex{}, err
		}
		state.portals[model.ID] = hash
	}
	for _, model := range models.links {
		index.links[model.ID] = model
		hash, err := modelFingerprint(model)
		if err != nil {
			return modelFingerprintState{}, modelIndex{}, err
		}
		state.links[model.ID] = hash
	}
	for _, model := range models.fields {
		index.fields[model.ID] = model
		hash, err := modelFingerprint(model)
		if err != nil {
			return modelFingerprintState{}, modelIndex{}, err
		}
		state.fields[model.ID] = hash
	}
	for _, model := range models.logs {
		index.logs[model.ID] = model
		hash, err := modelFingerprint(model)
		if err != nil {
			return modelFingerprintState{}, modelIndex{}, err
		}
		state.logs[model.ID] = hash
	}
	for _, model := range models.adminBans {
		index.adminBans[model.PlayerID] = model
		hash, err := modelFingerprint(model)
		if err != nil {
			return modelFingerprintState{}, modelIndex{}, err
		}
		state.adminBans[model.PlayerID] = hash
	}
	return state, index, nil
}

func inventoryModelKey(playerID, itemID string) string {
	return playerID + "\x00" + itemID
}

func splitInventoryModelKey(key string) (playerID, itemID string, ok bool) {
	for i := 0; i < len(key); i++ {
		if key[i] == '\x00' {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}

func modelFingerprint(value interface{}) (string, error) {
	switch model := value.(type) {
	case PlayerModel:
		// inventory_json 已迁移到 player_inventories，忽略旧列避免无效脏检查。
		model.LegacyInventoryJSON = nil
		value = model
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(raw)
	return hex.EncodeToString(sum[:]), nil
}

func diffModelIDs(oldState, newState map[string]string) (upsertIDs, deleteIDs []string) {
	for id, next := range newState {
		prev, ok := oldState[id]
		if !ok || prev != next {
			upsertIDs = append(upsertIDs, id)
		}
	}
	for id := range oldState {
		if _, ok := newState[id]; !ok {
			deleteIDs = append(deleteIDs, id)
		}
	}
	sort.Strings(upsertIDs)
	sort.Strings(deleteIDs)
	return upsertIDs, deleteIDs
}

func chunkIDs(ids []string, size int) [][]string {
	if size <= 0 || len(ids) == 0 {
		return nil
	}
	out := make([][]string, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}
		out = append(out, ids[start:end])
	}
	return out
}

func applyPlayerDelta(tx *gorm.DB, prev, next map[string]string, index map[string]PlayerModel, batchSize int) error {
	upsertIDs, deleteIDs := diffModelIDs(prev, next)
	for _, ids := range chunkIDs(upsertIDs, batchSize) {
		batch := make([]PlayerModel, 0, len(ids))
		for _, id := range ids {
			batch = append(batch, index[id])
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"username", "faction", "level", "ap", "xm", "max_xm",
				"portal_captures", "links_created", "fields_created", "mu_total",
				"pos_lat", "pos_lon", "auto_hack", "cooldowns_json",
				"view_json", "view_tiles_json", "target_json", "updated_at",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	for _, ids := range chunkIDs(deleteIDs, batchSize) {
		if err := tx.Where("id IN ?", ids).Delete(&PlayerModel{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func applyInventoryDelta(tx *gorm.DB, prev, next map[string]string, index map[string]PlayerInventoryModel, batchSize int) error {
	upsertIDs, deleteIDs := diffModelIDs(prev, next)
	for _, ids := range chunkIDs(upsertIDs, batchSize) {
		batch := make([]PlayerInventoryModel, 0, len(ids))
		for _, id := range ids {
			batch = append(batch, index[id])
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "player_id"}, {Name: "item_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"amount"}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	for _, ids := range chunkIDs(deleteIDs, batchSize) {
		query := tx.Model(&PlayerInventoryModel{})
		conditions := 0
		for _, id := range ids {
			playerID, itemID, ok := splitInventoryModelKey(id)
			if !ok {
				continue
			}
			if conditions == 0 {
				query = query.Where("player_id = ? AND item_id = ?", playerID, itemID)
			} else {
				query = query.Or("player_id = ? AND item_id = ?", playerID, itemID)
			}
			conditions++
		}
		if conditions == 0 {
			continue
		}
		if err := query.Delete(&PlayerInventoryModel{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func applyPortalDelta(tx *gorm.DB, prev, next map[string]string, index map[string]PortalModel, batchSize int) error {
	upsertIDs, deleteIDs := diffModelIDs(prev, next)
	for _, ids := range chunkIDs(upsertIDs, batchSize) {
		batch := make([]PortalModel, 0, len(ids))
		for _, id := range ids {
			batch = append(batch, index[id])
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"title", "cover_url", "latitude", "longitude", "faction", "level", "energy",
				"resonators_json", "mods_json", "updated_at",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	for _, ids := range chunkIDs(deleteIDs, batchSize) {
		if err := tx.Where("id IN ?", ids).Delete(&PortalModel{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func applyLinkDelta(tx *gorm.DB, prev, next map[string]string, index map[string]LinkModel, batchSize int) error {
	upsertIDs, deleteIDs := diffModelIDs(prev, next)
	for _, ids := range chunkIDs(upsertIDs, batchSize) {
		batch := make([]LinkModel, 0, len(ids))
		for _, id := range ids {
			batch = append(batch, index[id])
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"from_portal_id", "to_portal_id", "from_lat", "from_lon", "to_lat", "to_lon", "created_at",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	for _, ids := range chunkIDs(deleteIDs, batchSize) {
		if err := tx.Where("id IN ?", ids).Delete(&LinkModel{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func applyFieldDelta(tx *gorm.DB, prev, next map[string]string, index map[string]FieldModel, batchSize int) error {
	upsertIDs, deleteIDs := diffModelIDs(prev, next)
	for _, ids := range chunkIDs(upsertIDs, batchSize) {
		batch := make([]FieldModel, 0, len(ids))
		for _, id := range ids {
			batch = append(batch, index[id])
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"portal_1_id", "portal_2_id", "portal_3_id", "faction", "mu", "layer", "created_at",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	for _, ids := range chunkIDs(deleteIDs, batchSize) {
		if err := tx.Where("id IN ?", ids).Delete(&FieldModel{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func applyLogDelta(tx *gorm.DB, prev, next map[string]string, index map[string]LogModel, batchSize int) error {
	upsertIDs, deleteIDs := diffModelIDs(prev, next)
	for _, ids := range chunkIDs(upsertIDs, batchSize) {
		batch := make([]LogModel, 0, len(ids))
		for _, id := range ids {
			batch = append(batch, index[id])
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"type", "player_id", "portal_id", "faction", "mu", "message", "timestamp",
			}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	for _, ids := range chunkIDs(deleteIDs, batchSize) {
		if err := tx.Where("id IN ?", ids).Delete(&LogModel{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func applyAdminBanDelta(tx *gorm.DB, prev, next map[string]string, index map[string]AdminBanModel, batchSize int) error {
	upsertIDs, deleteIDs := diffModelIDs(prev, next)
	for _, ids := range chunkIDs(upsertIDs, batchSize) {
		batch := make([]AdminBanModel, 0, len(ids))
		for _, id := range ids {
			batch = append(batch, index[id])
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "player_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"reason", "updated_at"}),
		}).Create(&batch).Error; err != nil {
			return err
		}
	}
	for _, ids := range chunkIDs(deleteIDs, batchSize) {
		if err := tx.Where("player_id IN ?", ids).Delete(&AdminBanModel{}).Error; err != nil {
			return err
		}
	}
	return nil
}
