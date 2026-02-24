# 系统架构

`jetcache-go` 是一个分层缓存框架，面向 cache-aside 读路径与高并发保护。

## 分层视图

```mermaid
flowchart TB
    App["业务应用"] --> Cache["Cache API"]
    Cache --> Local["本地缓存 (TinyLFU/FreeCache)"]
    Cache --> Remote["远程缓存 (Redis)"]
    Cache --> SF["Singleflight"]
    SF --> Loader["Do(...) 回源函数"]
    Loader --> DB["数据库 / 上游服务"]
    Cache --> Refresh["刷新调度器"]
    Refresh --> Loader
    Cache --> Stats["统计处理器"]
```

## 缓存模式

- `local`：仅进程内缓存，延迟最低，但不跨节点共享。
- `remote`：仅远程缓存，跨节点一致性更强。
- `both`：本地 + 远程，适合高 QPS 读接口。

## 读路径（`Once`）

```mermaid
flowchart TD
    A["请求"] --> B{"本地命中?"}
    B -- 是 --> Z["返回"]
    B -- 否 --> C{"远程命中?"}
    C -- 是 --> D["回填本地"] --> Z
    C -- 否 --> E["Singleflight"]
    E --> F["仅一次执行 Do(...)"]
    F --> G{"回源结果"}
    G -- 成功 --> H["写入远程/本地"] --> Z
    G -- NotFound --> I["写入 not-found 占位符"] --> Z
    G -- 错误 --> J["返回错误"]
```

关键点：读路径应优先使用 `Once(...) + Do(...)`，让并发 miss 合并为一次回源。

## Singleflight 行为

```mermaid
sequenceDiagram
    participant C1 as 调用方 1
    participant C2 as 调用方 2
    participant SF as Singleflight
    participant DB as 后端

    C1->>SF: Once(key)
    C2->>SF: Once(key)
    SF->>DB: 回源一次
    DB-->>SF: 返回值
    SF-->>C1: 返回值
    SF-->>C2: 共享同一结果
```

## 自动刷新行为

自动刷新是 key 级别显式开启（`cache.Refresh(true)`）。

```mermaid
sequenceDiagram
    participant T as 调度器
    participant N1 as 节点 A
    participant L as Redis 分布式锁
    participant DB as 后端

    T->>N1: 触发刷新
    N1->>L: 尝试加锁
    alt 加锁成功
        N1->>DB: 拉取最新值
        N1->>N1: 更新缓存
    else 加锁失败
        N1->>N1: 跳过本轮
    end
```

仅建议对少量热点且回源代价高的 key 开启刷新。

## 组件职责

| 组件 | 职责 | 内置实现 |
| --- | --- | --- |
| `local.Local` | 进程内缓存 | `TinyLFU`、`FreeCache` |
| `remote.Remote` | 共享缓存后端 | `go-redis/v9` 适配器 |
| `encoding.Codec` | 序列化 | `msgpack`、`json`、`sonic` |
| `stats.Handler` | 指标统计 | logger、Prometheus 插件 |
| `singleflight` | miss 合并 | `x/sync/singleflight` |
| 刷新调度器 | 周期更新 | 内置实现 |

## 设计要点

- key 规则在读写两端必须一致。
- `WithErrNotFound(...)` 要与真实数据源 not-found 错误对齐。
- 开启刷新时建议配置 `WithStopRefreshAfterLastAccess(...)`。
- 服务优雅退出时调用 `Close()` 停止后台任务。
