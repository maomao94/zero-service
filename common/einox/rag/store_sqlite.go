package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
}

func newSQLiteStore(dataDir string) (*sqliteStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("rag sqlite mkdir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "einox_rag.sqlite")
	dsn := "file:" + dbPath + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &sqliteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *sqliteStore) migrate() error {
	ddl := []string{
		`CREATE TABLE IF NOT EXISTS rag_collections (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS rag_chunks (
			id TEXT PRIMARY KEY,
			collection_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			source_id TEXT NOT NULL,
			filename TEXT,
			text TEXT NOT NULL,
			dim INTEGER NOT NULL,
			embedding_json TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			FOREIGN KEY(collection_id) REFERENCES rag_collections(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_chunks_coll_user ON rag_chunks(collection_id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_rag_chunks_source ON rag_chunks(collection_id, source_id)`,
	}
	for _, q := range ddl {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("rag sqlite migrate: %w", err)
		}
	}
	return nil
}

func (s *sqliteStore) CreateCollection(ctx context.Context, userID, id, name string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO rag_collections(id,user_id,name,created_at) VALUES(?,?,?,?)`,
		id, userID, name, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("rag create collection: %w", err)
	}
	return nil
}

func (s *sqliteStore) DeleteCollection(ctx context.Context, userID, collectionID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM rag_collections WHERE id=? AND user_id=?`, collectionID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("collection not found")
	}
	return nil
}

func (s *sqliteStore) ListCollections(ctx context.Context, userID string) ([]Collection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,created_at FROM rag_collections WHERE user_id=? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Collection
	for rows.Next() {
		var id, name string
		var ts int64
		if err := rows.Scan(&id, &name, &ts); err != nil {
			return nil, err
		}
		out = append(out, Collection{ID: id, Name: name, CreatedAt: time.Unix(ts, 0)})
	}
	return out, rows.Err()
}

func (s *sqliteStore) UpsertChunks(ctx context.Context, userID, collectionID, sourceID, filename string, pairs []chunkVectorPair) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var own int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM rag_collections WHERE id=? AND user_id=?`, collectionID, userID).Scan(&own); err != nil {
		return err
	}
	if own == 0 {
		return fmt.Errorf("collection not found")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM rag_chunks WHERE collection_id=? AND source_id=?`, collectionID, sourceID); err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, p := range pairs {
		if len(p.Vector) == 0 {
			continue
		}
		b, err := json.Marshal(p.Vector)
		if err != nil {
			return err
		}
		id := p.ChunkID
		if id == "" {
			id = uuid.NewString()
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO rag_chunks(id,collection_id,user_id,source_id,filename,text,dim,embedding_json,created_at)
			VALUES(?,?,?,?,?,?,?,?,?)`,
			id, collectionID, userID, sourceID, filename, p.Text, len(p.Vector), string(b), now)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteStore) DeleteSource(ctx context.Context, userID, collectionID, sourceID string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM rag_chunks WHERE collection_id=? AND user_id=? AND source_id=?`,
		collectionID, userID, sourceID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("source not found or empty")
	}
	return nil
}

func (s *sqliteStore) ListSources(ctx context.Context, userID, collectionID string) ([]IngestedSource, error) {
	var own int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM rag_collections WHERE id=? AND user_id=?`, collectionID, userID).Scan(&own); err != nil {
		return nil, err
	}
	if own == 0 {
		return nil, fmt.Errorf("collection not found")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT source_id, MIN(filename) as fn, COUNT(1) as n, MIN(created_at) as t
		FROM rag_chunks WHERE collection_id=? AND user_id=?
		GROUP BY source_id ORDER BY t DESC`, collectionID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IngestedSource
	for rows.Next() {
		var sid, fn string
		var n, ts int64
		if err := rows.Scan(&sid, &fn, &n, &ts); err != nil {
			return nil, err
		}
		out = append(out, IngestedSource{ID: sid, Filename: fn, Chunks: int(n), CreatedAt: time.Unix(ts, 0)})
	}
	return out, rows.Err()
}

func (s *sqliteStore) Search(ctx context.Context, userID, collectionID string, query []float32, topK int) ([]storedHit, error) {
	var own int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM rag_collections WHERE id=? AND user_id=?`, collectionID, userID).Scan(&own); err != nil {
		return nil, err
	}
	if own == 0 {
		return nil, fmt.Errorf("collection not found")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,source_id,filename,text,embedding_json,dim FROM rag_chunks WHERE collection_id=? AND user_id=?`,
		collectionID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scored []storedHit
	for rows.Next() {
		var id, sid, fn, text, embJSON string
		var dim int
		if err := rows.Scan(&id, &sid, &fn, &text, &embJSON, &dim); err != nil {
			return nil, err
		}
		var vec []float32
		if err := json.Unmarshal([]byte(embJSON), &vec); err != nil {
			continue
		}
		if len(vec) != dim || len(vec) != len(query) {
			continue
		}
		scored = append(scored, storedHit{
			ChunkID: id, SourceID: sid, Filename: fn, Text: text,
			Score: cosineFloat32(query, vec),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if topK > 0 && len(scored) > topK {
		scored = scored[:topK]
	}
	return scored, nil
}

// Close 释放连接。
func (s *sqliteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
