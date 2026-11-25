import type { MethodDefinition, MethodField } from '../types'
import { GIS_METHODS } from './gisMethods'

function json(value: unknown): string {
  return JSON.stringify(value, null, 2)
}

const modbusCodeField: MethodField = {
  name: 'modbusCode',
  type: 'string',
  note: 'Modbus 配置唯一编码；留空使用 bridgemodbus 默认文件配置，非空使用该编码对应的自定义配置',
}

function modbusAddressField(note: string): MethodField {
  return { name: 'address', type: 'uint32', required: true, note }
}

function modbusQuantityField(note: string): MethodField {
  return { name: 'quantity', type: 'uint32', required: true, note }
}

const DECIMAL_UNSIGNED_NOTE = 'true=无符号（0–65535），false=有符号（-32768–32767）；缺省 false'

const BYTES_RESPONSE: MethodField[] = [
  { name: 'results', type: 'bytes', note: '协议结果或回显字节；protojson 中为 base64 字符串' },
]

const REGISTER_RESPONSE: MethodField[] = [
  ...BYTES_RESPONSE,
  { name: 'uintValues', type: 'uint32[]', note: '每个寄存器的无符号视图' },
  { name: 'intValues', type: 'int32[]', note: '每个寄存器的有符号视图' },
  { name: 'hexValues', type: 'string[]', note: '每个寄存器的十六进制表示' },
  { name: 'binaryValues', type: 'string[]', note: '每个寄存器的 16 位二进制表示' },
]

const MODBUS_TIPS = [
  'modbusCode 留空时使用 bridgemodbus 默认文件配置，非空时使用该编码对应的自定义配置。',
  '地址范围 0–65535；读线圈/离散输入最多 2000，读寄存器最多 125，写多个线圈最多 1968，写多个寄存器最多 123，读写多个寄存器的写入段最多 121。',
  '响应中的 results 等 bytes 字段在 protojson 中为 base64 字符串，不是十六进制。',
  '参数越界统一返回 100022；写操作超时或断链时设备可能已执行，需先核实设备状态。',
]

/**
 * 与 app/openapi/internal/dispatch 的 method 注册表同步。
 */
export const METHODS: MethodDefinition[] = [
  // ---------------- Podman ----------------
  {
    key: 'pod.list',
    label: '容器列表',
    category: 'podman',
    rpc: 'podengine.PodEngine/ListPods',
    description: '查询节点上的容器列表，支持按名称、ID、标签过滤与 offset/limit 分页。',
    example: json({ node: 'local', limit: 20, offset: 0 }),
    fields: [
      { name: 'node', type: 'string', note: 'Docker 节点名，默认 local' },
      { name: 'limit', type: 'int32', note: '单页数量，最大 1000' },
      { name: 'offset', type: 'int32', note: '偏移量，最大 1000' },
      { name: 'names', type: 'string[]', note: '按容器名过滤' },
      { name: 'ids', type: 'string[]', note: '按容器 ID 过滤' },
      { name: 'labels', type: 'map<string,string>', note: '按标签过滤' },
    ],
    responseFields: [
      { name: 'items', type: 'ListPodItemPb[]', note: '当前页容器列表' },
      { name: 'total', type: 'int32', note: '符合筛选条件的容器总数' },
    ],
    tips: ['这是只读方法，不会修改容器状态。', 'limit/offset 在 protojson 中传数字或数字字符串均可。'],
  },
  {
    key: 'pod.get',
    label: '容器详情',
    category: 'podman',
    rpc: 'podengine.PodEngine/GetPod',
    description: '按容器 ID 查询 inspect 详情，包含阶段、条件、容器状态与挂载信息。',
    example: json({ node: 'local', id: 'mock-pod-1' }),
    fields: [
      { name: 'node', type: 'string', note: 'Docker 节点名，默认 local' },
      { name: 'id', type: 'string', required: true, note: '容器 ID' },
    ],
    responseFields: [
      { name: 'pod', type: 'PodPb', note: '容器 inspect 详情' },
    ],
    tips: ['id 必填；可先用 pod.list 获取。'],
  },
  {
    key: 'pod.stats',
    label: '资源统计',
    category: 'podman',
    rpc: 'podengine.PodEngine/GetPodStats',
    description: '查询容器 CPU、内存、网络和存储统计，字段原样透传，不做裁剪。',
    example: json({ node: 'local', id: 'mock-pod-1' }),
    fields: [
      { name: 'node', type: 'string', note: 'Docker 节点名，默认 local' },
      { name: 'id', type: 'string', required: true, note: '容器 ID' },
    ],
    responseFields: [{ name: 'stats', type: 'ContainerStatsPb[]', note: '容器 CPU、内存、网络和存储统计' }],
    tips: ['统计值来自 Docker stats，容器停止时可能返回空数组。'],
  },
  {
    key: 'image.list',
    label: '镜像列表',
    category: 'podman',
    rpc: 'podengine.PodEngine/ListImages',
    description: '查询节点上的镜像列表，支持引用过滤、分页与 digest 展示。',
    example: json({ node: 'local', limit: 20, offset: 0, includeDigests: true }),
    fields: [
      { name: 'node', type: 'string', note: 'Docker 节点名，默认 local' },
      { name: 'limit', type: 'int32', note: '单页数量，最大 1000' },
      { name: 'offset', type: 'int32', note: '偏移量，最大 1000' },
      { name: 'references', type: 'string[]', note: '按镜像引用过滤，如 nginx:latest' },
      { name: 'includeDigests', type: 'bool', note: '是否返回 digest' },
    ],
    responseFields: [
      { name: 'items', type: 'ImagePb[]', note: '当前页镜像列表' },
      { name: 'total', type: 'int32', note: '符合筛选条件的镜像总数' },
    ],
    tips: ['镜像大小字段为 int64，响应中会序列化为字符串。'],
  },

  // ---------------- Modbus Read ----------------
  {
    key: 'modbus.readCoils',
    label: '读线圈',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadCoils',
    description: '读取线圈状态（功能码 0x01），返回按位结果与布尔数组。',
    example: json({ modbusCode: '', address: 0, quantity: 8 }),
    fields: [
      modbusCodeField,
      modbusAddressField('起始线圈地址，0–65535'),
      modbusQuantityField('读取数量，1–2000'),
    ],
    responseFields: [...BYTES_RESPONSE, { name: 'values', type: 'bool[]', note: '按地址顺序排列的线圈状态' }],
    tips: [...MODBUS_TIPS, 'values 与起始地址一一对应，results 的 bit0 对应起始地址。'],
  },
  {
    key: 'modbus.readDiscreteInputs',
    label: '读离散输入',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadDiscreteInputs',
    description: '读取离散输入状态（功能码 0x02），返回按位结果与布尔数组。',
    example: json({ modbusCode: '', address: 0, quantity: 8 }),
    fields: [
      modbusCodeField,
      modbusAddressField('起始离散输入地址，0–65535'),
      modbusQuantityField('读取数量，1–2000'),
    ],
    responseFields: [...BYTES_RESPONSE, { name: 'values', type: 'bool[]', note: '按地址顺序排列的离散输入状态' }],
    tips: [...MODBUS_TIPS, 'values 与起始地址一一对应，results 的 bit0 对应起始地址。'],
  },
  {
    key: 'modbus.readInputRegisters',
    label: '读输入寄存器',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadInputRegisters',
    responseFields: REGISTER_RESPONSE,
    description: '读取输入寄存器（功能码 0x04），返回原始字节及无符号、有符号、十六进制与二进制表示。',
    example: json({ modbusCode: '', address: 0, quantity: 4 }),
    fields: [
      modbusCodeField,
      modbusAddressField('起始寄存器地址，0–65535'),
      modbusQuantityField('读取数量，1–125'),
    ],
    tips: MODBUS_TIPS,
  },
  {
    key: 'modbus.readHoldingRegisters',
    label: '读保持寄存器',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadHoldingRegisters',
    responseFields: REGISTER_RESPONSE,
    description: '读取保持寄存器（功能码 0x03），返回原始字节及无符号、有符号、十六进制与二进制表示。',
    example: json({ modbusCode: '', address: 0, quantity: 4 }),
    fields: [
      modbusCodeField,
      modbusAddressField('起始寄存器地址，0–65535'),
      modbusQuantityField('读取数量，1–125'),
    ],
    tips: MODBUS_TIPS,
  },
  {
    key: 'modbus.readFIFOQueue',
    label: '读 FIFO 队列',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadFIFOQueue',
    description: '读取 FIFO 队列内容（功能码 0x18），队列长度由设备返回。',
    example: json({ modbusCode: '', address: 0 }),
    fields: [modbusCodeField, modbusAddressField('FIFO 队列寄存器地址，0–65535')],
    responseFields: BYTES_RESPONSE,
    tips: [...MODBUS_TIPS, '该入口只传队列寄存器地址，不传 quantity。'],
  },
  {
    key: 'modbus.readDeviceIdentification',
    label: '读设备标识',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadDeviceIdentification',
    description: '读取设备标识（功能码 0x2B / 0x0E），返回原始、十六进制与语义化映射。',
    example: json({ modbusCode: '', readDeviceIdCode: 1 }),
    fields: [
      modbusCodeField,
      {
        name: 'readDeviceIdCode',
        type: 'uint32',
        required: true,
        note: '读取类型：1=基本（0x01）、2=常规（0x02）、3=扩展（0x03）；传 4 或其它值返回 100022',
      },
    ],
    responseFields: [
      { name: 'results', type: 'map<uint32,string>', note: '十进制 Object ID 到对象值的映射' },
      { name: 'hexResults', type: 'map<string,string>', note: '十六进制 Object ID 到对象值的映射' },
      { name: 'semanticResults', type: 'map<string,string>', note: '语义化对象名称到对象值的映射' },
    ],
    tips: [...MODBUS_TIPS, '读取指定 Object ID 请使用 modbus.readDeviceIdentificationSpecificObject。'],
  },
  {
    key: 'modbus.readDeviceIdentificationSpecificObject',
    label: '读指定设备对象',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadDeviceIdentificationSpecificObject',
    description: '读取指定 Object ID 的设备标识（功能码 0x2B / 0x0E），例如厂商名称、产品代码、版本号。',
    example: json({ modbusCode: '', objectId: 0 }),
    fields: [
      modbusCodeField,
      {
        name: 'objectId',
        type: 'uint32',
        required: true,
        note: '对象 ID，0–255：0x00 厂商名 / 0x01 产品代码 / 0x02 版本 / 0x03 厂商 URL / 0x04 产品名 / 0x05 型号 / 0x06 应用名',
      },
    ],
    responseFields: [
      { name: 'results', type: 'map<uint32,string>', note: '十进制 Object ID 到对象值的映射' },
      { name: 'hexResults', type: 'map<string,string>', note: '十六进制 Object ID 到对象值的映射' },
      { name: 'semanticResults', type: 'map<string,string>', note: '语义化对象名称到对象值的映射' },
    ],
    tips: MODBUS_TIPS,
  },

  // ---------------- Modbus Write ----------------
  {
    key: 'modbus.writeSingleCoil',
    label: '写单个线圈',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteSingleCoil',
    description: '写单个线圈（功能码 0x05），true=ON，false=OFF。',
    example: json({ modbusCode: '', address: 0, value: true }),
    fields: [
      modbusCodeField,
      modbusAddressField('线圈地址，0–65535'),
      { name: 'value', type: 'bool', required: true, note: 'true=ON，false=OFF' },
    ],
    responseFields: BYTES_RESPONSE,
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeMultipleCoils',
    label: '写多个线圈',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteMultipleCoils',
    description: '写多个连续线圈（功能码 0x0F），values 从起始地址依次写入。',
    example: json({ modbusCode: '', address: 0, quantity: 4, values: [true, false, true, false] }),
    fields: [
      modbusCodeField,
      modbusAddressField('起始线圈地址，0–65535'),
      modbusQuantityField('写入数量，1–1968，且必须与 values 长度一致'),
      { name: 'values', type: 'bool[]', required: true, note: '每个元素对应一个线圈，false=OFF，true=ON' },
    ],
    responseFields: BYTES_RESPONSE,
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeSingleRegister',
    label: '写单个寄存器',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteSingleRegister',
    description: '写单个保持寄存器（功能码 0x06），value 为寄存器原始 16 位值。',
    example: json({ modbusCode: '', address: 0, value: 100 }),
    fields: [
      modbusCodeField,
      modbusAddressField('寄存器地址，0–65535'),
      { name: 'value', type: 'uint32', required: true, note: '寄存器原始值，0–65535（16 位）' },
    ],
    responseFields: BYTES_RESPONSE,
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeSingleRegisterWithDecimal',
    label: '写单个寄存器（十进制）',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteSingleRegisterWithDecimal',
    description: '按十进制数值写单个保持寄存器（功能码 0x06），由服务端换算为 16 位寄存器值。',
    example: json({ modbusCode: '', address: 0, value: -12, unsigned: false }),
    fields: [
      modbusCodeField,
      modbusAddressField('寄存器地址，0–65535'),
      {
        name: 'value',
        type: 'int32',
        required: true,
        note: '十进制数值：unsigned=false 时为 -32768–32767，unsigned=true 时为 0–65535',
      },
      { name: 'unsigned', type: 'bool', note: DECIMAL_UNSIGNED_NOTE },
    ],
    responseFields: BYTES_RESPONSE,
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeMultipleRegisters',
    label: '写多个寄存器',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteMultipleRegisters',
    description: '写多个保持寄存器（功能码 0x10），values 为原始 16 位值列表。',
    example: json({ modbusCode: '', address: 0, quantity: 2, values: [100, 200] }),
    fields: [
      modbusCodeField,
      modbusAddressField('起始寄存器地址，0–65535'),
      modbusQuantityField('写入数量，1–123，且必须与 values 长度一致'),
      { name: 'values', type: 'uint32[]', required: true, note: '每个值对应一个寄存器，取值 0–65535（16 位）' },
    ],
    responseFields: BYTES_RESPONSE,
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeMultipleRegistersWithDecimal',
    label: '写多个寄存器（十进制）',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteMultipleRegistersWithDecimal',
    description: '按十进制数值写多个保持寄存器（功能码 0x10），由服务端逐个换算为 16 位寄存器值。',
    example: json({ modbusCode: '', address: 0, quantity: 2, values: [-12, 34], unsigned: false }),
    fields: [
      modbusCodeField,
      modbusAddressField('起始寄存器地址，0–65535'),
      modbusQuantityField('写入数量，1–123，且必须与 values 长度一致'),
      {
        name: 'values',
        type: 'int32[]',
        required: true,
        note: '十进制数值列表：unsigned=false 时每个值 -32768–32767，unsigned=true 时每个值 0–65535',
      },
      { name: 'unsigned', type: 'bool', note: DECIMAL_UNSIGNED_NOTE },
    ],
    responseFields: BYTES_RESPONSE,
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.readWriteMultipleRegisters',
    label: '读写多个寄存器',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/ReadWriteMultipleRegisters',
    description: '在一个事务中先写后读保持寄存器（功能码 0x17），返回读取段数据。',
    example: json({
      modbusCode: '',
      readAddress: 0,
      readQuantity: 2,
      writeAddress: 10,
      writeQuantity: 2,
      values: [100, 200],
    }),
    fields: [
      modbusCodeField,
      { name: 'readAddress', type: 'uint32', required: true, note: '读取起始寄存器地址，0–65535' },
      { name: 'readQuantity', type: 'uint32', required: true, note: '读取数量，1–125' },
      { name: 'writeAddress', type: 'uint32', required: true, note: '写入起始寄存器地址，0–65535' },
      { name: 'writeQuantity', type: 'uint32', required: true, note: '写入数量，1–121，且必须与 values 长度一致' },
      { name: 'values', type: 'uint32[]', required: true, note: '写入的寄存器值，每个值 0–65535（16 位）' },
    ],
    responseFields: REGISTER_RESPONSE,
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.maskWriteRegister',
    label: '屏蔽写寄存器',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/MaskWriteRegister',
    description: '使用 AND/OR 掩码修改保持寄存器（功能码 0x16）。',
    example: json({ modbusCode: '', address: 0, andMask: 65535, orMask: 0 }),
    fields: [
      modbusCodeField,
      modbusAddressField('寄存器地址，0–65535'),
      { name: 'andMask', type: 'uint32', required: true, note: 'AND 掩码，0–65535（16 位）' },
      { name: 'orMask', type: 'uint32', required: true, note: 'OR 掩码，0–65535（16 位）' },
    ],
    responseFields: BYTES_RESPONSE,
    tips: MODBUS_TIPS,
    write: true,
  },
  // ---------------- Modbus Configuration ----------------
  {
    key: 'modbus.saveConfig',
    label: '保存配置',
    category: 'modbus-config',
    rpc: 'bridgemodbus.BridgeModbus/SaveConfig',
    description: '新增或更新 Modbus 链路配置。',
    write: true,
    example: json({ modbusCode: 'line-01', slaveAddress: '192.0.2.10:502', slave: 1 }),
    fields: [
      { name: 'modbusCode', type: 'string', required: true, note: 'Modbus 链路唯一编码' },
      { name: 'slaveAddress', type: 'string', required: true, note: '从站通信地址，如 host:port' },
      { name: 'slave', type: 'uint32', required: true, note: '从站地址（Slave ID）' },
    ],
    responseFields: [{ name: 'id', type: 'string', note: '新增或更新后的配置主键 ID' }],
    tips: [],
  },
  {
    key: 'modbus.deleteConfig',
    label: '删除配置',
    category: 'modbus-config',
    rpc: 'bridgemodbus.BridgeModbus/DeleteConfig',
    description: '按配置 ID 批量删除 Modbus 链路配置。',
    example: json({ ids: ['config-id-1', 'config-id-2'] }),
    fields: [{ name: 'ids', type: 'string[]', required: true, note: '待删除配置的主键 ID 列表' }],
    responseFields: [],
    tips: [],
    write: true,
  },
  {
    key: 'modbus.pageListConfig',
    label: '配置分页列表',
    category: 'modbus-config',
    rpc: 'bridgemodbus.BridgeModbus/PageListConfig',
    description: '分页查询 Modbus 链路配置，可按编码关键词和状态过滤。',
    example: json({ page: 1, pageSize: 20, keyword: '', status: 1 }),
    fields: [
      { name: 'page', type: 'int64', required: true, note: '页码' },
      { name: 'pageSize', type: 'int64', required: true, note: '每页数量' },
      { name: 'keyword', type: 'string', note: '模糊匹配 modbusCode' },
      { name: 'status', type: 'uint32', note: '省略或传 0 时不过滤，传 1 时仅查询启用配置' },
    ],
    responseFields: [
      { name: 'total', type: 'uint32', note: '符合条件的配置总数' },
      { name: 'cfg', type: 'PbModbusConfig[]', note: '当前页配置列表' },
    ],
    tips: [],
  },
  {
    key: 'modbus.getConfigByCode',
    label: '按编码查询配置',
    category: 'modbus-config',
    rpc: 'bridgemodbus.BridgeModbus/GetConfigByCode',
    description: '按 Modbus 链路编码查询配置详情。',
    example: json({ modbusCode: 'line-01' }),
    fields: [{ name: 'modbusCode', type: 'string', required: true, note: 'Modbus 链路唯一编码' }],
    responseFields: [{ name: 'cfg', type: 'PbModbusConfig', note: '匹配的链路配置详情' }],
    tips: [],
  },
  {
    key: 'modbus.batchGetConfigByCode',
    label: '按编码批量查询配置',
    category: 'modbus-config',
    rpc: 'bridgemodbus.BridgeModbus/BatchGetConfigByCode',
    description: '按多个 Modbus 链路编码批量查询配置详情。',
    example: json({ modbusCode: ['line-01', 'line-02'] }),
    fields: [{ name: 'modbusCode', type: 'string[]', required: true, note: 'Modbus 链路编码列表' }],
    responseFields: [{ name: 'cfg', type: 'PbModbusConfig[]', note: '查询到的链路配置列表' }],
    tips: [],
  },
  // ---------------- Modbus Conversion ----------------
  {
    key: 'modbus.batchConvertValue16ToRegisters',
    label: '16 位数值转寄存器',
    category: 'modbus-convert',
    rpc: 'bridgemodbus.BridgeModbus/BatchConvertValue16ToRegisters',
    description: '将 int16/uint16 十进制值编码为 16 位寄存器。',
    example: json({ values: [220, -12], unsigned: false }),
    fields: [
      { name: 'values', type: 'int32[]', required: true, note: '整数列表；signed 范围 -32768–32767，unsigned 范围 0–65535' },
      { name: 'unsigned', type: 'bool', note: 'true=无符号；false=有符号，缺省 false' },
    ],
    responseFields: [
      { name: 'uint16Values', type: 'uint32[]', note: '无符号寄存器值，范围 0–65535' },
      { name: 'int16Values', type: 'int32[]', note: '有符号视图，范围 -32768–32767' },
      { name: 'hexValues', type: 'string[]', note: '0xXXXX 格式' },
      { name: 'binaryValues', type: 'string[]', note: '16 位二进制字符串' },
      { name: 'bytes', type: 'bytes', note: '大端字节序；protojson 编码为 base64' },
    ],
    tips: ['由服务端执行纯计算，不访问设备。'],
  },
  {
    key: 'modbus.batchConvertValue32ToRegisters',
    label: '32 位数值转寄存器',
    category: 'modbus-convert',
    rpc: 'bridgemodbus.BridgeModbus/BatchConvertValue32ToRegisters',
    description: '将 uint32/int32/float32 按指定字序编码为两个连续寄存器。',
    example: json({ values: [220.5], dataType: 'VALUE32_FLOAT32', wordOrder: 'WORD_ORDER_CDAB' }),
    fields: [
      { name: 'values', type: 'double[]', required: true, note: '数值列表，范围按 dataType 校验' },
      { name: 'dataType', type: 'Value32Type', required: true, note: 'VALUE32_UINT32 / VALUE32_INT32 / VALUE32_FLOAT32' },
      { name: 'wordOrder', type: 'WordOrder', required: true, note: 'WORD_ORDER_ABCD / CDAB / BADC / DCBA，必须显式指定' },
    ],
    responseFields: [
      { name: 'results', type: 'Value32EncodeResult[]', note: '逐值结果，含 registers、bytes、hex、binary、value' },
      { name: 'registers', type: 'uint32[]', note: '按输入顺序展平的寄存器，可直接写连续保持寄存器' },
    ],
    tips: ['单次最多写 123 个寄存器。按完整 32 位值分组时，每批最多 61 个值。'],
  },
  {
    key: 'modbus.batchConvertRegistersToValue32',
    label: '寄存器转 32 位数值',
    category: 'modbus-convert',
    rpc: 'bridgemodbus.BridgeModbus/BatchConvertRegistersToValue32',
    description: '将连续寄存器按指定字序解码为 uint32/int32/float32。',
    example: json({ registers: [32768, 17244], dataType: 'VALUE32_FLOAT32', wordOrder: 'WORD_ORDER_CDAB' }),
    fields: [
      { name: 'registers', type: 'uint32[]', required: true, note: '设备地址顺序的寄存器值，每项 0–65535；长度必须为偶数，不要预先重排' },
      { name: 'dataType', type: 'Value32Type', required: true, note: 'VALUE32_UINT32 / VALUE32_INT32 / VALUE32_FLOAT32' },
      { name: 'wordOrder', type: 'WordOrder', required: true, note: 'WORD_ORDER_ABCD / CDAB / BADC / DCBA，必须显式指定' },
    ],
    responseFields: [
      { name: 'results', type: 'Value32DecodeResult[]', note: '每两个寄存器一项，包含 value、uint32Value、int32Value、float32Value 及原始寄存器回显' },
    ],
    tips: ['registers、bytes 和 hex 按设备顺序原样回显。wordOrder 只用于解释数值。'],
  },
  ...GIS_METHODS,
]

export const DEFAULT_METHOD = METHODS[0]

export function findMethod(key: string): MethodDefinition {
  return METHODS.find((method) => method.key === key) ?? DEFAULT_METHOD
}

export const METHOD_GROUPS: Record<MethodDefinition['category'], { service: string; label: string }> = {
  podman: { service: 'Podman', label: '容器与镜像' },
  'modbus-read': { service: 'Modbus', label: '状态读取' },
  'modbus-write': { service: 'Modbus', label: '设备写入' },
  'modbus-config': { service: 'Modbus', label: '链路配置' },
  'modbus-convert': { service: 'Modbus', label: '寄存器换算' },
  'gis-grid': { service: 'GIS', label: '空间网格' },
  'gis-analysis': { service: 'GIS', label: '空间分析' },
  'gis-fence': { service: 'GIS', label: '电子围栏' },
}
