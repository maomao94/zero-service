package configx

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

type KafkaPushConf struct {
	Brokers []string
	Topic   string
}

type KafkaMultiPushConf struct {
	Brokers []string
	Topics  []string
}

// KafkaConsumerConf is a consumer-only config; service.ServiceConf is injected at startup.
type KafkaConsumerConf struct {
	Brokers       []string
	Group         string
	Topic         string
	CaFile        string `json:",optional"`
	Offset        string `json:",options=first|last,default=last"`
	Conns         int    `json:",default=1"`
	Consumers     int    `json:",default=8"`
	Processors    int    `json:",default=8"`
	MinBytes      int    `json:",default=10240"`
	MaxBytes      int    `json:",default=10485760"`
	Username      string `json:",optional"`
	Password      string `json:",optional"`
	ForceCommit   bool   `json:",default=true"`
	CommitInOrder bool   `json:",default=false"`
}

// ToKqConf merges consumer fields with the given ServiceConf to produce a full kq.KqConf.
func (c KafkaConsumerConf) ToKqConf(svcConf service.ServiceConf) kq.KqConf {
	return kq.KqConf{
		ServiceConf:   svcConf,
		Brokers:       c.Brokers,
		Group:         c.Group,
		Topic:         c.Topic,
		CaFile:        c.CaFile,
		Offset:        c.Offset,
		Conns:         c.Conns,
		Consumers:     c.Consumers,
		Processors:    c.Processors,
		MinBytes:      c.MinBytes,
		MaxBytes:      c.MaxBytes,
		Username:      c.Username,
		Password:      c.Password,
		ForceCommit:   c.ForceCommit,
		CommitInOrder: c.CommitInOrder,
	}
}
