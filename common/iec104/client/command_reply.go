package client

import (
	"fmt"
	"time"

	"zero-service/common/antsx"

	"github.com/zeromicro/go-zero/core/logx"
)

// CommandAckStatus represents the outcome of a control command ACK.
type CommandAckStatus string

const (
	AckAccepted CommandAckStatus = "accepted"
	AckRejected CommandAckStatus = "rejected"
	AckCotError CommandAckStatus = "cot_error"
)

// CommandAck is the resolved ACK result for a pending control command.
type CommandAck struct {
	Status     CommandAckStatus
	TypeID     int
	Coa        uint
	Ioa        uint
	Value      any
	Cot        string
	CotCause   int
	IsNegative bool
}

// CommandReplyPool wraps antsx.ReplyPool for IEC104 control command ACK matching.
// Each IEC104 Client (per connection) owns its own pool to avoid cross-connection interference.
type CommandReplyPool struct {
	host       string
	port       int
	pool       *antsx.ReplyPool[*CommandAck]
	defaultTTL time.Duration
}

// NewCommandReplyPool creates a new per-connection command reply pool.
func NewCommandReplyPool(host string, port int, defaultTTL time.Duration) *CommandReplyPool {
	return &CommandReplyPool{
		host: host,
		port: port,
		pool: antsx.NewReplyPool[*CommandAck](
			antsx.WithName(fmt.Sprintf("cmd-reply-%s-%d", host, port)),
			antsx.WithDefaultTTL(defaultTTL),
		),
		defaultTTL: defaultTTL,
	}
}

// CommandKey builds a unique pending key for a control command target.
// Per-connection scoped: host:port is implicit from pool ownership.
func CommandKey(coa uint, typeID int, ioa uint) string {
	return fmt.Sprintf("%d:%d:%d", coa, typeID, ioa)
}

// Register registers a pending command and returns a Promise to await the ACK.
// Returns ErrDuplicateID if the same command target already has a pending entry.
func (r *CommandReplyPool) Register(key string, ttl ...time.Duration) (*antsx.Promise[*CommandAck], error) {
	promise, err := r.pool.Register(key, ttl...)
	if err != nil {
		return nil, err
	}
	logx.Debugf("[cmd-reply-%s-%d] registered %s", r.host, r.port, key)
	return promise, nil
}

// Resolve resolves a pending command with an accepted ACK.
func (r *CommandReplyPool) Resolve(key string, ack *CommandAck) bool {
	ok := r.pool.Resolve(key, ack)
	if ok {
		logx.Debugf("[cmd-reply-%s-%d] resolved %s: %s", r.host, r.port, key, ack.Status)
	}
	return ok
}

// Reject rejects a pending command with an error.
func (r *CommandReplyPool) Reject(key string, err error) bool {
	ok := r.pool.Reject(key, err)
	if ok {
		logx.Debugf("[cmd-reply-%s-%d] rejected %s: %v", r.host, r.port, key, err)
	}
	return ok
}

// Has checks whether a key has a pending entry.
func (r *CommandReplyPool) Has(key string) bool {
	return r.pool.Has(key)
}

// Close closes the pool and rejects all pending entries.
func (r *CommandReplyPool) Close() {
	r.pool.Close()
}
