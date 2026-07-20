package holiday

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

func loadEntriesFromFS(ctx context.Context, fsys fs.FS, dir string) (map[string]Entry, error) {
	entries := make(map[string]Entry)
	files, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		path := filepath.ToSlash(filepath.Join(dir, file.Name()))
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, err
		}
		var yearly map[string]Entry
		if err := json.Unmarshal(data, &yearly); err != nil {
			return nil, fmt.Errorf("parse holiday data %s: %w", path, err)
		}
		for date, entry := range yearly {
			entry = normalizeEntry(entry)
			if err := validateEntry(date, entry); err != nil {
				return nil, fmt.Errorf("parse holiday data %s: %w", path, err)
			}
			if _, exists := entries[date]; exists {
				return nil, fmt.Errorf("parse holiday data %s: duplicate date %q", path, date)
			}
			entries[date] = entry
		}
	}
	return entries, nil
}

func normalizeEntry(entry Entry) Entry {
	if entry.Name != "" {
		return entry
	}
	entry.Name = strings.TrimSuffix(entry.Note, "补班")
	return entry
}

func validateEntry(date string, entry Entry) error {
	if err := validateDate(date); err != nil {
		return fmt.Errorf("invalid date %q", date)
	}
	if entry.Name == "" {
		return fmt.Errorf("empty name for %s", date)
	}
	switch entry.Type {
	case DayTypeHoliday, DayTypeWorkday:
		return nil
	default:
		return fmt.Errorf("invalid type %q for %s", entry.Type, date)
	}
}

func validateDate(date string) error {
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		return err
	}
	return nil
}
