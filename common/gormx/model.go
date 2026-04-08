package gormx

import (
	"time"

	"gorm.io/gorm"
)

// Model 通用模型基类（uint 主键 + 软删除 + 时间戳）
//
// 设计说明：
// - 嵌入式结构体，符合 go-zero 风格
// - GORM v2 原生软删除，Delete() 自动转换为 UPDATE deleted_at
// - 无需任何插件配置
//
// 使用示例：
//
//	type User struct {
//	    gormx.Model
//	    Name  string
//	    Email string
//	}
type Model struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除，GORM 自动处理
}

// IntIDModel int 类型主键模型
//
// 使用场景：需要与其他系统对接时使用 int 类型 ID
type IntIDModel struct {
	ID        int            `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// StringIDModel string 类型主键模型
//
// 使用场景：需要字符串主键（如 UUID）
type StringIDModel struct {
	ID        string         `gorm:"primarykey;size:36" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TimeModel 仅时间戳模型（无软删除）
//
// 使用场景：不需要软删除的表
type TimeModel struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
