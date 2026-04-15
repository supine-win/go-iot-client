# Parity Matrix（C# IoTClient -> Go）

## 客户端级映射

| C# 类 | 路径 | Go 对应 | 状态 |
|---|---|---|---|
| `MitsubishiClient` | `IoTClient/Clients/PLC/MitsubishiClient.cs` | `iotclient.MitsubishiClient` | implemented |
| `SiemensClient` | `IoTClient/Clients/PLC/SiemensClient.cs` | `clients/plc/SiemensClient` | implemented**** |
| `OmronFinsClient` | `IoTClient/Clients/PLC/OmronFinsClient.cs` | `clients/plc/OmronFinsClient` | implemented** |
| `AllenBradleyClient` | `IoTClient/Clients/PLC/AllenBradleyClient.cs` | `clients/plc/AllenBradleyClient` | implemented*** |
| `ModbusTcpClient` | `IoTClient/Clients/Modbus/ModbusTcpClient.cs` | `clients/modbus/TcpClient` | implemented* |
| `ModbusRtuOverTcpClient` | `IoTClient/Clients/Modbus/ModbusRtuOverTcpClient.cs` | `clients/modbus/RtuOverTcpClient` | implemented* |
| `ModbusRtuClient` | `IoTClient/Clients/Modbus/ModbusRtuClient.cs` | `clients/modbus/RtuClient` | implemented* |
| `ModbusAsciiClient` | `IoTClient/Clients/Modbus/ModbusAsciiClient.cs` | `clients/modbus/AsciiClient` | implemented* |

## 基础模型映射

| C# 类型 | Go 类型 | 状态 |
|---|---|---|
| `Result` / `Result<T>` | `core.Result` / `core.ResultT[T]` | implemented |
| `DataTypeEnum` | （待补）`core.DataType` | compatible |
| `SocketBase` | `MitsubishiClient` 内部 tcp 流程 + `SetRetryPolicy` | compatible |

## 方法级覆盖（统一 IoTClient 接口）

| 方法 | Mitsubishi | Siemens | OmronFins | AllenBradley | Modbus 全系 |
|---|---|---|---|---|---|
| `Open/Close/Connected` | implemented | implemented | implemented | implemented | implemented |
| `ReadInt16` | implemented | implemented | implemented | implemented | implemented |
| `ReadInt32` | implemented | implemented | implemented | implemented | implemented |
| `ReadFloat` | implemented | implemented | implemented | implemented | implemented |
| `ReadString` | implemented | implemented | implemented | implemented | implemented |
| `Write(interface{})` | implemented | implemented | implemented | implemented | implemented |

## 状态定义

- `implemented`：协议语义已完整可用并通过回归
- `compatible`：API/类型已建立，可编译接入，语义实现按协议优先级补齐
- `deferred`：仅用于明确延期项（本矩阵当前无该状态）

\* `implemented` 当前范围：`FC03/FC04/FC05/FC10`、连接重试自恢复、RTU CRC 校验、ASCII LRC 校验、基础类型读写与字符串读写。

\** `OmronFinsClient` 当前范围：FINS/TCP 基础握手、`D/C/W/H/A` 区域字读写、基础类型读写与字符串读写、失败重连重试。

\*** `AllenBradleyClient` 当前范围：CIP 会话注册、Tag 基础读写、基础类型读写与字符串读写、失败重连重试。

\**** `SiemensClient` 当前范围：基于 `gos7` 的 S7 TCP 连接、`DB/M/I/Q` 区域基础读写、基础类型与字符串读写、失败重连重试。

