package config

type KqConfig struct {
	Brokers          []string
	AsduTopic        string
	BroadcastTopic   string `json:",optional,default=iec-broadcast"`
	BroadcastGroupId string `json:",optional,default=iec-caller"`
	IsPush           bool   `json:",optional"`
}
