import type { MethodDefinition, MethodField } from '../types'

function json(value: unknown): string {
  return JSON.stringify(value, null, 2)
}

const modbusCodeField: MethodField = {
  name: 'modbusCode',
  type: 'string',
  note: 'Modbus 配置唯一编码；留空使用默认连接配置',
}

const modbusAddressField: MethodField = {
  name: 'address',
  type: 'uint32',
  required: true,
  note: '起始地址，0–65535',
}

const modbusQuantityField = (note: string): MethodField => ({
  name: 'quantity',
  type: 'uint32',
  required: true,
  note,
})

const MODBUS_TIPS = [
  'modbusCode 留空时使用 bridgemodbus 的默认连接配置。',
  '地址与数量为 uint32；读寄存器数量最多 125，读线圈/离散输入最多 2000。',
  '响应中的 bytes 字段在 protojson 中为 base64 字符串。',
]

/**
 * 与 facade/openapi/internal/dispatch 的 method 注册表人工同步。
 * 后端新增 method 时，这里必须同步补一条，否则控制台无法选择。
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
      { name: 'offset', type: 'int32', note: '偏移量' },
      { name: 'names', type: 'string[]', note: '按容器名过滤' },
      { name: 'ids', type: 'string[]', note: '按容器 ID 过滤' },
      { name: 'labels', type: 'map<string,string>', note: '按标签过滤' },
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
      { name: 'offset', type: 'int32', note: '偏移量' },
      { name: 'references', type: 'string[]', note: '按镜像引用过滤，如 nginx:latest' },
      { name: 'includeDigests', type: 'bool', note: '是否返回 digest' },
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
    fields: [modbusAddressField, modbusQuantityField('读取数量，1–2000')],
    tips: [...MODBUS_TIPS, 'values 与起始地址一一对应。'],
  },
  {
    key: 'modbus.readDiscreteInputs',
    label: '读离散输入',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadDiscreteInputs',
    description: '读取离散输入状态（功能码 0x02），返回按位结果与布尔数组。',
    example: json({ modbusCode: '', address: 0, quantity: 8 }),
    fields: [modbusAddressField, modbusQuantityField('读取数量，1–2000')],
    tips: MODBUS_TIPS,
  },
  {
    key: 'modbus.readInputRegisters',
    label: '读输入寄存器',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadInputRegisters',
    description: '读取输入寄存器（功能码 0x04），返回无符号、有符号、十六进制与二进制表示。',
    example: json({ modbusCode: '', address: 0, quantity: 4 }),
    fields: [modbusAddressField, modbusQuantityField('读取数量，1–125')],
    tips: MODBUS_TIPS,
  },
  {
    key: 'modbus.readHoldingRegisters',
    label: '读保持寄存器',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadHoldingRegisters',
    description: '读取保持寄存器（功能码 0x03），返回多种数值表示。',
    example: json({ modbusCode: '', address: 0, quantity: 4 }),
    fields: [modbusAddressField, modbusQuantityField('读取数量，1–125')],
    tips: MODBUS_TIPS,
  },
  {
    key: 'modbus.readFIFOQueue',
    label: '读 FIFO 队列',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadFIFOQueue',
    description: '读取 FIFO 队列内容（功能码 0x18）。',
    example: json({ modbusCode: '', address: 0 }),
    fields: [modbusAddressField],
    tips: MODBUS_TIPS,
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
      { name: 'readDeviceIdCode', type: 'uint32', required: true, note: '0x01 基本 / 0x02 常规 / 0x03 扩展' },
    ],
    tips: MODBUS_TIPS,
  },
  {
    key: 'modbus.readDeviceIdentificationSpecificObject',
    label: '读指定设备对象',
    category: 'modbus-read',
    rpc: 'bridgemodbus.BridgeModbus/ReadDeviceIdentificationSpecificObject',
    description: '读取指定 Object ID 的设备标识，例如厂商名称、产品代码、版本号。',
    example: json({ modbusCode: '', objectId: 0 }),
    fields: [
      modbusCodeField,
      { name: 'objectId', type: 'uint32', required: true, note: '0x00 厂商名 / 0x01 产品代码 / 0x02 版本 / 0x03 URL / 0x04 产品名 / 0x05 型号 / 0x06 应用名' },
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
    fields: [modbusAddressField, { name: 'value', type: 'bool', required: true, note: 'true=ON，false=OFF' }],
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeMultipleCoils',
    label: '写多个线圈',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteMultipleCoils',
    description: '写多个线圈（功能码 0x0F），values 从起始地址依次写入。',
    example: json({ modbusCode: '', address: 0, quantity: 4, values: [true, false, true, false] }),
    fields: [
      modbusAddressField,
      modbusQuantityField('写入数量，与 values 长度一致'),
      { name: 'values', type: 'bool[]', required: true, note: '每个元素对应一个线圈' },
    ],
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeSingleRegister',
    label: '写单个寄存器',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteSingleRegister',
    description: '写单个保持寄存器（功能码 0x06），value 为寄存器原始 uint32 值。',
    example: json({ modbusCode: '', address: 0, value: 100 }),
    fields: [modbusAddressField, { name: 'value', type: 'uint32', required: true, note: '寄存器原始值' }],
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeSingleRegisterWithDecimal',
    label: '写单个寄存器（十进制）',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteSingleRegisterWithDecimal',
    description: '按十进制数值写单个保持寄存器，可指定有符号/无符号。',
    example: json({ modbusCode: '', address: 0, value: -12, unsigned: false }),
    fields: [
      modbusAddressField,
      { name: 'value', type: 'int32', required: true, note: '十进制数值，负数需 unsigned=false' },
      { name: 'unsigned', type: 'bool', note: 'true=无符号，false=有符号' },
    ],
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeMultipleRegisters',
    label: '写多个寄存器',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteMultipleRegisters',
    description: '写多个保持寄存器（功能码 0x10），values 为原始 uint32 列表。',
    example: json({ modbusCode: '', address: 0, quantity: 2, values: [100, 200] }),
    fields: [
      modbusAddressField,
      modbusQuantityField('写入数量，与 values 长度一致'),
      { name: 'values', type: 'uint32[]', required: true, note: '每个值对应一个寄存器' },
    ],
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.writeMultipleRegistersWithDecimal',
    label: '写多个寄存器（十进制）',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/WriteMultipleRegistersWithDecimal',
    description: '按十进制数值写多个保持寄存器，可指定有符号/无符号。',
    example: json({ modbusCode: '', address: 0, quantity: 2, values: [-12, 34], unsigned: false }),
    fields: [
      modbusAddressField,
      modbusQuantityField('写入数量，与 values 长度一致'),
      { name: 'values', type: 'int32[]', required: true, note: '十进制数值列表' },
      { name: 'unsigned', type: 'bool', note: 'true=无符号，false=有符号' },
    ],
    tips: MODBUS_TIPS,
    write: true,
  },
  {
    key: 'modbus.readWriteMultipleRegisters',
    label: '读写多个寄存器',
    category: 'modbus-write',
    rpc: 'bridgemodbus.BridgeModbus/ReadWriteMultipleRegisters',
    description: '在一个事务中先写后读保持寄存器（功能码 0x17）。',
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
      { name: 'readAddress', type: 'uint32', required: true, note: '读取起始地址' },
      { name: 'readQuantity', type: 'uint32', required: true, note: '读取数量，1–125' },
      { name: 'writeAddress', type: 'uint32', required: true, note: '写入起始地址' },
      { name: 'writeQuantity', type: 'uint32', required: true, note: '写入数量，与 values 一致' },
      { name: 'values', type: 'uint32[]', required: true, note: '写入的寄存器值' },
    ],
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
      modbusAddressField,
      { name: 'andMask', type: 'uint32', required: true, note: 'AND 掩码' },
      { name: 'orMask', type: 'uint32', required: true, note: 'OR 掩码' },
    ],
    tips: MODBUS_TIPS,
    write: true,
  },
]

export const DEFAULT_METHOD = METHODS[0]

export function findMethod(key: string): MethodDefinition {
  return METHODS.find((method) => method.key === key) ?? DEFAULT_METHOD
}

export const METHOD_GROUP_ORDER: MethodDefinition['category'][] = ['podman', 'modbus-read', 'modbus-write']

export const METHOD_GROUP_LABELS: Record<MethodDefinition['category'], string> = {
  podman: 'Podman 容器',
  'modbus-read': 'Modbus 读取',
  'modbus-write': 'Modbus 写入',
}
