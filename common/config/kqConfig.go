package config

type KqConfig struct {
	Brokers []string
	Topic   string
	IsPush  bool `json:",optional"`
}
