package gormmodel

import (
	"database/sql"
	"zero-service/common/gormx"
)

// ModbusSlaveConfig Modbus从站配置表
type ModbusSlaveConfig struct {
	gormx.LegacyBaseModel
	ModbusCode              string         `gorm:"column:modbus_code;type:varchar(128);uniqueIndex;comment:Modbus配置唯一编码（如：modbus-192.168.1.100）"`
	SlaveAddress            string         `gorm:"column:slave_address;type:varchar(64);comment:TCP设备地址（格式：IP:Port，对应结构体Address）"`
	Slave                   int64          `gorm:"column:slave;comment:Modbus从站地址（Slave ID/Unit ID，对应结构体Slave）"`
	Timeout                 int64          `gorm:"column:timeout;default:10000;comment:发送/接收超时（单位：毫秒，对应结构体Timeout，默认10000）"`
	IdleTimeout             int64          `gorm:"column:idle_timeout;default:60000;comment:空闲连接自动关闭时间（单位：毫秒，对应结构体IdleTimeout，默认60000）"`
	LinkRecoveryTimeout     int64          `gorm:"column:link_recovery_timeout;default:3000;comment:TCP连接出错重连间隔（单位：毫秒，对应结构体LinkRecoveryTimeout，默认3000）"`
	ProtocolRecoveryTimeout int64          `gorm:"column:protocol_recovery_timeout;default:2000;comment:协议异常重试间隔（单位：毫秒，对应结构体ProtocolRecoveryTimeout，默认2000）"`
	ConnectDelay            int64          `gorm:"column:connect_delay;default:100;comment:连接建立后等待时间（单位：毫秒，对应结构体ConnectDelay，默认100）"`
	EnableTls               int64          `gorm:"column:enable_tls;default:0;comment:是否启用TLS（对应结构体TLS.Enable：0-不启用，1-启用）"`
	TlsCertFile             sql.NullString `gorm:"column:tls_cert_file;type:varchar(512);comment:TLS客户端证书路径（对应结构体TLS.CertFile，enable_tls=1时生效）"`
	TlsKeyFile              sql.NullString `gorm:"column:tls_key_file;type:varchar(512);comment:TLS客户端密钥路径（对应结构体TLS.KeyFile，enable_tls=1时生效）"`
	TlsCaFile               sql.NullString `gorm:"column:tls_ca_file;type:varchar(512);comment:TLS根证书路径（对应结构体TLS.CAFile，enable_tls=1时生效）"`
	Status                  int64          `gorm:"column:status;default:1;comment:配置状态：1-启用（可初始化连接池），2-禁用（不加载）"`
	Remark                  sql.NullString `gorm:"column:remark;type:varchar(512);comment:备注（如：生产车间A-水泵控制从站）"`
}

func (ModbusSlaveConfig) TableName() string {
	return "modbus_slave_config"
}
