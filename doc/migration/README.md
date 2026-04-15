# C# IoTClient 迁移对齐

- [Parity Matrix](parity-matrix.md)

本目录记录 C# `IoTClient` 到 Go `go-iotclient` 的源码级映射，用于：

1. 功能完整性追踪
2. 迁移透明化审查
3. 后续维护与回归依据

## 如何使用 Parity Matrix

1. 先按“客户端级映射”确认是否已有 Go 对应实现
2. 再按“方法级覆盖”判断能力是否达到可用级
3. 对于 `compatible` 状态，需补齐实现或在文档中明确限制

## 迁移完成判定建议

- 所有目标客户端至少达到 `implemented` 或明确 `deferred` 原因
- 关键接口（`Open/Close/Read*/Write`）具备测试覆盖
- 文档中的协议矩阵与代码状态一致

