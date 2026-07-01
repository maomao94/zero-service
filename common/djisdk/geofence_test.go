package djisdk

import (
	"encoding/json"
	"testing"
)

func TestGeofencePolygon_JSON(t *testing.T) {
	coords := [][2]float64{
		{120.5, 30.2}, {121.5, 30.2}, {121.5, 31.2}, {120.5, 31.2}, {120.5, 30.2},
	}

	f := NewGeofencePolygonFeature("poly-1", "dfence", coords, true)
	fc := NewGeofenceFeatureCollection(f)
	raw, err := fc.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var parsed struct {
		Features []struct {
			ID           string `json:"id"`
			GeofenceType string `json:"geofence_type"`
			Geometry     struct {
				Type        string        `json:"type"`
				Coordinates [][][2]float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				Radius float64 `json:"radius"`
				Enable bool    `json:"enable"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	f0 := parsed.Features[0]
	if f0.GeofenceType != "dfence" {
		t.Errorf("geofence_type = %q", f0.GeofenceType)
	}
	if f0.Geometry.Type != "Polygon" {
		t.Errorf("geometry.type = %q", f0.Geometry.Type)
	}
	if f0.Properties.Radius != 0 {
		t.Errorf("radius = %v", f0.Properties.Radius)
	}
}

func TestGeofenceCircle_JSON(t *testing.T) {
	f := NewGeofenceCircleFeature("nfz-1", "nfz", 120.5, 30.2, 1000, true)
	fc := NewGeofenceFeatureCollection(f)
	raw, _ := fc.ToJSON()

	var parsed struct {
		Features []struct {
			GeofenceType string `json:"geofence_type"`
			Geometry     struct {
				Type        string    `json:"type"`
				Coordinates [2]float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				Radius  float64 `json:"radius"`
				SubType string  `json:"subType"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	f0 := parsed.Features[0]
	if f0.GeofenceType != "nfz" {
		t.Errorf("geofence_type = %q", f0.GeofenceType)
	}
	if f0.Geometry.Type != "Point" {
		t.Errorf("geometry.type = %q", f0.Geometry.Type)
	}
	if f0.Properties.Radius != 1000 {
		t.Errorf("radius = %v", f0.Properties.Radius)
	}
	if f0.Properties.SubType != "Circle" {
		t.Errorf("subType = %q", f0.Properties.SubType)
	}
}

func TestGeofenceAllCombinations(t *testing.T) {
	feats := []GeofenceFeature{
		NewGeofencePolygonFeature("d1", "dfence", [][2]float64{{120, 30}, {121, 30}, {121, 31}, {120, 31}, {120, 30}}, true),
		NewGeofenceCircleFeature("d2", "dfence", 120.5, 30.5, 500, true),
		NewGeofencePolygonFeature("n1", "nfz", [][2]float64{{121, 31}, {122, 31}, {122, 32}, {121, 32}, {121, 31}}, true),
		NewGeofenceCircleFeature("n2", "nfz", 121.5, 31.5, 200, true),
	}

	fc := NewGeofenceFeatureCollection(feats...)
	raw, err := fc.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var parsed GeofenceFeatureCollection
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed.Features) != 4 {
		t.Fatalf("features count = %d", len(parsed.Features))
	}
	for i, f := range parsed.Features {
		if f.ID != feats[i].ID {
			t.Errorf("feature[%d] id = %q, want %q", i, f.ID, feats[i].ID)
		}
		if f.GeofenceType != feats[i].GeofenceType {
			t.Errorf("feature[%d] geofence_type = %q, want %q", i, f.GeofenceType, feats[i].GeofenceType)
		}
	}

	// dfence polygon: coordinates 是 Polygon
	for _, f := range parsed.Features {
		var geom struct {
			Type string `json:"type"`
		}
		json.Unmarshal(f.Geometry, &geom)
		if f.ID == "d1" || f.ID == "n1" {
			if geom.Type != "Polygon" {
				t.Errorf("%s geometry.type = %q, want Polygon", f.ID, geom.Type)
			}
		}
		if f.ID == "d2" || f.ID == "n2" {
			if geom.Type != "Point" {
				t.Errorf("%s geometry.type = %q, want Point", f.ID, geom.Type)
			}
		}
	}
}
