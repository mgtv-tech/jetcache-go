<!-- TOC -->
* [Cache Interface](#cache-interface)
  * [Set Interface](#set-interface)
  * [Once Interface](#once-interface)
* [Generic Interfaces](#generic-interfaces)
  * [MGet Bulk Query](#mget-bulk-query)
<!-- TOC -->


# Cache Interface

The following interface, provided by `Cache`, is largely consistent with [go-redis/cache](https://github.com/go-redis/cache). However, some interfaces offer enhanced capabilities.

```go
// Set sets cache using ItemOption.
func Set(ctx context.Context, key string, opts ...ItemOption) error

// Once retrieves cache using ItemOption.  Single-flight mode; automatic cache refresh can be enabled.
func Once(ctx context.Context, key string, opts ...ItemOption) error

// Delete deletes cache.
func Delete(ctx context.Context, key string) error

// DeleteFromLocalCache deletes the local cache.
func DeleteFromLocalCache(key string)

// Exists checks if cache exists.
func Exists(ctx context.Context, key string) bool

// Get retrieves cache and serializes the result to `val`.
func Get(ctx context.Context, key string, val any) error

// GetSkippingLocal retrieves remote cache (skipping local cache).
func GetSkippingLocal(ctx context.Context, key string, val any) error

// TaskSize returns the number of cache auto-refresh tasks (for this instance and process).
func TaskSize() int

// CacheType returns the cache type.  Options are `Both`, `Remote`, and `Local`.
func CacheType() string

// Close closes cache resources.  This should be called when automatic cache refresh is enabled and is no longer needed.
func Close()
```

## Set Interface

This interface is used to set cache entries. It supports various options, such as setting the value (`Value`), remote expiration time (`TTL`), a fetch function (`Do`), and atomic operations for `Remote` caches.

Function Signature:

```go
func Set(ctx context.Context, key string, opts ...ItemOption) error
```

Parameters:

- `ctx`: `context.Context`, the request context. Used for cancellation or timeout settings.
- `key`: `string`, the cache key.
- `opts`: `...ItemOption`, a variadic parameter list for configuring various options of the cache item.  The following options are supported:
  - `Value(value any)`: Sets the cache value.
  - `TTL(duration time.Duration)`: Sets the expiration time for the remote cache item. (Local cache expiration time is uniformly set when building the Local cache instance.)
  - `Do(fn func(context.Context) (any, error))`: Uses the given fetch function `fn` to retrieve the value; this takes precedence over `Value`.
  - `SetNX(flag bool)`: Sets the cache item only if the key does not exist.  Applicable to remote caches to prevent overwriting existing values.
  - `SetXX(flag bool)`: Sets the cache item only if the key exists. Applicable to remote caches to ensure only existing values are updated.

Return Value:

- `error`: Returns an error if setting the cache fails.

Example 1: Setting a cache value using `Value`

```go
obj := struct {
    Name string
    Age  int
}{Name: "John Doe", Age: 30}

err := cache.Set(ctx, key, cache.Value(&obj), cache.TTL(time.Hour))
if err != nil {
    // Handle error
}
```


Example 2: Retrieving and setting a cache value using the `Do` function

```go
err := cache.Once(ctx, key, cache.TTL(time.Hour), cache.Do(func(ctx context.Context) (any, error) {
    return fetchData(ctx)
}))
if err != nil {
    // Handle error
}
```

Example 3: Performing an atomic operation using `SetNX` (sets only if the key doesn't exist)

```go
err := cache.Set(ctx, key, cache.TTL(time.Hour), cache.Value(obj), cache.SetNX(true))
if err != nil {
    // Handle error, e.g., key already exists
}
```


## Once Interface

This interface retrieves the value associated with a given `key` from the cache. If a cache miss occurs, the `Do` function is executed, the result is cached, and then returned.  It ensures that for a given `key`, only one execution is in progress at any time. If duplicate requests occur, subsequent callers will wait for the original request to complete and receive the same result.  Automatic cache refresh can be enabled by setting `Refresh(true)`.

`Once` Interface: Distributed Cache – A Weapon Against Cache Piercing - `singleflight`

![singleflight](/docs/images/singleflight.png)

> `singleflight`, provided in the Go standard library ("golang.org/x/sync/singleflight"), offers a mechanism to suppress redundant function calls. By assigning a key to each function call, concurrent calls with the same key will only be executed once, returning the same result.  Essentially, it reuses the results of function calls.


`Once` Interface: Distributed Cache – A Weapon Against Cache Piercing - `auto refresh`

![autoRefresh](/docs/images/autorefresh.png)

> The `Once` interface provides the ability to automatically refresh the cache. This is intended to prevent cascading failures that can overwhelm the database when caches expire.  Automatic refresh is suitable for scenarios with a small number of keys, low real-time requirements, and very high loading overhead. The code below (Example 1) specifies a refresh every minute and stops refreshing after an hour without access. If the cache is Redis or a multi-level cache where the last level is Redis, the cache loading behavior is globally unique.  That is, regardless of the number of servers, only one server refreshes at a time to reduce the load on the backend.

> Regarding the use case for the auto-refresh feature ("suitable for scenarios with a small number of keys, low real-time requirements, and very high loading overhead"), the "small number of keys" requires further clarification. To determine the appropriate number of keys, a model can be established. For example, when `refreshConcurrency=10`, there are 5 application servers, the average loading time for each key is 2 seconds, and `refreshDuration=30s`, the theoretically maximum number of keys that can be refreshed is 30 / 2 * 10 * 5 = 750.


> Regarding the AutoRefresh feature's usage scenario ("Suitable for scenarios with few keys, low real-time requirements, and very high loading overhead"), "few keys" requires clarification.
> To determine an appropriate number of keys, you can create a model. For example, with refreshConcurrency=10, 5 application servers, an average loading time of 2 seconds per key, and refreshDuration=30s, the theoretical maximum number of refreshable keys is 30 / 2 * 10 * 5 = 750.

Function Signature:

```go
func Once(ctx context.Context, key string, opts ...ItemOption) error
```

Parameters:

- `ctx`: `context.Context`, the request context. Used for cancellation or timeout settings.
- `key`: `string`, the cache key.
- `opts`: `...ItemOption`, a variadic parameter list for configuring various options of the cache item.  The following options are supported:
  - `Value(value any)`: Sets the cache value.
  - `TTL(duration time.Duration)`: Sets the expiration time for the remote cache item. (Local cache expiration time is uniformly set when building the Local cache instance.)
  - `Do(fn func(context.Context) (any, error))`: Uses the given fetch function `fn` to retrieve the value; this takes precedence over `Value`.
  - `SkipLocal(flag bool)`: Whether to skip the local cache.
  - `Refresh(refresh bool)`: Whether to enable automatic cache refresh.  Works with the Cache configuration parameter `config.refreshDuration` to set the refresh interval.

Return Value:

- `error`: Returns an error if retrieving the cache fails.

Example 1: Querying data using `Once` and enabling automatic cache refresh

```go
mycache := cache.New(cache.WithName("any"),
    // ...
    // cache.WithRefreshDuration sets the asynchronous refresh interval
    cache.WithRefreshDuration(time.Minute),
    // cache.WithStopRefreshAfterLastAccess sets the time to cancel refresh tasks after a cache key has not been accessed
    cache.WithStopRefreshAfterLastAccess(time.Hour))

// The `Once` interface enables automatic refresh via `cache.Refresh(true)`
err := mycache.Once(ctx, key, cache.Value(obj), cache.Refresh(true), cache.Do(func(ctx context.Context) (any, error) {
    return fetchData(ctx)
}))
if err != nil {
    // Handle error
}

mycache.Close()
```


# Generic Interfaces

```go
// Set generically sets cache entries.
func (w *T[K, V]) Set(ctx context.Context, key string, id K, v V) error

// Get generically retrieves cache entries (underlying call to Once interface).
func (w *T[K, V]) Get(ctx context.Context, key string, id K, fn func(context.Context, K) (V, error)) (V, error)

// MGet generically retrieves multiple cache entries.
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V)
```

## MGet Bulk Query

`MGet`, leveraging Go generics and the `Load` function, provides a user-friendly mechanism for bulk querying entities by ID in a multi-level cache. If the cache is Redis or a multi-level cache where the last level is Redis, read/write operations are performed using pipelining to improve performance. When a cache miss occurs in the local cache and a query to Redis and the database is required, the keys are sorted, and a single-flight (`singleflight`) call is used.  It's important to note that for exceptional scenarios (I/O errors, serialization errors, etc.), our design prioritizes providing a degraded service to prevent cache penetration.

![mget](/docs/images/mget.png)

Function Signature:

```go
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V)
```

Parameters:

- `ctx`: `context.Context`, the request context. Used for cancellation or timeout settings.
- `key`: `string`, the cache key.
- `ids`: `[]K`, the IDs of the cache objects.
- `fn func(context.Context, []K) (map[K]V, error)`: The fetch function. Used to query data and set the cache for IDs that miss the cache.

Return Value:

- `map[K]V`: Returns a map of key-value pairs with values.
