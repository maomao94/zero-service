package iec104

import (
	"sync"

	"github.com/zeromicro/go-zero/core/executors"
)

type ChunkSender func(msgs []string)

type ChunkAsduPusher struct {
	inserter    *executors.ChunkExecutor
	chunkSender ChunkSender
	writerLock  sync.Mutex
}

func NewChunkAsduPusher(chunkSender ChunkSender, chunkBytes int) *ChunkAsduPusher {
	pusher := ChunkAsduPusher{
		chunkSender: chunkSender,
	}

	pusher.inserter = executors.NewChunkExecutor(pusher.execute, executors.WithChunkBytes(chunkBytes))
	return &pusher
}

func (w *ChunkAsduPusher) Write(val string) error {
	w.writerLock.Lock()
	defer w.writerLock.Unlock()
	return w.inserter.Add(val, len(val))
}

func (w *ChunkAsduPusher) execute(vals []interface{}) {
	msgs := make([]string, 0, len(vals))
	for _, val := range vals {
		if s, ok := val.(string); ok {
			msgs = append(msgs, s)
		}
	}

	if len(msgs) == 0 {
		return
	}
	w.chunkSender(msgs)
}
