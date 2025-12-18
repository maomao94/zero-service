package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
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
			return "-"
		}

	// 其他类型：返回反射类型名称
	default:
		return reflect.TypeOf(value).String()
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

	insertedCount, err := mr.MapReduce(
		// generate
		func(source chan<- string) {
			for _, msgBody := range in.MsgBody {
				var bodyMap map[string]interface{}
				if err := json.Unmarshal([]byte(msgBody.BodyRaw), &bodyMap); err != nil {
					l.WithContext(ctx).Errorf("Failed to parse bodyRaw: %v, msgId: %s", err, msgBody.MsgId)
					continue
				}

				ioa, err := convertor.ToInt(bodyMap["ioa"])
				if err != nil {
					l.WithContext(ctx).Errorf("Failed to get ioa from bodyRaw, msgId: %s", msgBody.MsgId)
					continue
				}
				ioaValueStr := extractIoaValue(bodyMap)
				// 生成 stationId
				safeHost := strings.ReplaceAll(msgBody.Host, ".", "_")
				stationId := fmt.Sprintf("%s_%s", safeHost, msgBody.Port)
				deviceTableName := fmt.Sprintf("raw_%s", stationId)
				if len(msgBody.MetaDataRaw) > 0 {
					var metaDataMap map[string]interface{}
					err = json.Unmarshal([]byte(msgBody.MetaDataRaw), &metaDataMap)
					if err == nil {
						if sid, ok := metaDataMap["stationId"].(string); ok && sid != "" {
							stationId = sid
							deviceTableName = fmt.Sprintf("raw_%s", stationId)
						}
					}
				}

				// 查询本地缓存表
				query, ok, err := l.svcCtx.FindOneByTagStationCoaIoa(ctx, stationId, int64(msgBody.Coa), ioa)
				if err != nil {
					l.WithContext(ctx).Errorf("Failed to find point mapping: %v, msgId: %s", err, msgBody.MsgId)
					continue
				}
				if ok && query.EnableRawInsert == 1 {
					// 构建 TDengine 插入语句
					insertSQL := fmt.Sprintf(
						"INSERT INTO iec104.%s USING iec104.raw_point_data "+
							"TAGS ('%s', %d, %d) "+
							"VALUES ('%s', '%s', '%s', %d, '%s', %d, %d, %d, %d, '%s', '%s')",
						deviceTableName,  // 子表名
						stationId,        // tag_station
						msgBody.Coa,      // coa
						ioa,              // ioa
						msgBody.Time,     // ts
						msgBody.MsgId,    // msg_id
						msgBody.Host,     // host_v
						msgBody.Port,     // port_v
						msgBody.Asdu,     // asdu
						msgBody.TypeId,   // type_id
						msgBody.DataType, // data_type
						msgBody.Coa,      // coa
						ioa,              // ioa
						ioaValueStr,      // ioa_value
						msgBody.BodyRaw,  // raw_msg
					)
					source <- insertSQL
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
	}

	duration := timex.Since(startTime)
	l.WithContext(ctx).WithDuration(duration).Infof("PushChunkAsdu, received %d asdu, dispatch inserted %d rows", len(in.MsgBody), insertedCount)
	return &streamevent.PushChunkAsduRes{}, nil
}
