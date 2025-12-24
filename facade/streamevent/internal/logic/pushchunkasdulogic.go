package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"zero-service/common/iec104/util"
	"zero-service/common/tool"
	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
	"github.com/zeromicro/go-zero/core/timex"
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
	value, ok := bodyMap["value"]
	if !ok {
		// 如果没有value字段，返回空字符串
		return ""
	}

	// 处理nil值
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	// 简单类型：直接转换为字符串
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64)
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
		} else if value, ok := v["value"].(float64); ok {
			// 通用value字段
			return strconv.FormatFloat(value, 'f', -1, 64)
		} else if value, ok := v["value"].(string); ok {
			// 字符串类型value
			return value
		} else if value, ok := v["value"].(bool); ok {
			// 布尔类型value
			return strconv.FormatBool(value)
		} else {
			// 其他对象类型，返回JSON字符串
			if jsonStr, err := json.Marshal(v); err == nil {
				return string(jsonStr)
			}
			return "-"
		}

	// 切片类型：返回JSON字符串
	case []interface{}:
		if jsonStr, err := json.Marshal(v); err == nil {
			return string(jsonStr)
		}
		return fmt.Sprintf("[%d]items", len(v))

	// 其他类型
	default:
		return fmt.Sprintf("%v", v)
	}
	return "-"
}

func (l *PushChunkAsduLogic) PushChunkAsdu(in *streamevent.PushChunkAsduReq) (*streamevent.PushChunkAsduRes, error) {
	startTime := timex.Now()
	reqId := tool.GenMicroTS()
	ctx := logx.WithFields(context.WithValue(l.ctx, "taos_req_id", reqId), logx.Field("taosReqId", reqId))
	if l.svcCtx.TaosConn == nil {
		l.WithContext(ctx).Errorf("TDengine connection is not initialized")
		return &streamevent.PushChunkAsduRes{}, nil
	}
	var ignoreCount = 0
	insertedCount, err := mr.MapReduce(
		// generate
		func(source chan<- string) {
			for _, msgBody := range in.MsgBody {
				var bodyMap map[string]interface{}
				if err := json.Unmarshal([]byte(msgBody.BodyRaw), &bodyMap); err != nil {
					l.WithContext(ctx).Errorf("Failed to parse bodyRaw: %v, msgId: %s", err, msgBody.MsgId)
					ignoreCount++
					continue
				}

				ioa, err := convertor.ToInt(bodyMap["ioa"])
				if err != nil {
					l.WithContext(ctx).Errorf("Failed to get ioa from bodyRaw, msgId: %s", msgBody.MsgId)
					ignoreCount++
					continue
				}
				ioaValueStr := extractIoaValue(bodyMap)
				// 生成 stationId
				stationId := util.GenerateStationId(msgBody.Host, msgBody.Port)
				deviceTableName := fmt.Sprintf("raw_%s_%d_%d", stationId, msgBody.Coa, ioa)
				if len(msgBody.MetaDataRaw) > 0 {
					var metaDataMap map[string]interface{}
					err = json.Unmarshal([]byte(msgBody.MetaDataRaw), &metaDataMap)
					if err != nil {
						l.WithContext(ctx).Errorf("Failed to parse metaDataRaw: %v, msgId: %s", err, msgBody.MsgId)
					} else {
						if sid, ok := metaDataMap["stationId"].(string); ok && sid != "" {
							stationId = sid
							deviceTableName = fmt.Sprintf("raw_%s_%d_%d", stationId, msgBody.Coa, ioa)
						}
					}
				}
				// 查询本地缓存表
				query, ok, err := l.svcCtx.DevicePointMappingModel.FindCacheOneByTagStationCoaIoa(ctx, stationId, int64(msgBody.Coa), ioa)
				if err != nil {
					l.WithContext(ctx).Errorf("Failed to find point mapping: %v, msgId: %s", err, msgBody.MsgId)
					continue
				}
				if ok && query.EnableRawInsert == 1 {
					insertSQL := fmt.Sprintf(
						"INSERT INTO %s.%s USING %s.raw_point_data "+
							"TAGS ('%s', %d, %d) "+
							"VALUES ('%s', '%s', '%s', %d, '%s', %d, %d, %d, %d, '%s', '%s')",
						l.svcCtx.Config.TaosDB.DBName,
						deviceTableName, // 子表名
						l.svcCtx.Config.TaosDB.DBName,
						strings.ReplaceAll(stationId, "'", "''"), // tag_station
						msgBody.Coa,                              // coa
						ioa,                                      // ioa
						strings.ReplaceAll(msgBody.Time, "'", "''"),  // ts
						strings.ReplaceAll(msgBody.MsgId, "'", "''"), // msg_id
						strings.ReplaceAll(msgBody.Host, "'", "''"),  // host_v
						msgBody.Port, // port_v
						strings.ReplaceAll(msgBody.Asdu, "'", "''"), // asdu
						msgBody.TypeId,   // type_id
						msgBody.DataType, // data_type
						msgBody.Coa,      // coa
						ioa,              // ioa
						strings.ReplaceAll(ioaValueStr, "'", "''"),     // ioa_value
						strings.ReplaceAll(msgBody.BodyRaw, "'", "''")) // raw_msg
					source <- insertSQL
				} else {
					ignoreCount++
				}
			}
		},
		// Map
		func(s string, writer mr.Writer[int], cancel func(error)) {
			_, err := l.svcCtx.TaosConn.ExecCtx(ctx, s)
			if err != nil {
				l.WithContext(ctx).Errorf("Failed to insert into TDengine: %v", err)
				writer.Write(0) // 插入失败，计数 0
				return
			}
			writer.Write(1) // 插入成功，计数 1
		},
		// Reduce
		func(pipe <-chan int, writer mr.Writer[int], cancel func(error)) {
			total := 0
			for count := range pipe {
				total += count
			}
			writer.Write(total)
		},
	)

	if err != nil {
		l.WithContext(ctx).Errorf("MapReduce failed: %v", err)
		return nil, err
	}

	duration := timex.Since(startTime)
	l.WithContext(ctx).WithDuration(duration).Infof("PushChunkAsdu, tId: %s, received %d asdu, ignored %d asdu, dispatch inserted %d rows", in.TId, len(in.MsgBody), ignoreCount, insertedCount)
	return &streamevent.PushChunkAsduRes{}, nil
}
