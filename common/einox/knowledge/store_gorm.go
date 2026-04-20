package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zero-service/common/gormx"
)

// kbBase / kbChunk 表结构；兼容 sqlite 文件与 MySQL/Postgres（TEXT 存 JSON 向量）。
type kbBase struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"not null;index:idx_kb_bases_user"`
	Name      string `gorm:"not null"`
	CreatedAt int64  `gorm:"not null"`
}

func (kbBase) TableName() string { return "kb_bases" }

type kbChunk struct {
	ID            string `gorm:"primaryKey"`
	BaseID        string `gorm:"not null;index:idx_kb_chunks_base_user,priority:1"`
	UserID        string `gorm:"not null;index:idx_kb_chunks_base_user,priority:2"`
	SourceID      string `gorm:"not null;index:idx_kb_chunks_source,priority:1"`
	Filename      string
	Text          string `gorm:"not null"`
	Dim           int    `gorm:"not null"`
	EmbeddingJSON string `gorm:"column:embedding_json;not null"`
	CreatedAt     int64  `gorm:"not null"`
}

func (kbChunk) TableName() string { return "kb_chunks" }

type gormStore struct {
	db *gorm.DB
}

func newGORMStore(cfg Config) (*gormStore, error) {
	db, err := openGORMDB(cfg)
	if err != nil {
		return nil, err
	}
	s := &gormStore{db: db}
	if err := s.db.AutoMigrate(&kbBase{}, &kbChunk{}); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("knowledge gorm migrate: %w", err)
	}
	_ = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_kb_chunks_base_user ON kb_chunks(base_id, user_id)`).Error
	_ = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_kb_chunks_source ON kb_chunks(base_id, source_id)`).Error
	return s, nil
}

func openGORMDB(cfg Config) (*gorm.DB, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	gcfg := &gorm.Config{}
	if dsn == "" {
		if err := os.MkdirAll(cfg.EffectiveDataDir(), 0o755); err != nil {
			return nil, fmt.Errorf("knowledge gorm mkdir: %w", err)
		}
		path := filepath.Join(cfg.EffectiveDataDir(), "einox_knowledge.sqlite")
		db, err := gorm.Open(sqlite.Open(path), gcfg)
		if err != nil {
			return nil, err
		}
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxOpenConns(1)
		if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("knowledge sqlite pragma: %w", err)
		}
		if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
		return db, nil
	}

	dt := gormx.ParseDatabaseType(dsn)
	dialector, err := gormx.GetDialector(dt, dsn)
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(dialector, gcfg)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if dt == gormx.DatabaseSQLite {
		sqlDB.SetMaxOpenConns(1)
		_ = db.Exec("PRAGMA busy_timeout = 5000").Error
		_ = db.Exec("PRAGMA foreign_keys = ON").Error
	} else {
		sqlDB.SetMaxOpenConns(50)
	}
	return db, nil
}

func (s *gormStore) CreateBase(ctx context.Context, userID, id, name string) error {
	row := kbBase{
		ID:        id,
		UserID:    userID,
		Name:      name,
		CreatedAt: time.Now().Unix(),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("knowledge create base: %w", err)
	}
	return nil
}

func (s *gormStore) DeleteBase(ctx context.Context, userID, baseID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&kbBase{}).Where("id = ? AND user_id = ?", baseID, userID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("knowledge base not found")
		}
		if err := tx.Where("base_id = ?", baseID).Delete(&kbChunk{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", baseID, userID).Delete(&kbBase{}).Error
	})
}

func (s *gormStore) ListBases(ctx context.Context, userID string) ([]Base, error) {
	var rows []kbBase
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Base, 0, len(rows))
	for _, r := range rows {
		out = append(out, Base{
			ID:        r.ID,
			Name:      r.Name,
			CreatedAt: time.Unix(r.CreatedAt, 0),
		})
	}
	return out, nil
}

func (s *gormStore) UpsertChunks(ctx context.Context, userID, baseID, sourceID, filename string, pairs []chunkVectorPair) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&kbBase{}).Where("id = ? AND user_id = ?", baseID, userID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("knowledge base not found")
		}
		if err := tx.Where("base_id = ? AND source_id = ?", baseID, sourceID).Delete(&kbChunk{}).Error; err != nil {
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
			row := kbChunk{
				ID:            id,
				BaseID:        baseID,
				UserID:        userID,
				SourceID:      sourceID,
				Filename:      filename,
				Text:          p.Text,
				Dim:           len(p.Vector),
				EmbeddingJSON: string(b),
				CreatedAt:     now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *gormStore) DeleteSource(ctx context.Context, userID, baseID, sourceID string) error {
	tx := s.db.WithContext(ctx).
		Where("base_id = ? AND user_id = ? AND source_id = ?", baseID, userID, sourceID).
		Delete(&kbChunk{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("source not found or empty")
	}
	return nil
}

func (s *gormStore) ListSources(ctx context.Context, userID, baseID string) ([]IndexedDocument, error) {
	var n int64
	if err := s.db.WithContext(ctx).Model(&kbBase{}).Where("id = ? AND user_id = ?", baseID, userID).Count(&n).Error; err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("knowledge base not found")
	}
	var agg []struct {
		SourceID string `gorm:"column:source_id"`
		Fn       string `gorm:"column:fn"`
		Cnt      int64  `gorm:"column:n"`
		Ts       int64  `gorm:"column:t"`
	}
	err := s.db.WithContext(ctx).Raw(`
		SELECT source_id,
		       MIN(filename) AS fn,
		       COUNT(1) AS n,
		       MIN(created_at) AS t
		FROM kb_chunks
		WHERE base_id = ? AND user_id = ?
		GROUP BY source_id
		ORDER BY t DESC`, baseID, userID).Scan(&agg).Error
	if err != nil {
		return nil, err
	}
	out := make([]IndexedDocument, 0, len(agg))
	for _, r := range agg {
		out = append(out, IndexedDocument{
			ID:        r.SourceID,
			Filename:  r.Fn,
			Chunks:    int(r.Cnt),
			CreatedAt: time.Unix(r.Ts, 0),
		})
	}
	return out, nil
}

func (s *gormStore) Search(ctx context.Context, userID, baseID string, query []float32, topK int) ([]storedHit, error) {
	var n int64
	if err := s.db.WithContext(ctx).Model(&kbBase{}).Where("id = ? AND user_id = ?", baseID, userID).Count(&n).Error; err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, fmt.Errorf("knowledge base not found")
	}
	var chunks []kbChunk
	if err := s.db.WithContext(ctx).
		Where("base_id = ? AND user_id = ?", baseID, userID).
		Find(&chunks).Error; err != nil {
		return nil, err
	}
	var scored []storedHit
	for _, row := range chunks {
		var vec []float32
		if err := json.Unmarshal([]byte(row.EmbeddingJSON), &vec); err != nil {
			continue
		}
		if len(vec) != row.Dim || len(vec) != len(query) {
			continue
		}
		scored = append(scored, storedHit{
			ChunkID:  row.ID,
			SourceID: row.SourceID,
			Filename: row.Filename,
			Text:     row.Text,
			Score:    cosineFloat32(query, vec),
		})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if topK > 0 && len(scored) > topK {
		scored = scored[:topK]
	}
	return scored, nil
}

func (s *gormStore) Close() error {
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
