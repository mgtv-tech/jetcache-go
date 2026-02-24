# 常见问题（FAQ）

## 默认缓存模式是什么？

由配置决定：

- 只配 `WithLocal`：本地模式。
- 只配 `WithRemote`：远程模式。
- 两者都配：两级缓存模式。

## `Get` 和 `Once` 应该用哪个？

常规 cache-aside 读路径建议用 `Once`，它会处理 miss 回源和 singleflight。
只有在你明确要求“只读缓存、不回源”时使用 `Get`。

## 如何防止缓存穿透？

配置 `WithErrNotFound(err)`，并可配合 `WithNotFoundExpiry(...)`。
当 `Do(...)` 返回该未找到错误时，jetcache 会缓存占位符。

## 自动刷新如何生效？

自动刷新是 key 级别的显式开启：

1. 在缓存配置中设置 `WithRefreshDuration(...)`，
2. 在 `Once(...)` 调用中传入 `cache.Refresh(true)`，
3. 并提供 `Do(...)`，让刷新任务可以持续回源更新。

## 自动刷新适合所有 key 吗？

不适合。它更适用于 key 少、回源成本高、实时性要求相对低的场景。

## 为什么没有流量了还在刷新？

请配置 `WithStopRefreshAfterLastAccess(...)`，让冷 key 自动停止刷新。

## 如何临时跳过本地缓存读取？

可以用 `GetSkippingLocal(...)`，或在 `Once(...)` 中传 `cache.SkipLocal(true)`。

## 本地缓存用 TinyLFU 还是 FreeCache？

- `TinyLFU`：通常是更通用的默认选择。
- `FreeCache`：内存边界更严格，热点路径 GC 负担低。

建议用业务数据模型做 `go test -bench ...` 对比。

## codec 应该怎么选？

- `msgpack`：默认，综合表现均衡。
- `json`：可读性、互操作更好。
- `sonic`：高性能 JSON 场景可选。

## 可以自定义 codec/local/remote/stats 吗？

可以。分别实现 `encoding.Codec`、`local.Local`、`remote.Remote`、`stats.Handler` 即可。

## 泛型 `MGet` 回源和本地失效同步分别从哪个版本开始？

- 泛型 `MGet` 回源函数 + pipeline 优化：`v1.1.0+`
- 更新后跨进程本地缓存失效：`v1.1.1+`

详见 [版本与功能可用性](Versioning.md)。

## 出现 `cache: both remote and local are nil` 是什么原因？

表示缓存实例没有可用后端。至少配置 `WithLocal(...)` 或 `WithRemote(...)` 之一。

## 是否必须调用 `Close()`？

建议必须调用，尤其开启自动刷新时。它会停止后台协程，避免资源泄漏。
每个缓存实例生命周期内应只调用一次。
