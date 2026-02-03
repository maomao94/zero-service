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
	if len(outDescType) == 0 {
		outDescType = "O"
	}
	id, err := u.getNextId(outDescType, category)
	if err != nil {
		return "", err
	}
	now := time.Now()
	yy := now.Format("2006")             // 2位年份
	dateTime := now.Format("0102150405") // MMddHHmmss
	seq := fmt.Sprintf("%04d", id%10000)
	return fmt.Sprintf("%s%s%s%s", outDescType, yy, dateTime, seq), nil
}

func (u *IdUtil) getNextId(key string, category string) (int64, error) {
	fKey := fmt.Sprintf("%s:outId_%s", category, key)
	id, err := u.Redis.Incr(fKey)
	if err != nil {
		return 0, err
	}
	if id <= 1 || id >= 10000 {
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
