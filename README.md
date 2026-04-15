# go-iotclient

Go implementation inspired by C# `IoTClient`.

## Package Layout

- `core/` common result/error/retry models
- `clients/plc/` PLC client entry points
- `clients/modbus/` Modbus client entry points
- `mock/` integration mock server
- `doc/` categorized documentation

## Documentation

- [Doc Index](doc/README.md)
- [Parity Matrix](doc/migration/parity-matrix.md)

## Quick Test

```bash
go test ./...
```

