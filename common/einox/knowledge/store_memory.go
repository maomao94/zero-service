package knowledge

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type memoryStore struct {
	mu           sync.RWMutex
	bases        map[string]memBase
	chunks       map[string][]memChunk
	chunkByID    map[string]memChunk
	userIndexCol map[string]map[string]struct{}
}

type memBase struct {
	ID        string
	UserID    string
	Name      string
	CreatedAt time.Time
}

type memChunk struct {
	ID        string
	BaseID    string
	UserID    string
	SourceID  string
	Filename  string
	Text      string
	Vector    []float32
	CreatedAt time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		bases:        make(map[string]memBase),
		chunks:       make(map[string][]memChunk),
		chunkByID:    make(map[string]memChunk),
		userIndexCol: make(map[string]map[string]struct{}),
	}
}

func (m *memoryStore) CreateBase(ctx context.Context, userID, id, name string) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.bases[id]; ok {
		return fmt.Errorf("knowledge base %q exists", id)
	}
	m.bases[id] = memBase{ID: id, UserID: userID, Name: name, CreatedAt: time.Now()}
	if m.userIndexCol[userID] == nil {
		m.userIndexCol[userID] = make(map[string]struct{})
	}
	m.userIndexCol[userID][id] = struct{}{}
	return nil
}

func (m *memoryStore) DeleteBase(ctx context.Context, userID, baseID string) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.bases[baseID]
	if !ok || c.UserID != userID {
		return fmt.Errorf("knowledge base not found")
	}
	delete(m.bases, baseID)
	for _, ch := range m.chunks[baseID] {
		delete(m.chunkByID, ch.ID)
	}
	delete(m.chunks, baseID)
	if m.userIndexCol[userID] != nil {
		delete(m.userIndexCol[userID], baseID)
	}
	return nil
}

func (m *memoryStore) ListBases(ctx context.Context, userID string) ([]Base, error) {
	_ = ctx
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Base
	for id := range m.userIndexCol[userID] {
		if c, ok := m.bases[id]; ok {
			out = append(out, Base{ID: c.ID, Name: c.Name, CreatedAt: c.CreatedAt})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *memoryStore) UpsertChunks(ctx context.Context, userID, baseID, sourceID, filename string, pairs []chunkVectorPair) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.bases[baseID]
	if !ok || c.UserID != userID {
		return fmt.Errorf("knowledge base not found")
	}
	var filtered []memChunk
	if arr, ok := m.chunks[baseID]; ok {
		for _, ch := range arr {
			if ch.SourceID == sourceID {
				delete(m.chunkByID, ch.ID)
				continue
			}
			filtered = append(filtered, ch)
		}
	}
	for _, p := range pairs {
		ch := memChunk{
			ID: uuid.NewString(), BaseID: baseID, UserID: userID,
			SourceID: sourceID, Filename: filename, Text: p.Text, Vector: append([]float32(nil), p.Vector...),
			CreatedAt: time.Now(),
		}
		filtered = append(filtered, ch)
		m.chunkByID[ch.ID] = ch
	}
	m.chunks[baseID] = filtered
	return nil
}

func (m *memoryStore) DeleteSource(ctx context.Context, userID, baseID, sourceID string) error {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.bases[baseID]
	if !ok || c.UserID != userID {
		return fmt.Errorf("knowledge base not found")
	}
	var next []memChunk
	for _, ch := range m.chunks[baseID] {
		if ch.SourceID == sourceID {
			delete(m.chunkByID, ch.ID)
			continue
		}
		next = append(next, ch)
	}
	m.chunks[baseID] = next
	return nil
}

func (m *memoryStore) ListSources(ctx context.Context, userID, baseID string) ([]IndexedDocument, error) {
	_ = ctx
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.bases[baseID]
	if !ok || c.UserID != userID {
		return nil, fmt.Errorf("knowledge base not found")
	}
	bySource := map[string]struct {
		name   string
		chunks int
		first  time.Time
	}{}
	for _, ch := range m.chunks[baseID] {
		if ch.UserID != userID {
			continue
		}
		s := bySource[ch.SourceID]
		s.chunks++
		if s.name == "" {
			s.name = ch.Filename
		}
		if s.first.IsZero() || ch.CreatedAt.Before(s.first) {
			s.first = ch.CreatedAt
		}
		bySource[ch.SourceID] = s
	}
	var out []IndexedDocument
	for sid, v := range bySource {
		out = append(out, IndexedDocument{ID: sid, Filename: v.name, Chunks: v.chunks, CreatedAt: v.first})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *memoryStore) Search(ctx context.Context, userID, baseID string, query []float32, topK int) ([]storedHit, error) {
	_ = ctx
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.bases[baseID]
	if !ok || c.UserID != userID {
		return nil, fmt.Errorf("knowledge base not found")
	}
	var scored []storedHit
	for _, ch := range m.chunks[baseID] {
		if ch.UserID != userID {
			continue
		}
		s := cosineFloat32(query, ch.Vector)
		scored = append(scored, storedHit{
			ChunkID: ch.ID, SourceID: ch.SourceID, Filename: ch.Filename, Text: ch.Text, Score: s,
		})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if topK > 0 && len(scored) > topK {
		scored = scored[:topK]
	}
	return scored, nil
}

func (m *memoryStore) Close() error {
	return nil
}

func cosineFloat32(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
