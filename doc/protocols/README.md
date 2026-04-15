# 协议支持矩阵

| 协议域 | 客户端 | 状态 |
|---|---|---|
| Mitsubishi MC | `MitsubishiClient` | 已实现（Qna_3E） |
| Siemens S7 | `clients/plc/SiemensClient` | 已实现（基于 gos7 的 S7 读写 + 重试） |
| Omron FINS | `clients/plc/OmronFinsClient` | 已实现（FINS/TCP 握手 + 读写 + 重试） |
| Allen-Bradley | `clients/plc/AllenBradleyClient` | 已实现（CIP 会话 + Tag 读写 + 重试） |
| Modbus TCP | `clients/modbus/TcpClient` | 已实现（FC03/04/05/10，含重试自恢复） |
| Modbus RTU over TCP | `clients/modbus/RtuOverTcpClient` | 已实现（FC03/04/05/10，含 CRC 校验） |
| Modbus RTU | `clients/modbus/RtuClient` | 已实现（串口链路 + RTU 帧 + CRC 校验） |
| Modbus ASCII | `clients/modbus/AsciiClient` | 已实现（串口链路 + ASCII 帧 + LRC 校验） |

## 状态定义

- `已实现`：主流程与关键异常路径已具备测试覆盖，可用于生产集成
- `兼容`：接口已对齐，语义实现仍在补齐
- `规划中`：暂未进入代码实现阶段

## 能力边界说明

- Modbus 全系客户端当前支持 `ReadInt16/ReadInt32/ReadFloat/ReadString` 与 `Write(bool/int/int16/uint/uint16/int32/uint32/int64/uint64/float32/float64/string/[]byte)`。
- Siemens S7 当前支持 `DB/M/I/Q` 区域地址（如 `DB1.DBW20`、`M100`）的基础读写。
- Omron FINS 当前支持区域 `D/C/W/H/A` 的字读写（`ReadInt16/ReadInt32/ReadFloat/ReadString` 与通用 `Write`）。
- Allen-Bradley 当前支持会话注册、Tag 基础读写（`ReadInt16/ReadInt32/ReadFloat/ReadString` 与通用 `Write`）。
- 串口客户端默认参数：RTU(`9600/8N1`)、ASCII(`9600/7E1`)，可通过 `SetSerialMode` 覆盖。
- 对 bridge 当前链路所需能力，保持“已实现 + 回归通过”优先级最高。

## 后续增强建议

1. 增加 Siemens 的协议级 mock 集成测试（不仅地址解析）
2. 增加 Allen-Bradley / Omron 的更细粒度异常码回归
3. 增加跨协议统一性能基线（延迟/吞吐/重试开销）

