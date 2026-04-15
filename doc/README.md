# go-iotclient 文档总览

## 阅读入口

- 仓库总览（中文）：[`README.zh-CN.md`](../README.zh-CN.md)
- 仓库总览（English）：[`README.md`](../README.md)

## 目录导航

- [架构设计](architecture/README.md)
- [协议支持矩阵](protocols/README.md)
- [API 使用说明](api/README.md)
- [测试与覆盖率](testing/README.md)
- [C# IoTClient 迁移对齐](migration/README.md)
- [常见问题（FAQ）](faq/README.md)

## 建议阅读顺序

1. `协议支持矩阵`：先确认当前支持范围与边界
2. `API 使用说明`：再看接入方式和错误处理约定
3. `测试与覆盖率`：最后看验证方法和质量门禁

## 文档维护约定

- 协议行为变化时，必须同步更新 `protocols` 和 `migration/parity-matrix`
- 新增公开 API 时，必须同步更新 `api` 文档与最小可运行示例
- 提交前至少执行一次 `go test ./...`，并确保示例代码与当前 API 一致

> 所有链接均使用相对路径，确保 GitHub 在线浏览与目录跳转可用。

