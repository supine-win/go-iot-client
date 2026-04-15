# 常见问题（FAQ）

## 1) 为什么返回 `Result/ResultT`，而不是直接返回 `error`？

为了统一协议层返回结构，除错误信息外还可携带耗时、请求/响应摘要等诊断信息，便于 UI 与桥接层统一处理。

## 2) 客户端断线后会自动重连吗？

会。多数客户端默认启用重试与重连，具体次数和间隔可通过 `SetRetryPolicy` 调整。

## 3) `ReadString` 长度单位是什么？

统一使用“字符/字节长度”语义（按客户端实现处理），建议在协议对接时固定长度并做上限校验。

## 4) 什么时候需要自己实现 mock？

当你需要验证特定厂商设备的边界行为（例如异常码、特殊帧字段）时，建议扩展 `mock/` 或在测试中自建专用 mock 服务。

## 5) 如何判断一个协议能力是否完整？

优先看两处：

- [`doc/protocols/README.md`](../protocols/README.md) 的状态说明
- [`doc/migration/parity-matrix.md`](../migration/parity-matrix.md) 的方法级覆盖
