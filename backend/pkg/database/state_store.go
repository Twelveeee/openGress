package database

import (
	"errors"
	"sync"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func LoadSnapshotFromDB() (state.Snapshot, error) {
	if GormDB == nil {
		return state.Snapshot{}, errors.New("database not initialized")
	}
	models, err := loadModelSnapshotFromDB(GormDB)
	if err != nil {
		return state.Snapshot{}, err
	}
	return modelsToSnapshot(models)
}

var (
	defaultDeltaWriterMu sync.Mutex
	defaultDeltaWriter   *DeltaWriter
	defaultDeltaWriterDB interface{}
)

func SaveSnapshotToDB(snapshot state.Snapshot) error {
	if GormDB == nil {
		return errors.New("database not initialized")
	}
	defaultDeltaWriterMu.Lock()
	if defaultDeltaWriter == nil || defaultDeltaWriterDB != GormDB {
		defaultDeltaWriter = NewDeltaWriter(GormDB, defaultDeltaBatchSize)
		defaultDeltaWriterDB = GormDB
	}
	writer := defaultDeltaWriter
	defaultDeltaWriterMu.Unlock()
	return writer.SaveSnapshot(snapshot)
}
