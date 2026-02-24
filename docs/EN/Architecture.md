# System Architecture

`jetcache-go` is a layered cache framework for cache-aside reads and high-concurrency protection.

## Layered View

```mermaid
flowchart TB
    App["Application"] --> Cache["Cache API"]
    Cache --> Local["Local Cache (TinyLFU/FreeCache)"]
    Cache --> Remote["Remote Cache (Redis)"]
    Cache --> SF["Singleflight"]
    SF --> Loader["Do(...) Loader"]
    Loader --> DB["Database / Upstream"]
    Cache --> Refresh["Refresh Scheduler"]
    Refresh --> Loader
    Cache --> Stats["Stats Handler"]
```

## Cache Modes

- `local`: in-process only, lowest latency, no cross-node sharing.
- `remote`: shared cache only, stronger cross-node consistency.
- `both`: local + remote, recommended for high-QPS read APIs.

## Read Path (`Once`)

```mermaid
flowchart TD
    A["Request"] --> B{"Local hit?"}
    B -- Yes --> Z["Return"]
    B -- No --> C{"Remote hit?"}
    C -- Yes --> D["Write local"] --> Z
    C -- No --> E["Singleflight"]
    E --> F["Run Do(...) once"]
    F --> G{"Result"}
    G -- Success --> H["Write remote/local"] --> Z
    G -- NotFound --> I["Write not-found placeholder"] --> Z
    G -- Error --> J["Return error"]
```

Key point: use `Once(...) + Do(...)` on read paths so concurrent misses collapse to one backend call.

## Singleflight Behavior

```mermaid
sequenceDiagram
    participant C1 as Caller 1
    participant C2 as Caller 2
    participant SF as Singleflight
    participant DB as Backend

    C1->>SF: Once(key)
    C2->>SF: Once(key)
    SF->>DB: Load once
    DB-->>SF: Value
    SF-->>C1: Value
    SF-->>C2: Shared value
```

## Auto-Refresh Behavior

Refresh is key-level and opt-in (`cache.Refresh(true)`).

```mermaid
sequenceDiagram
    participant T as Scheduler
    participant N1 as Node A
    participant L as Redis Lock
    participant DB as Backend

    T->>N1: trigger refresh
    N1->>L: try lock
    alt lock acquired
        N1->>DB: load latest value
        N1->>N1: update caches
    else lock denied
        N1->>N1: skip this round
    end
```

Use refresh only for a small set of hot keys with expensive loaders.

## Component Responsibilities

| Component | Responsibility | Built-in Choice |
| --- | --- | --- |
| `local.Local` | In-process cache | `TinyLFU`, `FreeCache` |
| `remote.Remote` | Shared cache backend | `go-redis/v9` adapter |
| `encoding.Codec` | Serialization | `msgpack`, `json`, `sonic` |
| `stats.Handler` | Metrics emission | logger, Prometheus plugin |
| `singleflight` | Miss coalescing | `x/sync/singleflight` |
| refresh scheduler | Periodic update | built-in |

## Design Notes

- Keep key format deterministic across writers/readers.
- Keep `WithErrNotFound(...)` aligned with real datastore not-found errors.
- Enable `WithStopRefreshAfterLastAccess(...)` when refresh is on.
- Call `Close()` on graceful shutdown to stop background tasks.
