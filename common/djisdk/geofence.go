package djisdk

import (
	"encoding/json"
)

// ==================== 自定义飞行区 GeoJSON 工具 ====================
// 参考: https://developer.dji.com/doc/cloud-api-tutorial/cn/feature-set/dock-feature-set/custom-flight-area.html
// 坐标系: WGS84 (EPSG:4326)，坐标顺序 [经度, 纬度]。

// GeofenceFeatureCollection DJI 自定义飞行区 FeatureCollection。
type GeofenceFeatureCollection struct {
	Type     string            `json:"type"`
	Features []GeofenceFeature `json:"features"`
}

// GeofenceFeature DJI 自定义飞行区 Feature。
type GeofenceFeature struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	GeofenceType string             `json:"geofence_type"`
	Geometry     json.RawMessage    `json:"geometry"`
	Properties   GeofenceProperties `json:"properties"`
}

// GeofenceProperties 飞行区属性。
type GeofenceProperties struct {
	Radius  float64 `json:"radius"`
	SubType string  `json:"subType,omitempty"`
	Enable  bool    `json:"enable"`
}

// NewGeofenceFeatureCollection 创建 FeatureCollection。
func NewGeofenceFeatureCollection(features ...GeofenceFeature) GeofenceFeatureCollection {
	return GeofenceFeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}
}

// NewGeofencePolygonFeature 创建多边形几何的飞行区 Feature。
func NewGeofencePolygonFeature(id, geofenceType string, coordinates [][2]float64, enabled bool) GeofenceFeature {
	rings := [][][2]float64{coordinates}
	geom, _ := json.Marshal(map[string]any{
		"type":        "Polygon",
		"coordinates": rings,
	})
	return GeofenceFeature{
		ID:           id,
		Type:         "Feature",
		GeofenceType: geofenceType,
		Geometry:     geom,
		Properties: GeofenceProperties{
			Radius: 0,
			Enable: enabled,
		},
	}
}

// NewGeofenceCircleFeature 创建圆形几何的飞行区 Feature。
func NewGeofenceCircleFeature(id, geofenceType string, lng, lat, radius float64, enabled bool) GeofenceFeature {
	geom, _ := json.Marshal(map[string]any{
		"type":        "Point",
		"coordinates": [2]float64{lng, lat},
	})
	return GeofenceFeature{
		ID:           id,
		Type:         "Feature",
		GeofenceType: geofenceType,
		Geometry:     geom,
		Properties: GeofenceProperties{
			SubType: "Circle",
			Radius:  radius,
			Enable:  enabled,
		},
	}
}

// ToJSON 序列化 FeatureCollection。
func (fc *GeofenceFeatureCollection) ToJSON() ([]byte, error) {
	return json.Marshal(fc)
}
