# API 使用说明

## 根包快速使用

```go
client := iotclient.NewMitsubishiClient(iotclient.MitsubishiVersionQna3E, "192.168.10.10", 5002, 1500)
open := client.Open()
if !open.IsSucceed { panic(open.Err) }
defer client.Close()

r := client.ReadInt16("D4001")
if !r.IsSucceed { panic(r.Err) }
```

## 连接健壮性参数

- `SetReadWriteTimeout(read, write)`
- `SetRetryPolicy(maxRetries, retryDelay)`：失败后自动重连并重试请求
- `SetRoute(networkNo, stationNo, moduleIO, multidropStation)`

