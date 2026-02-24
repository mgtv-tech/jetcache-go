# FAQ

## What is the default cache mode?

It depends on your config.

- `WithLocal` only: local mode.
- `WithRemote` only: remote mode.
- both set: two-level mode.

## Should I use `Get` or `Once`?

Use `Once` for standard cache-aside read path because it handles miss loading and singleflight.
Use `Get` when data must already be in cache and you do not want fallback loading.

## How do I avoid cache penetration?

Set `WithErrNotFound(err)` and optionally `WithNotFoundExpiry(...)`.
When `Do(...)` returns that not-found error, jetcache stores a placeholder entry.

## How does auto-refresh work?

Auto-refresh is key-level and opt-in:

1. configure `WithRefreshDuration(...)` in cache options,
2. pass `cache.Refresh(true)` in `Once(...)` call,
3. provide `Do(...)` so refresh can load fresh values.

## Is auto-refresh suitable for all keys?

No. It is for few keys with expensive load and lower realtime requirements.

## Why does refresh continue after traffic is gone?

Set `WithStopRefreshAfterLastAccess(...)` to stop idle key refresh tasks.

## How to bypass local cache for one read?

Use `GetSkippingLocal(...)` or `Once(..., cache.SkipLocal(true), ...)`.

## How do I choose local cache implementation?

- `TinyLFU`: generally a good default.
- `FreeCache`: strict memory control and no GC overhead in hot path.

Test with your own workload using `go test -bench ...`.

## Which codec should I use?

- `msgpack` is default and usually best balanced.
- `json` for readability/interoperability.
- `sonic` for high-performance JSON scenarios.

## Can I add custom codec/local/remote/stats?

Yes. Implement `encoding.Codec`, `local.Local`, `remote.Remote`, or `stats.Handler`.

## Which versions introduced generic `MGet` load and sync-local invalidation?

- Generic `MGet` load callback + pipeline optimization: `v1.1.0+`
- Cross-process local cache invalidation after updates: `v1.1.1+`

See [Versioning](Versioning.md).

## Why do I get `cache: both remote and local are nil`?

You created cache without usable backend. Configure at least one of `WithLocal(...)` or `WithRemote(...)`.

## Do I need to call `Close()`?

Yes, especially when refresh is enabled. It stops background goroutines and avoids leaks.
Call it once per cache instance lifecycle.
