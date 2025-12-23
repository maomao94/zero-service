package executorx

import (
	"sync"

	"github.com/zeromicro/go-zero/core/executors"
)

type ChunkSender func(messages []string)

type ChunkMessagesPusher struct {
	inserter    *executors.ChunkExecutor
	chunkSender ChunkSender
	writerLock  sync.Mutex
}

func NewChunkMessagesPusher(chunkSender ChunkSender, chunkBytes int) *ChunkMessagesPusher {
	pusher := ChunkMessagesPusher{
		chunkSender: chunkSender,
	}

	pusher.inserter = executors.NewChunkExecutor(pusher.execute, executors.WithChunkBytes(chunkBytes))
	return &pusher
}

func (w *ChunkMessagesPusher) Write(val string) error {
	w.writerLock.Lock()
	defer w.writerLock.Unlock()
	return w.inserter.Add(val, len(val))
}

func (w *ChunkMessagesPusher) execute(vals []interface{}) {
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
