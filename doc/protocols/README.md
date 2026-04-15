# 协议支持矩阵

| 协议域 | 客户端 | 状态 |
|---|---|---|
| Mitsubishi MC | `MitsubishiClient` | 已实现（Qna_3E） |
| Siemens S7 | `clients/plc/SiemensClient` | 骨架已建立 |
| Omron FINS | `clients/plc/OmronFinsClient` | 骨架已建立 |
| Allen-Bradley | `clients/plc/AllenBradleyClient` | 骨架已建立 |
| Modbus TCP | `clients/modbus/TcpClient` | 骨架已建立 |
| Modbus RTU over TCP | `clients/modbus/RtuOverTcpClient` | 骨架已建立 |
| Modbus RTU | `clients/modbus/RtuClient` | 骨架已建立 |
| Modbus ASCII | `clients/modbus/AsciiClient` | 骨架已建立 |

## 兼容约定

- “骨架已建立”表示类型、构造器、统一接口已可编译接入，后续按 parity 清单补齐协议语义。
- 对 bridge 当前链路所需能力，保持“已实现 + 回归通过”优先级最高。

