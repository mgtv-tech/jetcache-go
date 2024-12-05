<!-- TOC -->
* [Cache API](#cache-api)
  * [Set API](#set-api)
  * [Once API](#once-api)
* [Generic API](#generic-api)
  * [Batch Query with MGet](#batch-query-with-mget)
<!-- TOC -->


# Cache API

Below are the interfaces provided by Cache. These methods and signatures are similar to go-redis/cache, but we offer enhanced capabilities in certain interfaces.

```go
// Set caches data using ItemOptions
func Set(ctx context.Context, key string, opts ...ItemOption) error

// Once queries the cache using ItemOptions. Fly mode, enables automatic cache refreshing
func Once(ctx context.Context, key string, opts ...ItemOption) error

// Delete deletes cached data
func Delete(ctx context.Context, key string) error

// DeleteFromLocalCache deletes data from the local cache
func DeleteFromLocalCache(key string)

// Exists checks if the cache exists
func Exists(ctx context.Context, key string) bool

// Get queries the cache and serializes the result into val
func Get(ctx context.Context, key string, val any) error

// GetSkippingLocal queries the remote cache (skips the local cache)
func GetSkippingLocal(ctx context.Context, key string, val any) error

// TaskSize returns the number of tasks for automatic cache refreshing (in this instance/process)
func TaskSize() int

// CacheType returns the type of cache. It can be Both, Remote, or Local
func CacheType() string

// Close closes the cache resources. It is necessary to close when automatic cache refreshing is enabled and no longer needed
func Close()
```

## Set API
This interface is used to set cache values. It supports various options such as setting the value (Value), expiration time (TTL), data source function (Do), and atomic operations for Remote cache.

Function Signature:
```go
func Set(ctx context.Context, key string, opts ...ItemOption) error
```

Parameters:

- ctx: context.Context, the request context used for cancellation or setting timeouts.
- key: string, the cache key.
- opts: ...ItemOption, variadic parameter list for configuring various options of the cache item. It supports the following options:
  - Value(value any): Set the cache value.
  - TTL(duration time.Duration): Set the expiration time of the cache item.
  - Do(fn func(context.Context) (any, error)): Use the given data source function fn to fetch the value, taking precedence over Value.
  - SetNX(flag bool): Set the cache item only if the key does not exist. Useful for Remote cache to prevent overwriting existing values.
  - SetXX(flag bool): Set the cache item only if the key exists. Useful for Remote cache to ensure updating only existing values.

Return Value:
- error: Returns an error if setting the cache fails.

Example 1: Setting cache value using Value

```go
obj := struct {
	Name string
	Age  int
}{Name: "John Doe", Age: 30}

err := cache.Set(ctx, key, Value(&obj), TTL(time.Hour))
if err != nil {
    // Handle error
}
```

Example 2: Fetching and setting cache value using `Do` function

```go
err := cache.Once(ctx, key, TTL(time.Hour), Do(func(ctx context.Context) (any, error) {
	return fetchData(ctx)
}))
if err != nil {
	// Handle error
}
```

Example 3: Performing an atomic operation using `SetNX` (set only if key does not exist)

```go
err := cache.Set(ctx, key, TTL(time.Hour), Value(obj), SetNX(true))
if err != nil {
    // Handle error, for example, key already exists
}
```

## Once API

This interface retrieves the value for a given Key from the cache or executes the `Do` function, caches the result, and returns it. It ensures that only one execution is in progress at a time for a given Key. If duplicate requests occur, the duplicate callers will wait for the original request to complete and receive the same result.
By setting `Refresh(true)`, automatic cache refreshing can be enabled.

Once API : Distributed tool against cache breakdown - `singleflight`

![singleflight](/docs/images/singleflight.png)

> singleflight, provided in the Go standard library ("golang.org/x/sync/singleflight"), offers a mechanism to suppress repeated function calls. By assigning a key to each function call, when concurrent calls to the same function with the same key occur, it will only be executed once and return the same result. Essentially, it reuses the results of function calls.


Once API: Distributed tool against cache breakdown - `auto refresh`

![autoRefresh](/docs/images/autorefresh.png)

> The `Once` API provides the capability for automatic cache refreshing, aiming to prevent the cascading effect caused by cache expiration leading to overwhelming database requests. It is suitable for scenarios where there are relatively few keys, low real-time requirements, and high loading costs for cache loading.
> The following code specifies refreshing every minute and stopping refreshing after 1 hour of no access. If the cache is Redis or the last level of a multi-level cache is Redis, and the cache loading behavior is globally unique, meaning only one server is refreshing at a time regardless of the number of servers, the goal is to reduce the backend loading burden.

> Regarding the AutoRefresh feature's usage scenario ("Suitable for scenarios with few keys, low real-time requirements, and very high loading overhead"), "few keys" requires clarification.
> To determine an appropriate number of keys, you can create a model. For example, with refreshConcurrency=10, 5 application servers, an average loading time of 2 seconds per key, and refreshDuration=30s, the theoretical maximum number of refreshable keys is 30 / 2 * 10 * 5 = 750.

Function Signature:

```go
func Once(ctx context.Context, key string, opts ...ItemOption) error
```

Parameters:

- ctx: context.Context, the request context used for cancellation or setting timeouts.
- key: string, the cache key.
- opts: ...ItemOption, variadic parameter list for configuring various options of the cache item. It supports the following options:
  - Value(value any): Set the cache value.
  - TTL(duration time.Duration): Set the expiration time of the cache item.
  - Do(fn func(context.Context) (any, error)): Use the given data source function fn to fetch the value, taking precedence over Value.
  - SkipLocal(flag bool): Whether to skip the local cache.
  - Refresh(refresh bool): Enable automatic cache refreshing. Combined with the Cache configuration parameter config.refreshDuration to set the refresh interval.
  - Return Value:

error: 
- Returns an error if setting the cache fails.

Example 1: Using `Once` to query data and enabling automatic cache refreshing

```go
mycache := cache.New(cache.WithName("any"),
		// ...
		// Set the asynchronous refresh interval
        cache.WithRefreshDuration(time.Minute),
		// Set the time to stop refreshing the cache key after it has not been accessed
        cache.WithStopRefreshAfterLastAccess(time.Hour))

// Enable automatic refreshing using `cache.Refresh(true)` with the `Once` interface
err := mycache.Once(ctx, key, cache.Value(obj), cache.Refresh(true), cache.Do(func(ctx context.Context) (any, error) {
    return fetchData(ctx)
}))
if err != nil {
    // Handle error
}

mycache.Close()
```

# Generic API

```go
// Set generic cache setting
func (w *T[K, V]) Set(ctx context.Context, key string, id K, v V) error

// Get generic cache query (underlying calls to the Once interface)
func (w *T[K, V]) Get(ctx context.Context, key string, id K, fn func(context.Context, K) (V, error)) (V, error)

// MGet generic batch cache query
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V)
```

## Batch Query with MGet

`MGet` provides a user-friendly way to perform bulk queries for entities corresponding to IDs through the combination of Go's generic mechanism and the Load function. If the cache is Redis or the last level of a multi-level cache is Redis, the query is implemented using `pipeline` for read and write operations to enhance performance. In cases where the local cache is missed and queries need to be made to Redis and the database, keys are sorted, and the `singleflight` mode is used for invocation.

It is important to note that for exceptional scenarios (such as IO errors, serialization issues, etc.), our design philosophy aims to provide a service that degrades gracefully to prevent penetration.

![mget](/docs/images/mget.png)

Function Signature:

```go
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V)
```

Parameters:

- ctx: context.Context, the request context used for cancellation or setting timeouts.
- key: string, the cache key.
- ids: []K, IDs of the cached objects.
- fn func(context.Context, []K) (map[K]V, error): Data source function. Used to query data for IDs not found in the cache and set the cache.

Return Value:
- map[K]V: Returns a map of key-value pairs with values.
