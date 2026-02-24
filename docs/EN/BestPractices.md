# Production Best Practices

This guide focuses on practical patterns for running `jetcache-go` in production.

## 1. Choose the Right Cache Topology

- Local-only: best for single-instance workloads or short-lived hot data.
- Remote-only: best for strict cross-instance consistency.
- Both (local + remote): recommended default for high-QPS APIs.

Rule of thumb:

- Read-heavy + high QPS: `both`
- Cross-instance coherence only: `remote`
- Extremely low latency and no shared state requirement: `local`

## 2. Key Design

- Use stable prefixes by domain: `user:profile:1001`.
- Keep keys deterministic and avoid embedding volatile fields.
- For generic cache wrappers (`NewT`), keep separator consistent (`WithSeparator`).

## 3. TTL Strategy

- Set business TTL by data freshness SLA, not by infrastructure defaults.
- Configure remote TTL with `WithRemoteExpiry(...)` and override hot keys with `TTL(...)`.
- Configure not-found TTL with `WithNotFoundExpiry(...)` to mitigate penetration.
- Keep random offset enabled (`WithOffset`) to avoid synchronized expiration.

## 4. Cache Penetration and Breakdown

- Configure `WithErrNotFound(err)` (for example `sql.ErrNoRows`) to cache placeholders.
- Use `Once(...)` + `Do(...)` for read path to benefit from singleflight.
- Avoid plain `Get(...)` + manual load in high-concurrency hot-key paths.

## 5. Auto-Refresh Safely

Use refresh only for:

- few keys,
- expensive loads,
- low realtime sensitivity.

Checklist:

- Set `WithRefreshDuration(...)` (for example 30s-5m).
- Set `WithStopRefreshAfterLastAccess(...)` to avoid idle-key waste.
- Set `WithRefreshConcurrency(...)` based on backend capacity.
- Call `Close()` on graceful shutdown.

Capacity estimation:

```text
max_refresh_keys ~= refreshDuration/loadCost * refreshConcurrency * instanceCount
```

## 6. Protect Backend Under Failure

- Keep local cache enabled in `both` mode for degraded reads.
- Use short request-level context timeouts on `Once(...)` Do functions.
- Keep retry policy in backend client, not inside cache callback loops.

## 7. Observability Baseline

- Always keep at least one stats handler enabled (`stats.NewStatsLogger`).
- Export Prometheus metrics in production.
- Alert on:
  - hit ratio drop,
  - remote miss surge,
  - query fail increase.

See [Monitoring](Monitoring.md).

## 8. Serialization Choices

- Default `msgpack` is a good general production default.
- Use `json` when interoperability/readability is prioritized.
- Validate payload size and schema compatibility before switching codec.

## 9. Batch Access Pattern

Use generic `MGet` for ID batch reads:

- built-in pipeline access to remote cache,
- singleflight on grouped misses,
- partial-result friendly degradation.

Prefer `MGetWithErr` when upstream must know partial-failure details.

## 10. Benchmark and Capacity Test

The repo contains benchmark cases in `bench_test.go`.

Run examples:

```bash
go test -run '^$' -bench 'BenchmarkOnce|BenchmarkMGet' -benchmem ./...
```

Recommended benchmark dimensions:

- local implementation (`TinyLFU` vs `FreeCache`),
- payload size (small/medium/large),
- miss ratio (1%, 10%, 50%),
- concurrency levels (e.g. 50/200/1000).

Store benchmark outputs in CI artifacts to track regressions over time.

## 11. Deployment Checklist

- Set explicit cache name via `WithName(...)`.
- Set `WithErrNotFound(...)` and not-found TTL.
- Ensure Redis timeout/pool configs are tuned.
- Ensure metrics and alert rules are active.
- Call `Close()` on process shutdown.
