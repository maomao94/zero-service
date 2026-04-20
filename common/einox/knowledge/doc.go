// Package knowledge 提供企业知识库向量索引：Embedding、写入、检索。
//
// 存储后端：memory、gorm（SQLite 文件或 DSN 连 MySQL/Postgres）、redis、milvus；
// 与 cloudwego/eino-ext 中 indexer（redis/milvus/pg 等）同属向量落地方向，本包保持独立业务 API（按用户/知识库隔离）。
//
// # 与 eino-ext Indexer 的分工
//
// eino-ext 的 indexer 将 schema.Document + 向量写入各向量库，适合纯 Eino Graph 编排或需要 RediSearch/Milvus 原生索引参数的场景。
// 本包的 [Service] 与 vectorStore 为 aisolo（search_knowledge_base）与 aigtw（/solo/v1/knowledge/*）提供同一套用户/知识库隔离与配置面；
// 部署上若网关与 aisolo 需共享索引，应优先对齐本包后端与 DSN/Redis/Milvus，而不是在两侧各接一套独立 indexer。
// 仅在编排层实验、或明确单进程专用时，可在 Graph 中直接使用 eino-ext indexer；与网关共库时再评估薄封装或数据迁移。
package knowledge
