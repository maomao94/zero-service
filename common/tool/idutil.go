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

func (u *IdUtil) NextId(outDescType string, category string) string {
	id := u.getNextId(outDescType, category)
	now := time.Now()
	yy := now.Format("06")               // 2位年份
	dateTime := now.Format("0102150405") // MMddHHmmss
	seq := fmt.Sprintf("%02d", id%100)
	return fmt.Sprintf("%s%s%s%s", outDescType, yy, dateTime, seq)
}

func (u *IdUtil) getNextId(key string, category string) int64 {
	fKey := fmt.Sprintf("%s:outId_%s", category, key)
	id, err := u.Redis.Incr(fKey)
	if err != nil {
		return 0
	}
	if id <= 1 || id >= 100 {
		err = u.Redis.Expire(fKey, 3600) // 1小时 = 3600秒
		if err != nil {
			return 0
		}
	}
	return id
}

// SimpleUUID 生成不带 "-" 的 UUID v4
func (u *IdUtil) SimpleUUID() (string, error) {
	uid, err := random.UUIdV4()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(uid, "-", ""), nil
}
