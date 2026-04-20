package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Redis 存储：向量 JSON + 集合索引；检索时在进程内算余弦（与 gorm/sqlite 路径一致，适合中小规模）。
// 大规模请用 backend=milvus 或上游 Eino redis indexer + RediSearch 向量索引。

type redisStore struct {
	rdb       *redis.Client
	maxChunks int // 0 = no limit; see Config.EffectiveMaxChunksPerBase
}

type redisChunkPayload struct {
	UserID   string    `json:"u"`
	BaseID   string    `json:"b"`
	SourceID string    `json:"s"`
	Filename string    `json:"f"`
	Text     string    `json:"t"`
	Vector   []float32 `json:"v"`
	Created  int64     `json:"c"`
}

type redisBaseMeta struct {
	Name string `json:"n"`
	Ts   int64  `json:"t"`
}

func newRedisStore(cfg Config) (*redisStore, error) {
	addr := strings.TrimSpace(cfg.Redis.Addr)
	if addr == "" {
		return nil, fmt.Errorf("knowledge redis: Redis.addr is required when backend=redis")
	}
	opt := &redis.Options{
		Addr:     addr,
		Username: strings.TrimSpace(cfg.Redis.Username),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	return &redisStore{
		rdb:       redis.NewClient(opt),
		maxChunks: cfg.EffectiveMaxChunksPerBase(),
	}, nil
}

func (s *redisStore) basesKey(userID string) string {
	return fmt.Sprintf("kb:v3:bases:%s", userID)
}

func (s *redisStore) chunkKey(id string) string {
	return fmt.Sprintf("kb:v3:chunk:%s", id)
}

func (s *redisStore) chunkSetKey(userID, baseID string) string {
	return fmt.Sprintf("kb:v3:chunkids:%s:%s", userID, baseID)
}

func (s *redisStore) CreateBase(ctx context.Context, userID, id, name string) error {
	b, err := json.Marshal(redisBaseMeta{Name: name, Ts: time.Now().Unix()})
	if err != nil {
		return err
	}
	return s.rdb.HSet(ctx, s.basesKey(userID), id, b).Err()
}

func (s *redisStore) DeleteBase(ctx context.Context, userID, baseID string) error {
	ok, err := s.rdb.HExists(ctx, s.basesKey(userID), baseID).Result()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("knowledge base not found")
	}
	ids, err := s.rdb.SMembers(ctx, s.chunkSetKey(userID, baseID)).Result()
	if err != nil {
		return err
	}
	if len(ids) > 0 {
		keys := make([]string, 0, len(ids))
		for _, id := range ids {
			keys = append(keys, s.chunkKey(id))
		}
		_ = s.rdb.Del(ctx, append(keys, s.chunkSetKey(userID, baseID))...).Err()
	} else {
		_ = s.rdb.Del(ctx, s.chunkSetKey(userID, baseID)).Err()
	}
	return s.rdb.HDel(ctx, s.basesKey(userID), baseID).Err()
}

func (s *redisStore) ListBases(ctx context.Context, userID string) ([]Base, error) {
	m, err := s.rdb.HGetAll(ctx, s.basesKey(userID)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Base, 0, len(m))
	for id, raw := range m {
		var meta redisBaseMeta
		if err := json.Unmarshal([]byte(raw), &meta); err != nil {
			continue
		}
		out = append(out, Base{ID: id, Name: meta.Name, CreatedAt: time.Unix(meta.Ts, 0)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *redisStore) UpsertChunks(ctx context.Context, userID, baseID, sourceID, filename string, pairs []chunkVectorPair) error {
	ok, err := s.rdb.HExists(ctx, s.basesKey(userID), baseID).Result()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("knowledge base not found")
	}
	// 删除该 source 旧 chunk
	if err := s.deleteChunksBySource(ctx, userID, baseID, sourceID); err != nil {
		return err
	}
	nAdd := 0
	for _, p := range pairs {
		if len(p.Vector) > 0 {
			nAdd++
		}
	}
	if s.maxChunks > 0 {
		nCur, err := s.rdb.SCard(ctx, s.chunkSetKey(userID, baseID)).Result()
		if err != nil {
			return err
		}
		if int(nCur)+nAdd > s.maxChunks {
			return fmt.Errorf("knowledge redis: chunk count would exceed maxChunksPerBase (%d): have %d, adding %d", s.maxChunks, nCur, nAdd)
		}
	}
	now := time.Now().Unix()
	pipe := s.rdb.Pipeline()
	for _, p := range pairs {
		if len(p.Vector) == 0 {
			continue
		}
		cid := p.ChunkID
		if cid == "" {
			cid = uuid.NewString()
		}
		pl := redisChunkPayload{
			UserID: userID, BaseID: baseID, SourceID: sourceID, Filename: filename,
			Text: p.Text, Vector: p.Vector, Created: now,
		}
		raw, err := json.Marshal(pl)
		if err != nil {
			return err
		}
		pipe.Set(ctx, s.chunkKey(cid), raw, 0)
		pipe.SAdd(ctx, s.chunkSetKey(userID, baseID), cid)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *redisStore) deleteChunksBySource(ctx context.Context, userID, baseID, sourceID string) error {
	ids, err := s.rdb.SMembers(ctx, s.chunkSetKey(userID, baseID)).Result()
	if err != nil {
		return err
	}
	var toDel []string
	for _, cid := range ids {
		raw, err := s.rdb.Get(ctx, s.chunkKey(cid)).Bytes()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return err
		}
		var pl redisChunkPayload
		if json.Unmarshal(raw, &pl) != nil {
			continue
		}
		if pl.SourceID == sourceID {
			toDel = append(toDel, cid)
		}
	}
	if len(toDel) == 0 {
		return nil
	}
	pipe := s.rdb.Pipeline()
	for _, cid := range toDel {
		pipe.Del(ctx, s.chunkKey(cid))
		pipe.SRem(ctx, s.chunkSetKey(userID, baseID), cid)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *redisStore) DeleteSource(ctx context.Context, userID, baseID, sourceID string) error {
	ok, err := s.rdb.HExists(ctx, s.basesKey(userID), baseID).Result()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("knowledge base not found")
	}
	ids, err := s.rdb.SMembers(ctx, s.chunkSetKey(userID, baseID)).Result()
	if err != nil {
		return err
	}
	var match int
	for _, cid := range ids {
		raw, err := s.rdb.Get(ctx, s.chunkKey(cid)).Bytes()
		if err != nil {
			continue
		}
		var pl redisChunkPayload
		if json.Unmarshal(raw, &pl) != nil {
			continue
		}
		if pl.SourceID == sourceID {
			match++
		}
	}
	if match == 0 {
		return fmt.Errorf("source not found or empty")
	}
	return s.deleteChunksBySource(ctx, userID, baseID, sourceID)
}

func (s *redisStore) ListSources(ctx context.Context, userID, baseID string) ([]IndexedDocument, error) {
	ok, err := s.rdb.HExists(ctx, s.basesKey(userID), baseID).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("knowledge base not found")
	}
	ids, err := s.rdb.SMembers(ctx, s.chunkSetKey(userID, baseID)).Result()
	if err != nil {
		return nil, err
	}
	bySource := map[string]struct {
		name  string
		n     int
		first int64
	}{}
	for _, cid := range ids {
		raw, err := s.rdb.Get(ctx, s.chunkKey(cid)).Bytes()
		if err != nil {
			continue
		}
		var pl redisChunkPayload
		if json.Unmarshal(raw, &pl) != nil {
			continue
		}
		if pl.UserID != userID || pl.BaseID != baseID {
			continue
		}
		sid := pl.SourceID
		agg := bySource[sid]
		agg.n++
		if agg.name == "" {
			agg.name = pl.Filename
		}
		if agg.first == 0 || pl.Created < agg.first {
			agg.first = pl.Created
		}
		bySource[sid] = agg
	}
	var out []IndexedDocument
	for sid, v := range bySource {
		ts := v.first
		if ts == 0 {
			ts = time.Now().Unix()
		}
		out = append(out, IndexedDocument{ID: sid, Filename: v.name, Chunks: v.n, CreatedAt: time.Unix(ts, 0)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *redisStore) Search(ctx context.Context, userID, baseID string, query []float32, topK int) ([]storedHit, error) {
	ok, err := s.rdb.HExists(ctx, s.basesKey(userID), baseID).Result()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("knowledge base not found")
	}
	ids, err := s.rdb.SMembers(ctx, s.chunkSetKey(userID, baseID)).Result()
	if err != nil {
		return nil, err
	}
	var scored []storedHit
	for _, cid := range ids {
		raw, err := s.rdb.Get(ctx, s.chunkKey(cid)).Bytes()
		if err != nil {
			continue
		}
		var pl redisChunkPayload
		if json.Unmarshal(raw, &pl) != nil {
			continue
		}
		if pl.UserID != userID || pl.BaseID != baseID {
			continue
		}
		if len(pl.Vector) != len(query) {
			continue
		}
		scored = append(scored, storedHit{
			ChunkID: cid, SourceID: pl.SourceID, Filename: pl.Filename, Text: pl.Text,
			Score: cosineFloat32(query, pl.Vector),
		})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if topK > 0 && len(scored) > topK {
		scored = scored[:topK]
	}
	return scored, nil
}

func (s *redisStore) Close() error {
	if s.rdb == nil {
		return nil
	}
	return s.rdb.Close()
}
