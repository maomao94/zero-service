import type { MethodDefinition, MethodField } from '../types'

type Group = 'gis-grid' | 'gis-analysis' | 'gis-fence'

const point = { lon: 120.15, lat: 30.28 }
const nearby = { lon: 120.16, lat: 30.29 }
const polygon = { outer: { points: [point, { lon: 120.16, lat: 30.28 }, nearby, { lon: 120.15, lat: 30.29 }] } }
const h3Index = '8928308280fffff'
const fenceId = 'test-fence-id'

const field = (name: string, type: string, note: string): MethodField => ({ name, type, note })
const pointField = field('point', 'Point', '经纬度对象 {lon, lat}，单位为度')
const pointsField = field('points', 'Point[]', '经纬度点集合')
const polygonField = field('polygon', 'Polygon', 'outer.points 定义外环，holes 可定义排除区域')
const resolution = field('resolution', 'uint32', 'H3 分辨率 0–15，默认 9')
const precision = field('precision', 'uint32', 'Geohash 精度 1–12，默认 7')
const index = field('h3Index', 'string', '十六进制 H3 索引')
const ring = field('k', 'uint32', '邻域圈数，0 仅返回中心格')
const fence = field('fence', 'Fence', '传 polygon 直接计算，或传 fenceId 查询已保存围栏')
const id = field('fenceId', 'string', '围栏 ID')
const name = field('name', 'string', '围栏名称')
const coordFields = [field('sourceType', 'CoordType', '源坐标系'), field('targetType', 'CoordType', '目标坐标系')]
const coord = { sourceType: 'COORD_TYPE_WGS84', targetType: 'COORD_TYPE_GCJ02' }
const fenceOptions = [field('h3Resolution', 'uint32', 'H3 分辨率，默认 9'), field('geohashPrecision', 'uint32', 'Geohash 精度，默认 7')]
const coverage = [field('h3Cells', 'string[]', '围栏覆盖的 H3 格网'), field('geohashes', 'string[]', '围栏覆盖的 Geohash 格网')]
const grid = [field('origin', 'string', '中心 H3 索引'), field('cells', 'GridDiskCell[]', '按圈数递增的格网列表')]

function method(category: Group, rpc: string, label: string, example: object, fields: MethodField[], responseFields: MethodField[], write = false): MethodDefinition {
  return {
    category, rpc: `gis.Gis/${rpc}`, key: `gis.${rpc[0].toLowerCase()}${rpc.slice(1)}`,
    label, description: label, example: JSON.stringify(example, null, 2), fields, responseFields, write, tips: [],
  }
}

export const GIS_METHODS: MethodDefinition[] = [
  method('gis-grid', 'EncodeGeoHash', '编码 Geohash', { point, precision: 7 }, [pointField, precision], [field('geohash', 'string', '格网编码')]),
  method('gis-grid', 'EncodeGeoHashMulti', '多精度 Geohash 编码', { point, precisions: [5, 7] }, [pointField, field('precisions', 'uint32[]', '精度列表，每项 1–12')], [field('geohashes', 'GeoHashIndex[]', '按输入顺序返回编码')]),
  method('gis-grid', 'DecodeGeoHash', '解码 Geohash', { geohash: 'wtmknz2' }, [field('geohash', 'string', '待解码的格网编码')], [field('point', 'Point', '格网中心'), precision, ...['lonMin', 'lonMax', 'latMin', 'latMax'].map((key) => field(key, 'double', '格网边界坐标'))]),
  method('gis-grid', 'EncodeH3', '编码 H3', { point, resolution: 9 }, [pointField, resolution], [index]),
  method('gis-grid', 'EncodeH3Multi', '多分辨率 H3 编码', { point, resolutions: [7, 9] }, [pointField, field('resolutions', 'uint32[]', '分辨率列表，每项 0–15')], [field('h3Indexes', 'H3Index[]', '按输入顺序返回索引')]),
  method('gis-grid', 'DecodeH3', '解码 H3', { h3Index }, [index], [field('center', 'Point', '格网中心'), field('boundary', 'Point[]', '格网边界顶点'), resolution]),
  method('gis-grid', 'GridDisk', '按 H3 索引查邻域', { h3Index, k: 1 }, [index, ring], grid),
  method('gis-grid', 'GridDiskByPoint', '按坐标查 H3 邻域', { point, resolution: 9, k: 1 }, [pointField, resolution, ring], grid),
  method('gis-grid', 'GenerateFenceCells', '计算围栏 Geohash 格网', { polygon, precision: 6 }, [polygonField, precision, field('includeNeighbors', 'bool', '是否包含相邻格网')], [coverage[1]]),
  method('gis-grid', 'GenerateFenceH3Cells', '计算围栏 H3 格网', { polygon, resolution: 8 }, [polygonField, resolution], [field('h3Indexes', 'string[]', '围栏覆盖的 H3 索引')]),
  method('gis-analysis', 'PointsWithinRadius', '查询半径内的点', { center: point, points: [nearby], radiusMeters: 2000 }, [field('center', 'Point', '查询中心'), pointsField, field('radiusMeters', 'double', '查询半径，单位为米')], [field('hits', 'RadiusHit[]', '命中的输入下标与距离')]),
  method('gis-analysis', 'PointInFence', '判断是否命中围栏', { point, fence: { polygon } }, [pointField, fence], [field('hit', 'bool', '点是否在围栏内')]),
  method('gis-analysis', 'PointInFences', '批量判断围栏命中', { point, fences: [{ fenceId, polygon }] }, [pointField, field('fences', 'Fence[]', '待判断的围栏列表')], [field('hitFenceIds', 'string[]', '命中的围栏 ID')]),
  method('gis-analysis', 'Distance', '计算两点距离', { a: point, b: nearby }, [field('a', 'Point', '点 A'), field('b', 'Point', '点 B')], [field('meters', 'double', '球面距离，单位为米')]),
  method('gis-analysis', 'BatchDistance', '批量计算距离', { pairs: [{ a: point, b: nearby }] }, [field('pairs', 'PointPair[]', '待计算的点对列表')], [field('meters', 'double[]', '按输入顺序返回距离，单位为米')]),
  method('gis-analysis', 'NearbyFences', '查询附近围栏', { point, km: 1 }, [pointField, field('km', 'double', '搜索半径，单位为公里')], [field('fenceIds', 'string[]', '格网粗筛得到的围栏 ID')]),
  method('gis-analysis', 'TransformCoord', '转换坐标系', { point, ...coord }, [pointField, ...coordFields], [field('transformedPoint', 'Point', '转换后的点')]),
  method('gis-analysis', 'BatchTransformCoord', '批量转换坐标系', { points: [point, nearby], ...coord }, [pointsField, ...coordFields], [field('transformedPoints', 'Point[]', '按输入顺序返回转换结果')]),
  method('gis-analysis', 'RoutePoints', '计算途经点访问顺序', { start: point, points: [nearby] }, [field('start', 'Point', '起点'), pointsField], [field('visitOrder', 'int32[]', '访问顺序对应输入下标'), field('totalDistanceMeters', 'double', '总路程，单位为米')]),
  method('gis-fence', 'CreateFence', '新增电子围栏', { name: '测试围栏', polygon }, [name, polygonField, ...fenceOptions], [id, ...coverage], true),
  method('gis-fence', 'UpdateFence', '更新电子围栏', { fenceId, name: '测试围栏', polygon }, [id, name, polygonField, ...fenceOptions], coverage, true),
  method('gis-fence', 'DeleteFence', '删除电子围栏', { fenceId }, [id], [], true),
  method('gis-fence', 'ListFences', '分页查询电子围栏', { page: '1', pageSize: '20' }, [field('page', 'int64', '页码，从 1 开始'), field('pageSize', 'int64', '每页条数，默认 20，最大 200'), field('name', 'string', '按名称模糊搜索')], [field('list', 'FenceDetail[]', '当前页围栏列表'), field('total', 'int64', '总条数')]),
  method('gis-fence', 'GetFence', '查询电子围栏详情', { fenceId }, [id], [field('fence', 'FenceDetail', '围栏详情')]),
]
