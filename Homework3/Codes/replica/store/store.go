package store

import (
	"fmt"
	"sync"
	"time"
)

type Entry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Version   int64  `json:"version"`
	UpdatedBy string `json:"updated_by"`
	UpdatedAt string `json:"updated_at"`
}

type ConflictPolicy string

const (

	PolicyLastWriteWins ConflictPolicy = "last_write_wins"
)

type Store struct {
	mu        sync.RWMutex
	replicaID string
	data      map[string]*Entry
}

func New(replicaID string) *Store {
	return &Store{
		replicaID: replicaID,
		data:      make(map[string]*Entry),
	}
}

type PutResult struct {
	Entry     *Entry
	Conflict  bool   
	Skipped   bool   
	SkipMsg   string 
}

func (s *Store) Put(key, value string) *Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	var version int64 = 1
	if existing, ok := s.data[key]; ok {
		version = existing.Version + 1
	}

	entry := &Entry{
		Key:       key,
		Value:     value,
		Version:   version,
		UpdatedBy: s.replicaID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	s.data[key] = entry
	return entry
}

func (s *Store) ApplyReplication(incoming *Entry) PutResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.data[incoming.Key]
	if !ok {
		s.data[incoming.Key] = incoming
		return PutResult{Entry: incoming}
	}

	if incoming.Version > existing.Version {
		s.data[incoming.Key] = incoming
		return PutResult{Entry: incoming}
	}

	if incoming.Version < existing.Version {
		return PutResult{
			Entry:   existing,
			Skipped: true,
			SkipMsg: fmt.Sprintf("rejected v%d from %s (local is v%d)", incoming.Version, incoming.UpdatedBy, existing.Version),
		}
	}

	if incoming.Value != existing.Value {
		if incoming.UpdatedBy > existing.UpdatedBy {
			s.data[incoming.Key] = incoming
			return PutResult{Entry: incoming, Conflict: true}
		}
		return PutResult{Entry: existing, Conflict: true}
	}

	return PutResult{Entry: existing}
}

func (s *Store) Get(key string) *Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}

func (s *Store) Snapshot() map[string]*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]*Entry, len(s.data))
	for k, v := range s.data {
		cp := *v
		snapshot[k] = &cp
	}
	return snapshot
}
