package knowledge

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// Milvus 实现：单 collection + 过滤表达式，向量检索走 Milvus Search（对齐 eino-ext indexer/milvus 用法）。
// 元数据（知识库列表）存内存 map；向量与文本在 Milvus。

type milvusStore struct {
	cli   client.Client
	coll  string
	dim   int
	mu    sync.RWMutex
	bases map[string]map[string]memBase // userID -> baseID
}

func newMilvusStore(cfg Config) (*milvusStore, error) {
	addr := strings.TrimSpace(cfg.Milvus.Addr)
	if addr == "" {
		return nil, fmt.Errorf("knowledge milvus: Milvus.addr is required when backend=milvus")
	}
	dim := cfg.Milvus.VectorDim
	if dim <= 0 {
		return nil, fmt.Errorf("knowledge milvus: Milvus.vectorDim must match embedding output dimension")
	}
	ctx := context.Background()
	var cli client.Client
	var err error
	if cfg.Milvus.Username != "" || cfg.Milvus.Password != "" {
		cli, err = client.NewDefaultGrpcClientWithURI(ctx, addr, cfg.Milvus.Username, cfg.Milvus.Password)
	} else {
		cli, err = client.NewGrpcClient(ctx, addr)
	}
	if err != nil {
		return nil, fmt.Errorf("knowledge milvus connect: %w", err)
	}
	coll := cfg.EffectiveMilvusCollection()
	s := &milvusStore{
		cli:   cli,
		coll:  coll,
		dim:   dim,
		bases: make(map[string]map[string]memBase),
	}
	if err := s.ensureCollection(ctx, dim); err != nil {
		_ = cli.Close()
		return nil, err
	}
	if err := s.reloadMeta(ctx); err != nil {
		_ = cli.Close()
		return nil, err
	}
	return s, nil
}

const milvusMetaSource = "__meta__"

func (s *milvusStore) reloadMeta(ctx context.Context) error {
	rs, err := s.cli.Query(ctx, s.coll, []string{},
		fmt.Sprintf(`source_id == %s`, milvusEsc(milvusMetaSource)),
		[]string{"user_id", "base_id", "text", "ts"},
		client.WithLimit(65536))
	if err != nil {
		return fmt.Errorf("knowledge milvus reloadMeta: %w", err)
	}
	uc := rs.GetColumn("user_id")
	bc := rs.GetColumn("base_id")
	tc := rs.GetColumn("text")
	tsc := rs.GetColumn("ts")
	if uc == nil || bc == nil || tc == nil || tsc == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < uc.Len(); i++ {
		uid, _ := uc.GetAsString(i)
		bid, _ := bc.GetAsString(i)
		name, _ := tc.GetAsString(i)
		tsv, _ := tsc.GetAsInt64(i)
		if s.bases[uid] == nil {
			s.bases[uid] = make(map[string]memBase)
		}
		s.bases[uid][bid] = memBase{
			ID: bid, UserID: uid, Name: name, CreatedAt: time.Unix(tsv, 0),
		}
	}
	return nil
}

func floatVectorDimFromSchema(sch *entity.Schema, fieldName string) (int, error) {
	if sch == nil {
		return 0, fmt.Errorf("nil schema")
	}
	for _, f := range sch.Fields {
		if f == nil {
			continue
		}
		if f.Name != fieldName || f.DataType != entity.FieldTypeFloatVector {
			continue
		}
		dimStr, ok := f.TypeParams[entity.TypeParamDim]
		if !ok || strings.TrimSpace(dimStr) == "" {
			return 0, fmt.Errorf("field %q has no %s type param", fieldName, entity.TypeParamDim)
		}
		d, err := strconv.Atoi(strings.TrimSpace(dimStr))
		if err != nil {
			return 0, fmt.Errorf("field %q dim: %w", fieldName, err)
		}
		return d, nil
	}
	return 0, fmt.Errorf("no float vector field %q in collection schema", fieldName)
}

func (s *milvusStore) ensureCollection(ctx context.Context, dim int) error {
	ok, err := s.cli.HasCollection(ctx, s.coll)
	if err != nil {
		return err
	}
	dimStr := strconv.Itoa(dim)
	sch := entity.NewSchema().WithName(s.coll).WithDescription("einox knowledge chunks").WithAutoID(false).
		WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(512).WithIsPrimaryKey(true)).
		WithField(entity.NewField().WithName("user_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(256)).
		WithField(entity.NewField().WithName("base_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(256)).
		WithField(entity.NewField().WithName("source_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(256)).
		WithField(entity.NewField().WithName("filename").WithDataType(entity.FieldTypeVarChar).WithMaxLength(1024)).
		WithField(entity.NewField().WithName("text").WithDataType(entity.FieldTypeVarChar).WithMaxLength(8192)).
		WithField(entity.NewField().WithName("ts").WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().WithName("vec").WithDataType(entity.FieldTypeFloatVector).WithTypeParams(entity.TypeParamDim, dimStr))
	if !ok {
		if err := s.cli.CreateCollection(ctx, sch, 1); err != nil {
			return fmt.Errorf("knowledge milvus CreateCollection: %w", err)
		}
		idx, err := entity.NewIndexFlat(entity.COSINE)
		if err != nil {
			return err
		}
		if err := s.cli.CreateIndex(ctx, s.coll, "vec", idx, false); err != nil {
			return fmt.Errorf("knowledge milvus CreateIndex: %w", err)
		}
	} else {
		coll, err := s.cli.DescribeCollection(ctx, s.coll)
		if err != nil {
			return fmt.Errorf("knowledge milvus DescribeCollection: %w", err)
		}
		got, err := floatVectorDimFromSchema(coll.Schema, "vec")
		if err != nil {
			return fmt.Errorf("knowledge milvus: %w", err)
		}
		if got != dim {
			return fmt.Errorf("knowledge milvus: collection %q field vec dim %d != configured Milvus.vectorDim %d (drop collection or fix config)", s.coll, got, dim)
		}
	}
	if err := s.cli.LoadCollection(ctx, s.coll, false); err != nil {
		return fmt.Errorf("knowledge milvus LoadCollection: %w", err)
	}
	return nil
}

func milvusEsc(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

func (s *milvusStore) CreateBase(ctx context.Context, userID, id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bases[userID] == nil {
		s.bases[userID] = make(map[string]memBase)
	}
	if _, ok := s.bases[userID][id]; ok {
		return fmt.Errorf("knowledge base %q exists", id)
	}
	now := time.Now().Unix()
	metaPK := "meta:" + userID + ":" + id
	zvec := make([]float32, s.dim)
	_, err := s.cli.Insert(ctx, s.coll, "",
		entity.NewColumnVarChar("id", []string{metaPK}),
		entity.NewColumnVarChar("user_id", []string{userID}),
		entity.NewColumnVarChar("base_id", []string{id}),
		entity.NewColumnVarChar("source_id", []string{milvusMetaSource}),
		entity.NewColumnVarChar("filename", []string{""}),
		entity.NewColumnVarChar("text", []string{name}),
		entity.NewColumnInt64("ts", []int64{now}),
		entity.NewColumnFloatVector("vec", s.dim, [][]float32{zvec}),
	)
	if err != nil {
		return fmt.Errorf("knowledge milvus CreateBase meta: %w", err)
	}
	_ = s.cli.Flush(ctx, s.coll, false)
	s.bases[userID][id] = memBase{ID: id, UserID: userID, Name: name, CreatedAt: time.Unix(now, 0)}
	return nil
}

func (s *milvusStore) DeleteBase(ctx context.Context, userID, baseID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.bases[userID]
	if !ok {
		return fmt.Errorf("knowledge base not found")
	}
	if _, ok := u[baseID]; !ok {
		return fmt.Errorf("knowledge base not found")
	}
	expr := fmt.Sprintf("user_id == %s && base_id == %s", milvusEsc(userID), milvusEsc(baseID))
	if err := s.cli.Delete(ctx, s.coll, "", expr); err != nil {
		return err
	}
	delete(u, baseID)
	return nil
}

func (s *milvusStore) ListBases(ctx context.Context, userID string) ([]Base, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.bases[userID]
	if !ok {
		return nil, nil
	}
	var out []Base
	for _, b := range u {
		out = append(out, Base{ID: b.ID, Name: b.Name, CreatedAt: b.CreatedAt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *milvusStore) UpsertChunks(ctx context.Context, userID, baseID, sourceID, filename string, pairs []chunkVectorPair) error {
	s.mu.RLock()
	u, ok := s.bases[userID]
	_, baseOk := u[baseID]
	s.mu.RUnlock()
	if !ok || !baseOk {
		return fmt.Errorf("knowledge base not found")
	}
	delExpr := fmt.Sprintf("user_id == %s && base_id == %s && source_id == %s", milvusEsc(userID), milvusEsc(baseID), milvusEsc(sourceID))
	if err := s.cli.Delete(ctx, s.coll, "", delExpr); err != nil {
		return err
	}
	if len(pairs) == 0 {
		return nil
	}
	now := time.Now().Unix()
	var ids, uids, bids, sids, fns, texts []string
	var tss []int64
	var vecs [][]float32
	for _, p := range pairs {
		if len(p.Vector) == 0 || len(p.Vector) != s.dim {
			continue
		}
		id := p.ChunkID
		if id == "" {
			id = uuid.NewString()
		}
		ids = append(ids, id)
		uids = append(uids, userID)
		bids = append(bids, baseID)
		sids = append(sids, sourceID)
		fns = append(fns, filename)
		texts = append(texts, p.Text)
		tss = append(tss, now)
		vv := make([]float32, len(p.Vector))
		copy(vv, p.Vector)
		vecs = append(vecs, vv)
	}
	if len(ids) == 0 {
		return nil
	}
	_, err := s.cli.Insert(ctx, s.coll, "",
		entity.NewColumnVarChar("id", ids),
		entity.NewColumnVarChar("user_id", uids),
		entity.NewColumnVarChar("base_id", bids),
		entity.NewColumnVarChar("source_id", sids),
		entity.NewColumnVarChar("filename", fns),
		entity.NewColumnVarChar("text", texts),
		entity.NewColumnInt64("ts", tss),
		entity.NewColumnFloatVector("vec", s.dim, vecs),
	)
	if err != nil {
		return fmt.Errorf("knowledge milvus Insert: %w", err)
	}
	_ = s.cli.Flush(ctx, s.coll, false)
	return nil
}

func (s *milvusStore) DeleteSource(ctx context.Context, userID, baseID, sourceID string) error {
	s.mu.RLock()
	u, ok := s.bases[userID]
	_, baseOk := u[baseID]
	s.mu.RUnlock()
	if !ok || !baseOk {
		return fmt.Errorf("knowledge base not found")
	}
	expr := fmt.Sprintf("user_id == %s && base_id == %s && source_id == %s", milvusEsc(userID), milvusEsc(baseID), milvusEsc(sourceID))
	if err := s.cli.Delete(ctx, s.coll, "", expr); err != nil {
		return err
	}
	return nil
}

func (s *milvusStore) ListSources(ctx context.Context, userID, baseID string) ([]IndexedDocument, error) {
	s.mu.RLock()
	u, ok := s.bases[userID]
	_, baseOk := u[baseID]
	s.mu.RUnlock()
	if !ok || !baseOk {
		return nil, fmt.Errorf("knowledge base not found")
	}
	expr := fmt.Sprintf("user_id == %s && base_id == %s && source_id != %s",
		milvusEsc(userID), milvusEsc(baseID), milvusEsc(milvusMetaSource))
	rs, err := s.cli.Query(ctx, s.coll, []string{}, expr, []string{"source_id", "filename", "ts"})
	if err != nil {
		return nil, err
	}
	srcCol := rs.GetColumn("source_id")
	fnCol := rs.GetColumn("filename")
	tsCol := rs.GetColumn("ts")
	if srcCol == nil || fnCol == nil || tsCol == nil {
		return nil, nil
	}
	n := srcCol.Len()
	by := map[string]struct {
		fn  string
		cnt int
		ts  int64
	}{}
	for i := 0; i < n; i++ {
		sid, _ := srcCol.GetAsString(i)
		fn, _ := fnCol.GetAsString(i)
		tsv, _ := tsCol.GetAsInt64(i)
		agg := by[sid]
		agg.cnt++
		if agg.fn == "" {
			agg.fn = fn
		}
		if agg.ts == 0 || tsv < agg.ts {
			agg.ts = tsv
		}
		by[sid] = agg
	}
	var out []IndexedDocument
	for sid, v := range by {
		ts := v.ts
		if ts == 0 {
			ts = time.Now().Unix()
		}
		out = append(out, IndexedDocument{ID: sid, Filename: v.fn, Chunks: v.cnt, CreatedAt: time.Unix(ts, 0)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *milvusStore) Search(ctx context.Context, userID, baseID string, query []float32, topK int) ([]storedHit, error) {
	s.mu.RLock()
	u, ok := s.bases[userID]
	_, baseOk := u[baseID]
	s.mu.RUnlock()
	if !ok || !baseOk {
		return nil, fmt.Errorf("knowledge base not found")
	}
	if len(query) != s.dim {
		return nil, fmt.Errorf("knowledge milvus: query dim %d != collection dim %d", len(query), s.dim)
	}
	if topK <= 0 {
		topK = 5
	}
	expr := fmt.Sprintf("user_id == %s && base_id == %s && source_id != %s",
		milvusEsc(userID), milvusEsc(baseID), milvusEsc(milvusMetaSource))
	sp, err := entity.NewIndexFlatSearchParam()
	if err != nil {
		return nil, err
	}
	res, err := s.cli.Search(ctx, s.coll, []string{}, expr,
		[]string{"id", "source_id", "filename", "text"},
		[]entity.Vector{entity.FloatVector(query)},
		"vec", entity.COSINE, topK, sp,
	)
	if err != nil {
		return nil, fmt.Errorf("knowledge milvus Search: %w", err)
	}
	if len(res) == 0 {
		return nil, nil
	}
	r0 := res[0]
	idCol := r0.Fields.GetColumn("id")
	srcCol := r0.Fields.GetColumn("source_id")
	fnCol := r0.Fields.GetColumn("filename")
	txtCol := r0.Fields.GetColumn("text")
	if idCol == nil || r0.ResultCount == 0 {
		return nil, nil
	}
	var hits []storedHit
	for i := 0; i < r0.ResultCount; i++ {
		cid, _ := idCol.GetAsString(i)
		sid, _ := srcCol.GetAsString(i)
		fn, _ := fnCol.GetAsString(i)
		txt, _ := txtCol.GetAsString(i)
		sc := float64(0)
		if i < len(r0.Scores) {
			// COSINE distance in Milvus: smaller is more similar for some versions; expose raw score
			sc = float64(r0.Scores[i])
		}
		hits = append(hits, storedHit{ChunkID: cid, SourceID: sid, Filename: fn, Text: txt, Score: sc})
	}
	return hits, nil
}

func (s *milvusStore) Close() error {
	if s.cli == nil {
		return nil
	}
	return s.cli.Close()
}
