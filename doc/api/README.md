# API 使用说明

## 根包快速使用

```go
package main

import (
	"fmt"

	iotclient "github.com/example/go-iotclient"
)

func readSample() error {
	client := iotclient.NewMitsubishiClient(iotclient.MitsubishiVersionQna3E, "192.168.10.10", 5002, 1500)
	open := client.Open()
	if !open.IsSucceed {
		return fmt.Errorf("open plc failed: %s", open.Err)
	}
	defer client.Close()

	r := client.ReadInt16("D4001")
	if !r.IsSucceed {
		return fmt.Errorf("read D4001 failed: %s", r.Err)
	}
	return nil
}
```

## 统一接口客户端

- `clients/plc`: `MitsubishiClient`, `SiemensClient`, `OmronFinsClient`, `AllenBradleyClient`
- `clients/modbus`: `TcpClient`, `RtuOverTcpClient`, `RtuClient`, `AsciiClient`
- 以上客户端都实现 `clients.IoTClient`：`Open/Close/Connected/ReadInt16/ReadInt32/ReadFloat/ReadString/Write`

## 客户端示例（以 Modbus TCP 为例）

```go
package main

import (
	"fmt"

	modbusclient "github.com/example/go-iotclient/clients/modbus"
)

func modbusSample() error {
	modbus := modbusclient.NewTcpClient("127.0.0.1:502")
	if r := modbus.Open(); !r.IsSucceed {
		return fmt.Errorf("open modbus failed: %s", r.Err)
	}
	defer modbus.Close()

	if w := modbus.Write("100", int16(123)); !w.IsSucceed {
		return fmt.Errorf("write failed: %s", w.Err)
	}
	return nil
}
```

## 连接健壮性参数

- `SetReadWriteTimeout(read, write)`
- `SetRetryPolicy(maxRetries, retryDelay)`：失败后自动重连并重试请求
- `SetRoute(networkNo, stationNo, moduleIO, multidropStation)`

## 健壮性与异常处理

- 所有读写接口均返回 `core.Result` / `core.ResultT[T]`，避免直接 panic。
- 默认启用重试（可配置），网络断连会触发自动重连。
- 协议层包含校验与异常码处理（例如 Modbus CRC/LRC、功能码异常、FINS end code）。

