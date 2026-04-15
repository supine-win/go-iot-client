# Parity Matrix（C# IoTClient -> Go）

## 客户端级映射

| C# 类 | 路径 | Go 对应 | 状态 |
|---|---|---|---|
| `MitsubishiClient` | `IoTClient/Clients/PLC/MitsubishiClient.cs` | `iotclient.MitsubishiClient` | implemented |
| `SiemensClient` | `IoTClient/Clients/PLC/SiemensClient.cs` | `clients/plc/SiemensClient` | compatible |
| `OmronFinsClient` | `IoTClient/Clients/PLC/OmronFinsClient.cs` | `clients/plc/OmronFinsClient` | compatible |
| `AllenBradleyClient` | `IoTClient/Clients/PLC/AllenBradleyClient.cs` | `clients/plc/AllenBradleyClient` | compatible |
| `ModbusTcpClient` | `IoTClient/Clients/Modbus/ModbusTcpClient.cs` | `clients/modbus/TcpClient` | compatible |
| `ModbusRtuOverTcpClient` | `IoTClient/Clients/Modbus/ModbusRtuOverTcpClient.cs` | `clients/modbus/RtuOverTcpClient` | compatible |
| `ModbusRtuClient` | `IoTClient/Clients/Modbus/ModbusRtuClient.cs` | `clients/modbus/RtuClient` | compatible |
| `ModbusAsciiClient` | `IoTClient/Clients/Modbus/ModbusAsciiClient.cs` | `clients/modbus/AsciiClient` | compatible |

## 基础模型映射

| C# 类型 | Go 类型 | 状态 |
|---|---|---|
| `Result` / `Result<T>` | `core.Result` / `core.ResultT[T]` | implemented |
| `DataTypeEnum` | （待补）`core.DataType` | compatible |
| `SocketBase` | `MitsubishiClient` 内部 tcp 流程 + `SetRetryPolicy` | compatible |

## 状态定义

- `implemented`：协议语义已完整可用并通过回归
- `compatible`：API/类型已建立，可编译接入，语义实现按协议优先级补齐
- `deferred`：仅用于明确延期项（本矩阵当前无该状态）

