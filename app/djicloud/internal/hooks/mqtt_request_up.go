package hooks

import (
	"context"
	"fmt"
	"time"

	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"
	"zero-service/common/gormx"
	"zero-service/common/ossx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

// NewDeviceRequestHandler 构造 thing/product/{gateway_sn}/requests 上行处理器。
//
// requests 是设备向云端拉取平台侧配置或状态的通道，必须按 method 返回匹配的 output：
// airport_organization_get 返回组织信息占位；airport_bind_status 返回绑定状态；flight_areas_get 返回自定义飞行区列表。
func NewDeviceRequestHandler(db *gormx.DB, ossTemplate ossx.OssTemplate) djisdk.RequestHandler {
	return func(ctx context.Context, gatewaySn string, req *djisdk.RequestMessage) (any, error) {
		if req == nil {
			logx.WithContext(ctx).Errorf("[dji-cloud] request: nil request payload, sn=%s", gatewaySn)
			return nil, &djisdk.PlatformError{Code: djisdk.PlatformResultHandlerError, Err: fmt.Errorf("nil request payload for sn=%s", gatewaySn)}
		}
		logx.WithContext(ctx).Infof("[dji-cloud] request: method=%s sn=%s tid=%s", req.Method, gatewaySn, req.Tid)
		switch req.Method {
		case djisdk.MethodAirportOrganizationGet:
			return nil, djisdk.ErrSkipRequestReply
		case djisdk.MethodAirportBindStatus:
			return nil, djisdk.ErrSkipRequestReply
		case djisdk.MethodFlightAreasGet:
			return buildFlightAreasReply(ctx, db, ossTemplate, gatewaySn)
		default:
			return nil, nil
		}
	}
}

func buildFlightAreasReply(ctx context.Context, db *gormx.DB, ossTemplate ossx.OssTemplate, gatewaySn string) (*djisdk.FlightAreasGetReplyData, error) {
	if db == nil {
		return &djisdk.FlightAreasGetReplyData{Files: []djisdk.FlightAreasFile{}}, nil
	}

	var regions []gormmodel.DjiFlyRegion
	if err := db.WithContext(ctx).Where("gateway_sn = ?", gatewaySn).Order("id DESC").Find(&regions).Error; err != nil {
		return &djisdk.FlightAreasGetReplyData{Files: []djisdk.FlightAreasFile{}}, nil
	}

	files := make([]djisdk.FlightAreasFile, len(regions))
	fns := make([]func() error, 0, len(regions))
	for i := range regions {
		i := i
		r := regions[i]
		fns = append(fns, func() error {
			fileURL := r.FileName
			if ossTemplate != nil && r.BucketName != "" {
				u, err := ossTemplate.SignUrl(ctx, "", r.BucketName, r.FileName, 7*24*time.Hour)
				if err != nil {
					logx.WithContext(ctx).Errorf("[dji-cloud] flight_areas_get: sign url failed: %v", err)
					return err
				} else {
					fileURL = u
				}
			}
			files[i] = djisdk.FlightAreasFile{
				Name:     r.FileName,
				URL:      fileURL,
				Size:     r.FileSize,
				Checksum: r.Checksum,
			}
			return nil
		})
	}
	err := mr.Finish(fns...)
	if err != nil {
		logx.WithContext(ctx).Errorf("[dji-cloud] flight_areas_get: build files failed: %v", err)
		return nil, err
	}

	return &djisdk.FlightAreasGetReplyData{Files: files}, nil
}
