# 生产环境最佳实践

本文档聚焦 `jetcache-go` 在生产场景中的落地建议。

## 1. 先选对缓存拓扑

- 仅本地：单实例或短时热点数据场景。
- 仅远程：跨实例一致性优先。
- 两级缓存（local + remote）：高 QPS 接口默认推荐。

经验规则：

- 读多写少且 QPS 高：`both`
- 只强调跨实例共享：`remote`
- 追求极低延迟且无需共享：`local`

## 2. Key 设计

- 使用稳定业务前缀：`user:profile:1001`。
- key 构成保持确定性，避免嵌入频繁变化字段。
- 使用泛型 `NewT` 时，统一分隔符策略（`WithSeparator`）。

## 3. TTL 策略

- TTL 应由业务新鲜度 SLA 决定，而不是默认值。
- 用 `WithRemoteExpiry(...)` 设全局 TTL，热点 key 用 `TTL(...)` 覆盖。
- 用 `WithNotFoundExpiry(...)` 控制空值缓存时长，防穿透。
- 保留 `WithOffset` 抖动，避免雪崩式同时过期。

## 4. 防穿透与防击穿

- 配置 `WithErrNotFound(err)`（例如 `sql.ErrNoRows`）启用空值缓存。
- 读路径优先用 `Once(...)` + `Do(...)`，利用 singleflight。
- 热点 key 高并发场景避免 `Get(...)` + 手工回源。

## 5. 自动刷新要谨慎开启

仅在以下条件满足时使用：

- key 数量少，
- 回源代价高，
- 实时性要求不高。

检查项：

- 设置 `WithRefreshDuration(...)`（例如 30s-5m）。
- 设置 `WithStopRefreshAfterLastAccess(...)`，避免冷 key 长期刷新。
- 按后端容量设置 `WithRefreshConcurrency(...)`。
- 服务退出时调用 `Close()`。

容量估算：

```text
max_refresh_keys ~= refreshDuration/loadCost * refreshConcurrency * instanceCount
```

## 6. 故障时保护后端

- `both` 模式保持本地缓存以增强降级能力。
- `Do(...)` 回源函数使用短超时 context。
- 重试策略放在后端客户端层，不要在缓存回调里无限重试。

## 7. 观测基线

- 至少启用一种统计处理器（`stats.NewStatsLogger`）。
- 生产接入 Prometheus。
- 重点告警：
  - 命中率快速下降，
  - remote miss 激增，
  - query fail 持续上升。

参考：[监控指南](Monitoring.md)

## 8. 序列化选择

- 默认 `msgpack` 适合作为通用生产默认。
- 追求可读性或跨语言调试可选 `json`。
- 切换 codec 前先验证 payload 大小和 schema 兼容性。

## 9. 批量访问模式

ID 批量读取优先使用泛型 `MGet`：

- 远程读写使用 pipeline，
- miss 分组走 singleflight，
- 异常时优先返回可用结果。

上游必须感知部分失败时，使用 `MGetWithErr`。

## 10. 性能压测与容量验证

仓库自带基准测试：`bench_test.go`。

执行示例：

```bash
go test -run '^$' -bench 'BenchmarkOnce|BenchmarkMGet' -benchmem ./...
```

建议维度：

- 本地缓存实现（`TinyLFU` vs `FreeCache`），
- payload 大小（小/中/大），
- miss 比例（1%、10%、50%），
- 并发等级（如 50/200/1000）。

建议把基准结果纳入 CI 工件，持续追踪回归。

## 11. 上线前检查清单

- 明确设置 `WithName(...)`。
- 配置 `WithErrNotFound(...)` 与 not-found TTL。
- 调优 Redis 超时与连接池。
- 检查指标和告警规则是否生效。
- 进程退出路径调用 `Close()`。
