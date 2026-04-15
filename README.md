# go-iotclient

Go implementation inspired by C# `IoTClient`, focused on robust industrial communication clients with a unified result model.

[English](README.md) | [简体中文](README.zh-CN.md)

[![CI](https://github.com/supine-win/go-iot-client/actions/workflows/ci.yml/badge.svg)](https://github.com/supine-win/go-iot-client/actions/workflows/ci.yml)

## Features

- Unified client interface: `Open/Close/Connected/ReadInt16/ReadInt32/ReadFloat/ReadString/Write`
- PLC clients: `Mitsubishi`, `Siemens S7`, `Omron FINS`, `Allen-Bradley`
- Modbus clients: `TCP`, `RTU over TCP`, `RTU`, `ASCII`
- Built-in retry and reconnect behavior with consistent `core.Result` / `core.ResultT[T]`

## Install

```bash
go get github.com/example/go-iotclient
```

## Quick Start

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

## Package Layout

- `core/` common result/error/retry models
- `clients/plc/` PLC clients (`Mitsubishi`, `Siemens S7`, `Omron FINS`, `Allen-Bradley`)
- `clients/modbus/` Modbus clients (`TCP`, `RTU over TCP`, `RTU`, `ASCII`)
- `mock/` integration mock server
- `doc/` categorized documentation

## Documentation

- [Doc Index](doc/README.md)
- [API Guide](doc/api/README.md)
- [Protocol Matrix](doc/protocols/README.md)
- [Parity Matrix](doc/migration/parity-matrix.md)
- [Testing Guide](doc/testing/README.md)

## Project Standards

- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [Changelog](CHANGELOG.md)
- [License](LICENSE)

## Development

```bash
go test ./...
go test -race ./...
go vet ./...
```

