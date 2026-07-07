package tool

import (
	"fmt"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/random"
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
	if count > 10000 {
		return nil, fmt.Errorf("count must be less than or equal to 10000")
	}

	for {
		now := time.Now()
		yy := now.Format("2006")             // 4位年份
		dateTime := now.Format("0102150405") // MMddHHmmss
		endId, err := u.getNextIds(outDescType, category, dateTime, count)
		if err != nil {
			return nil, err
		}
		if endId > 10000 {
			time.Sleep(time.Until(now.Truncate(time.Second).Add(time.Second)))
			continue
		}

		startId := endId - count + 1
		ids := make([]string, 0, count)
		for id := startId; id <= endId; id++ {
			seq := fmt.Sprintf("%04d", id%10000)
			ids = append(ids, fmt.Sprintf("%s%s%s%s", outDescType, yy, dateTime, seq))
		}
		return ids, nil
	}
}

func (u *IdUtil) getNextId(key string, category string) (int64, error) {
	return u.getNextIds(key, category, time.Now().Format("0102150405"), 1)
}

func (u *IdUtil) getNextIds(key string, category string, dateTime string, count int64) (int64, error) {
	fKey := fmt.Sprintf("%s:outId_%s_%s", category, key, dateTime)
	id, err := u.Redis.Incrby(fKey, count)
	if err != nil {
		return 0, err
	}
	if id == count || id >= 10000 {
		err = u.Redis.Expire(fKey, 3600) // 1小时 = 3600秒
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

// SimpleUUID 生成不带 "-" 的 UUID v4
func (u *IdUtil) SimpleUUID() (string, error) {
	uid, err := random.UUIdV4()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(uid, "-", ""), nil
}

// SimpleUUID 生成不带 "-" 的 UUID v4
func SimpleUUID() (string, error) {
	uid, err := random.UUIdV4()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(uid, "-", ""), nil
}
