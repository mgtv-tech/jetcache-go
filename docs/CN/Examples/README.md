# 场景示例

本目录提供按场景组织的示例文档。

## 目录

- [基础 CRUD](BasicCRUD.md)
- [微服务模式](Microservice.md)
- [高并发热点 Key](HighConcurrency.md)

## 运行建议

- 先启动本地 Redis。
- 用 `go test ./...` 验证核心行为。
- 根据业务 SLA 调整 key 命名和 TTL 策略。
- 使用旧版本标签时请先确认功能可用性：
  - 泛型 `MGet` 回源函数 + pipeline 优化：`v1.1.0+`
  - 跨进程本地失效同步（`WithSyncLocal`）：`v1.1.1+`
- 详见 [版本与功能可用性](../Versioning.md)。
