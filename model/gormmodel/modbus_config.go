package gormmodel

import (
	"database/sql"
	"zero-service/common/gormx"
)

// ModbusSlaveConfig Modbus从站配置表
type ModbusSlaveConfig struct {
	gormx.LegacyStringBaseModel
	gormx.VersionMixin
	ModbusCode              string         `gorm:"column:modbus_code;type:varchar(128);uniqueIndex:uk_modbus_slave_config_modbus_code;comment:Modbus配置唯一编码（如：modbus-192.168.1.100）"`
	SlaveAddress            string         `gorm:"column:slave_address;type:varchar(64);comment:TCP设备地址（格式：IP:Port，对应结构体Address）"`
	Slave                   int64          `gorm:"column:slave;comment:Modbus从站地址（Slave ID/Unit ID，对应结构体Slave）"`
	Timeout                 int64          `gorm:"column:timeout;default:10000;comment:发送/接收超时（单位：毫秒，对应结构体Timeout，默认10000）"`
	IdleTimeout             int64          `gorm:"column:idle_timeout;default:60000;comment:空闲断链毫秒数，默认60000，-1不主动断链"`
	LinkRecoveryTimeout     int64          `gorm:"column:link_recovery_timeout;default:0;comment:只读链路恢复窗口毫秒数，默认0关闭"`
	ProtocolRecoveryTimeout int64          `gorm:"column:protocol_recovery_timeout;default:0;comment:迟到响应恢复窗口毫秒数，默认0关闭"`
	ConnectDelay            int64          `gorm:"column:connect_delay;default:0;comment:建连后等待毫秒数，默认0不等待"`
	EnableTls               int            `gorm:"column:enable_tls;default:0;comment:是否启用TLS（对应结构体TLS.Enable：0-不启用，1-启用）"`
	TlsCertFile             sql.NullString `gorm:"column:tls_cert_file;type:varchar(512);comment:TLS客户端证书路径（对应结构体TLS.CertFile，enable_tls=1时生效）"`
	TlsKeyFile              sql.NullString `gorm:"column:tls_key_file;type:varchar(512);comment:TLS客户端密钥路径（对应结构体TLS.KeyFile，enable_tls=1时生效）"`
	TlsCaFile               sql.NullString `gorm:"column:tls_ca_file;type:varchar(512);comment:TLS根证书路径（对应结构体TLS.CAFile，enable_tls=1时生效）"`
	Status                  int            `gorm:"column:status;default:1;comment:配置状态：1-启用（可创建客户端），2-禁用（不加载）"`
	Remark                  sql.NullString `gorm:"column:remark;type:varchar(512);comment:备注（如：生产车间A-水泵控制从站）"`
}

func (ModbusSlaveConfig) TableName() string {
	return "modbus_slave_config"
}
