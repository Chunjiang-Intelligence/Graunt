package store

import (
	"graunt/internal/model"
	"sync"
)

type DataStore struct {
	mu            sync.RWMutex
	ExpertData    []model.QAPair
	ReferenceData []model.QAPair
}

var GlobalDataStore = &DataStore{
	ExpertData:    make([]model.QAPair, 0),
	ReferenceData: make([]model.QAPair, 0),
}

func (ds *DataStore) AddExpert(qa model.QAPair) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.ExpertData = append(ds.ExpertData, qa)
}

func (ds *DataStore) AddReference(qa model.QAPair) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.ReferenceData = append(ds.ReferenceData, qa)
}

func (ds *DataStore) GetExpertData() []model.QAPair {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return append([]model.QAPair(nil), ds.ExpertData...)
}

func (ds *DataStore) GetReferenceData() []model.QAPair {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return append([]model.QAPair(nil), ds.ReferenceData...)
}