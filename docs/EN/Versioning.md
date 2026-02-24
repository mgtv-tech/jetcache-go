# Versioning and Feature Availability

This page clarifies when major user-facing capabilities became available.

## Feature Timeline

| Capability | Since | Notes |
| --- | --- | --- |
| Generic `MGet` supports load callback (`fn`) with remote pipeline optimization | `v1.1.0+` | In distributed cache scenarios, remote reads use pipeline-based `MGet` to reduce round trips. |
| Cross-process local cache invalidation after updates | `v1.1.1+` | Use `WithSyncLocal(true)` + `WithSourceId(...)` + `WithEventHandler(...)` to propagate invalidation events across processes/nodes. |

## Compatibility Guidance

- If your service is on `< v1.1.0`, avoid relying on generic `MGet` load callback behavior.
- If your service is on `< v1.1.1`, local cache invalidation must be implemented outside jetcache-go.
- For production upgrades, pin explicit module tags in `go.mod` and validate key read paths (`Once`, `MGet`, refresh, sync local).

## Documentation Scope

Main docs in this repository describe the latest stable behavior.
When your deployed version is older than the documented feature "since" version, follow your running tag behavior first.
