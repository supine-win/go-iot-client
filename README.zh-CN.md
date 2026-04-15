# go-iotclient（中文说明）

受 C# `IoTClient` 启发的 Go 实现，目标是提供统一、稳健、可测试的工业协议客户端能力。

[English](README.md) | [简体中文](README.zh-CN.md)

[![CI](https://github.com/supine-win/go-iot-client/actions/workflows/ci.yml/badge.svg)](https://github.com/supine-win/go-iot-client/actions/workflows/ci.yml)

## 功能概览

- 统一客户端接口：`Open/Close/Connected/ReadInt16/ReadInt32/ReadFloat/ReadString/Write`
- PLC 客户端：`Mitsubishi`、`Siemens S7`、`Omron FINS`、`Allen-Bradley`
- Modbus 客户端：`TCP`、`RTU over TCP`、`RTU`、`ASCII`
- 内置重试与重连机制，统一返回 `core.Result` / `core.ResultT[T]`

## 安装

```bash
go get github.com/example/go-iotclient
```

## 快速开始

```go
package main

import (
	"fmt"

	iot "github.com/example/go-iotclient"
)

func main() {
	client := iot.NewMitsubishiClient(iot.MitsubishiVersionQna3E, "192.168.10.10", 5002, 1500)
	if r := client.Open(); !r.IsSucceed {
		panic(r.Err)
	}
	defer client.Close()

	value := client.ReadInt16("D4001")
	if !value.IsSucceed {
		panic(value.Err)
	}
	fmt.Println("D4001 =", value.Value)
}
```

## 包结构

- `core/`：结果模型、错误定义、重试策略
- `clients/plc/`：PLC 客户端实现
- `clients/modbus/`：Modbus 客户端实现
- `mock/`：协议模拟服务（用于集成/故障注入测试）
- `doc/`：分层文档

## 文档导航

- [文档总览](doc/README.md)
- [API 使用说明](doc/api/README.md)
- [协议支持矩阵](doc/protocols/README.md)
- [迁移对齐矩阵](doc/migration/parity-matrix.md)
- [测试与覆盖说明](doc/testing/README.md)

## 项目治理

- [贡献指南](CONTRIBUTING.md)
- [行为准则](CODE_OF_CONDUCT.md)
- [安全策略](SECURITY.md)
- [变更日志](CHANGELOG.md)
- [许可证](LICENSE)

## 开发命令

```bash
go test ./...
go test -race ./...
go vet ./...
```
