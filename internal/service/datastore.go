package service

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
	ds.mu.Lock(); defer ds.mu.Unlock(); ds.ExpertData = append(ds.ExpertData, qa)
}

func (ds *DataStore) AddReference(qa model.QAPair) {
	ds.mu.Lock(); defer ds.mu.Unlock(); ds.ReferenceData = append(ds.ReferenceData, qa)
}