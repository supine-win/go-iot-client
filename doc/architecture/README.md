# 架构设计

## 包结构

- `core/`：结果模型、错误类型、重试策略
- `clients/`：按协议/设备族组织客户端
  - `clients/plc/`
  - `clients/modbus/`
- `mock/`：本地可控模拟服务器（用于集成测试）
- 根包 `iotclient`：对外稳定入口（当前主实现为 `MitsubishiClient`）

## 设计原则

1. 对外 API 保持稳定，内部可逐步替换为协议子包实现。
2. 传输、协议、业务语义分层，便于测试与故障定位。
3. 先保证 PLC-MES bridge 生产链路稳定，再扩展全协议覆盖。

