package tool

import (
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GenOssFilename builds an OSS object path with a date partition and unique filename.
func GenOssFilename(filename, pathPrefix string) string {
	u, _ := uuid.NewUUID()
	return pathPrefix + "/" + time.Now().Format("20060102") + "/" +
		strings.ReplaceAll(u.String(), "-", "") +
		path.Ext(filename)
}
