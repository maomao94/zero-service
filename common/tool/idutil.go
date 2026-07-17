package tool

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type IdUtil struct {
	Redis *redis.Redis
}

func NewIdUtil(redis *redis.Redis) *IdUtil {
	return &IdUtil{
		Redis: redis,
	}
}

func (u *IdUtil) NextId(outDescType string, category string) (string, error) {
	ids, err := u.NextIds(outDescType, category, 1)
	if err != nil {
		return "", err
	}
	return ids[0], nil
}

func (u *IdUtil) NextIds(outDescType string, category string, count int64) ([]string, error) {
	if len(outDescType) == 0 {
		outDescType = "O"
	}
	if count <= 0 {
		count = 1
	}

	now := time.Now()
	yy := now.Format("2006")             // 4位年份
	dateTime := now.Format("0102150405") // MMddHHmmss

	endId, err := u.getNextIds(outDescType, category, count)
	if err != nil {
		return nil, err
	}

	startId := endId - count + 1
	ids := make([]string, 0, count)
	for id := startId; id <= endId; id++ {
		seq := fmt.Sprintf("%04d", id)
		ids = append(ids, fmt.Sprintf("%s%s%s%s", outDescType, yy, dateTime, seq))
	}
	return ids, nil
}

func (u *IdUtil) getNextIds(key string, category string, count int64) (int64, error) {
	fKey := fmt.Sprintf("%s:outId_%s", category, key)
	id, err := u.Redis.Incrby(fKey, count)
	if err != nil {
		return 0, err
	}
	if id == count {
		err = u.Redis.Expire(fKey, 60)
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

// SimpleUUID 生成不带 "-" 的 UUID v4
func (u *IdUtil) SimpleUUID() (string, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(uid.String(), "-", ""), nil
}

// SimpleUUID 生成不带 "-" 的时间有序 UUID v7
func SimpleUUID() (string, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(uid.String(), "-", ""), nil
}

// UUID 生成带 "-" 的时间有序 UUID v7
func UUID() (string, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return uid.String(), nil
}
