package logic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/djisdk"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitCustomFlyRegionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSubmitCustomFlyRegionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitCustomFlyRegionLogic {
	return &SubmitCustomFlyRegionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SubmitCustomFlyRegion 提交自定义飞行区（新增/更新）。
// 将 protobuf 的结构化几何参数转换为 DJI GeoJSON，上传至 OSS 对象存储，
// 写入飞行区配置记录后通过 MQTT 触发目标设备更新。
func (l *SubmitCustomFlyRegionLogic) SubmitCustomFlyRegion(in *djicloud.SubmitCustomFlyRegionReq) (*djicloud.SubmitCustomFlyRegionRes, error) {
	if len(in.GetFeatures()) == 0 {
		return nil, errors.New("至少需要一个 feature")
	}

	fileId := uuid.NewString()
	geofenceType := in.GetFeatures()[0].GetGeofenceType()
	if geofenceType == "" {
		geofenceType = "geofence"
	}

	var features []djisdk.GeofenceFeature
	for _, ft := range in.GetFeatures() {
		id := ft.GetId()
		if id == "" {
			id = uuid.NewString()
		}
		switch g := ft.Geometry.(type) {
		case *djicloud.FlyRegionFeature_Polygon:
			coords := make([][2]float64, len(g.Polygon.GetCoordinates()))
			for i, c := range g.Polygon.GetCoordinates() {
				coords[i] = [2]float64{c.GetLng(), c.GetLat()}
			}
			features = append(features, djisdk.NewGeofencePolygonFeature(id, ft.GetGeofenceType(), coords, ft.GetEnable()))
		case *djicloud.FlyRegionFeature_Circle:
			c := g.Circle.GetCenter()
			features = append(features, djisdk.NewGeofenceCircleFeature(id, ft.GetGeofenceType(), c.GetLng(), c.GetLat(), g.Circle.GetRadius(), ft.GetEnable()))
		default:
			return nil, fmt.Errorf("缺少 geometry")
		}
	}

	if l.svcCtx.OssTemplate == nil {
		return nil, errors.New("OSS 未配置")
	}

	fc := djisdk.NewGeofenceFeatureCollection(features...)
	jsonBytes, err := fc.ToJSON()
	if err != nil {
		return nil, fmt.Errorf("生成 GeoJSON 失败: %w", err)
	}

	fileName := geofenceType + "_" + fileId + ".json"
	gatewaySn := in.GetDeviceSn()
	bucketName := l.svcCtx.Config.Oss.BucketName

	ossFile, err := l.svcCtx.OssTemplate.PutObject(l.ctx, "", bucketName, fileName, "application/json", bytes.NewReader(jsonBytes), int64(len(jsonBytes)))
	if err != nil {
		return nil, fmt.Errorf("上传 OSS 失败: %w", err)
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(jsonBytes))

	region := &gormmodel.DjiFlyRegion{
		GatewaySn:    gatewaySn,
		Name:         in.GetName(),
		FileId:       fileId,
		BucketName:   bucketName,
		FileName:     ossFile.Name,
		FileSize:     ossFile.Size,
		Checksum:     checksum,
		GeofenceJSON: string(jsonBytes),
	}
	if err := l.svcCtx.DB.WithContext(l.ctx).Create(region).Error; err != nil {
		return nil, fmt.Errorf("写入飞行区记录失败: %w", err)
	}

	tid, err := l.svcCtx.DjiClient.FlightAreasUpdate(l.ctx, gatewaySn)
	if err != nil {
		return &djicloud.SubmitCustomFlyRegionRes{Code: -1, Message: err.Error(), Tid: tid, FileId: fileId}, nil
	}

	return &djicloud.SubmitCustomFlyRegionRes{Code: 0, Message: "success", Tid: tid, FileId: fileId}, nil
}
