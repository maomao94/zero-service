//go:build integration

package knowledge

import (
	"os"
	"testing"
)

// 需要可连通的 Milvus（如本地 docker-compose），未设置 MILVUS_TEST_ADDR 时跳过。
func TestMilvusNewStoreRequiresEnv(t *testing.T) {
	if os.Getenv("MILVUS_TEST_ADDR") == "" {
		t.Skip("set MILVUS_TEST_ADDR=host:19530 to run milvus integration smoke")
	}
	t.Log("integration placeholder: extend with NewMilvusStore against MILVUS_TEST_ADDR when CI provides Milvus")
}
