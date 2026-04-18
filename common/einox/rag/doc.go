// Package rag 提供与 Eino 生态一致的 RAG / 向量检索抽象层命名与数据形态。
//
// 命名与流程对齐 cloudwego/eino-examples/quickstart/chatwitheino/rag：
//   - 文本分块产出 []*schema.Document（与示例中 load → chunk 节点一致）；
//   - 对外工具/编排侧可继续采用「文档 + 问题 → 摘录/答案」的 Input/Output 思路；
//   - 向量路径下与 eino-ext 习惯一致：Embedding（向量）、Indexer（写入）、Retriever（检索）。
//
// 本包不依赖 ADK；仅承载配置、文档块与检索结果类型，供 aisolo、aigtw 等注入。

package rag
