package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushChunkAsduLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushChunkAsduLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushChunkAsduLogic {
	return &PushChunkAsduLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// extractIoaValue 根据bodyMap中的value类型提取合适的值
func extractIoaValue(bodyMap map[string]interface{}) string {
	// 获取value字段
	value, ok := bodyMap["value"]
	if !ok {
		// 如果没有value字段，返回空字符串
		return ""
	}

	// 处理不同类型的value
	switch v := value.(type) {
	// 简单类型：直接转换为字符串
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case string:
		return v

	// 复杂类型：对象类型
	case map[string]interface{}:
		// 尝试提取对象中的关键值
		if counterReading, ok := v["counterReading"].(float64); ok {
			// 累计量类型：返回计数器读数
			return strconv.FormatFloat(counterReading, 'f', -1, 64)
		} else if val, ok := v["val"].(float64); ok {
			// 步位置信息：返回val值
			return strconv.FormatFloat(val, 'f', -1, 64)
		} else if counterReading, ok := v["counterReading"].(float64); ok {
			// 累计量类型：返回计数器读数
			return strconv.FormatFloat(counterReading, 'f', -1, 64)
		} else {
			// 其他对象类型：返回JSON字符串
			if jsonStr, err := json.Marshal(v); err == nil {
				return string(jsonStr)
			}
		}

	// 其他类型：返回反射类型名称
	default:
		return reflect.TypeOf(value).String()
	}

	// 默认返回空字符串
	return ""
}

func (l *PushChunkAsduLogic) PushChunkAsdu(in *streamevent.PushChunkAsduReq) (*streamevent.PushChunkAsduRes, error) {
	// 检查TDengine连接是否可用
	if l.svcCtx.TaoDB == nil {
		l.Errorf("TDengine connection is not initialized")
		return &streamevent.PushChunkAsduRes{}, nil
	}

	// 所有数据类型都入库，不做过滤
	insertedCount := 0

	// 遍历所有消息体
	for _, msgBody := range in.MsgBody {
		// 解析bodyRaw获取ioa和value字段
		var bodyMap map[string]interface{}
		if err := json.Unmarshal([]byte(msgBody.BodyRaw), &bodyMap); err != nil {
			l.Errorf("Failed to parse bodyRaw: %v, msgId: %s", err, msgBody.MsgId)
			continue
		}

		// 获取ioa字段
		ioaValue, ok := bodyMap["ioa"].(float64)
		if !ok {
			l.Errorf("Failed to get ioa from bodyRaw, msgId: %s", msgBody.MsgId)
			continue
		}
		ioa := uint32(ioaValue)

		// 处理ioa_value字段：根据value类型提取合适的值
		ioaValueStr := extractIoaValue(bodyMap)

		// 1. 移除预处理语句，使用直接执行SQL的方式
		// 2. 为每个设备生成唯一子表名，避免"Table already exists in other stables"错误
		// 3. 将IP地址中的点号替换为下划线，避免TDengine语法错误
		// 格式：device_${host_with_underscores}_${port}_${coa}_${ioa}
		safeHost := strings.ReplaceAll(msgBody.Host, ".", "_")
		deviceTableName := fmt.Sprintf("device_%s_%d_%d_%d", safeHost, msgBody.Port, msgBody.Coa, ioa)

		// 构建正确的TDengine插入语句
		// raw_msg字段已设置为5000长度，可以存储完整的JSON数据
		insertSQL := fmt.Sprintf(
			"INSERT INTO iec104.%s USING iec104.raw_point_data "+
				"TAGS ('%s', %d, %d, %d) "+
				"VALUES ('%s', '%s', '%s', %d, %d, '%s', '%s')",
			deviceTableName,  // 子表名
			msgBody.Host,     // tag_host
			msgBody.Port,     // tag_port
			msgBody.Coa,      // coa
			ioa,              // ioa
			msgBody.Time,     // ts
			msgBody.MsgId,    // msg_id
			msgBody.Asdu,     // asdu
			msgBody.TypeId,   // type_id
			msgBody.DataType, // data_type
			ioaValueStr,      // ioa_value (存储提取的value值)
			msgBody.BodyRaw,  // raw_msg (存储完整的JSON数据)
		)

		// 执行插入
		_, err := l.svcCtx.TaoDB.Exec(insertSQL)
		if err != nil {
			l.Errorf("Failed to insert data: %v, sql: %s, msgId: %s", err, insertSQL, msgBody.MsgId)
			continue
		}

		insertedCount++
	}

	l.Infof("Successfully pushed %d ASDU messages to TDengine", insertedCount)
	return &streamevent.PushChunkAsduRes{}, nil
}
