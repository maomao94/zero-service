package dispatch

import "zero-service/app/gis/gis"

// GIS 网关方法。
const (
	MethodGISEncodeGeoHash        = "gis.encodeGeoHash"
	MethodGISEncodeGeoHashMulti   = "gis.encodeGeoHashMulti"
	MethodGISDecodeGeoHash        = "gis.decodeGeoHash"
	MethodGISEncodeH3             = "gis.encodeH3"
	MethodGISEncodeH3Multi        = "gis.encodeH3Multi"
	MethodGISDecodeH3             = "gis.decodeH3"
	MethodGISGridDisk             = "gis.gridDisk"
	MethodGISGridDiskByPoint      = "gis.gridDiskByPoint"
	MethodGISGenerateFenceCells   = "gis.generateFenceCells"
	MethodGISGenerateFenceH3Cells = "gis.generateFenceH3Cells"
	MethodGISPointsWithinRadius   = "gis.pointsWithinRadius"
	MethodGISPointInFence         = "gis.pointInFence"
	MethodGISPointInFences        = "gis.pointInFences"
	MethodGISDistance             = "gis.distance"
	MethodGISBatchDistance        = "gis.batchDistance"
	MethodGISNearbyFences         = "gis.nearbyFences"
	MethodGISTransformCoord       = "gis.transformCoord"
	MethodGISBatchTransformCoord  = "gis.batchTransformCoord"
	MethodGISRoutePoints          = "gis.routePoints"
	MethodGISCreateFence          = "gis.createFence"
	MethodGISUpdateFence          = "gis.updateFence"
	MethodGISDeleteFence          = "gis.deleteFence"
	MethodGISListFences           = "gis.listFences"
	MethodGISGetFence             = "gis.getFence"
)

func gisRoutes(client gis.GisClient) map[string]Handler {
	return map[string]Handler{
		MethodGISEncodeGeoHash:        relay(client.EncodeGeoHash),
		MethodGISEncodeGeoHashMulti:   relay(client.EncodeGeoHashMulti),
		MethodGISDecodeGeoHash:        relay(client.DecodeGeoHash),
		MethodGISEncodeH3:             relay(client.EncodeH3),
		MethodGISEncodeH3Multi:        relay(client.EncodeH3Multi),
		MethodGISDecodeH3:             relay(client.DecodeH3),
		MethodGISGridDisk:             relay(client.GridDisk),
		MethodGISGridDiskByPoint:      relay(client.GridDiskByPoint),
		MethodGISGenerateFenceCells:   relay(client.GenerateFenceCells),
		MethodGISGenerateFenceH3Cells: relay(client.GenerateFenceH3Cells),
		MethodGISPointsWithinRadius:   relay(client.PointsWithinRadius),
		MethodGISPointInFence:         relay(client.PointInFence),
		MethodGISPointInFences:        relay(client.PointInFences),
		MethodGISDistance:             relay(client.Distance),
		MethodGISBatchDistance:        relay(client.BatchDistance),
		MethodGISNearbyFences:         relay(client.NearbyFences),
		MethodGISTransformCoord:       relay(client.TransformCoord),
		MethodGISBatchTransformCoord:  relay(client.BatchTransformCoord),
		MethodGISRoutePoints:          relay(client.RoutePoints),
		MethodGISCreateFence:          relay(client.CreateFence),
		MethodGISUpdateFence:          relay(client.UpdateFence),
		MethodGISDeleteFence:          relay(client.DeleteFence),
		MethodGISListFences:           relay(client.ListFences),
		MethodGISGetFence:             relay(client.GetFence),
	}
}
