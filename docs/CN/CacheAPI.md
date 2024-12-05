<!-- TOC -->
* [缓存接口](#缓存接口)
  * [Set 接口](#set-接口)
  * [Once 接口](#once-接口)
* [泛型接口](#泛型接口)
  * [MGet批量查询](#mget批量查询)
<!-- TOC -->


# 缓存接口

以下是 Cache 提供的接口，这些方法和签名和 [go-redis/cache](https://github.com/go-redis/cache) 基本一致。但某些接口我们提供了更强大的能力。

```go
// Set 通过 ItemOption 设置缓存
func Set(ctx context.Context, key string, opts ...ItemOption) error

// Once 通过 ItemOption 查询缓存。单飞模式、可开启缓存自动刷新
func Once(ctx context.Context, key string, opts ...ItemOption) error

// Delete 删除缓存
func Delete(ctx context.Context, key string) error

// DeleteFromLocalCache 删除本地缓存
func DeleteFromLocalCache(key string)

// Exists 判断缓存是否存在
func Exists(ctx context.Context, key string) bool

// Get 查询缓存，并将查询结果序列化到 val 
func Get(ctx context.Context, key string, val any) error

// GetSkippingLocal查询远程缓存（跳过本地缓存）
func GetSkippingLocal(ctx context.Context, key string, val any) error

// TaskSize 自动刷新缓存的任务数量（本实例本进程）
func TaskSize() int

// CacheType 缓存类型。共 Both、Remote、Local 三种类型
func CacheType() string

// Close 关闭缓存资源，当开启了缓存自动刷新且不再需要的时候，需要关闭
func Close()
```

## Set 接口

该接口用于设置缓存。它支持多种选项，例如设置值(`Value`)、过期时间(`TTL`)、回源函数(`Do`)、以及针对 `Remote` 缓存的原子操作。

函数签名：
```go
func Set(ctx context.Context, key string, opts ...ItemOption) error
```

参数：
- `ctx`: `context.Context`，请求上下文。用于取消操作或设置超时。
- `key`: `string`，缓存键。
- `opts`: `...ItemOption`，可变参数列表，用于配置缓存项的各种选项。 支持以下选项：
    - `Value(value any)`: 设置缓存值。
    - `TTL(duration time.Duration)`: 设置缓存项的过期时间。
    - `Do(fn func(context.Context) (any, error))`: 给定的回源函数 `fn` 来获取值，优先级高于 `Value`。
    - `SetNX(flag bool)`: 仅当键不存在时才设置缓存项。 适用于远程缓存，防止覆盖已存在的值。
    - `SetXX(flag bool)`: 仅当键存在时才设置缓存项。适用于远程缓存，确保只更新已存在的值。

返回值：
- `error`: 如果设置缓存失败，则返回错误。

示例1：使用 `Value` 设置缓存值
```go
obj := struct {
	Name string
	Age  int
}{Name: "John Doe", Age: 30}

err := cache.Set(ctx, key, Value(&obj), TTL(time.Hour))
if err != nil {
	// 处理错误
}
```

示例 2: 使用 `Do` 函数获取并设置缓存值
```go
err := cache.Once(ctx, key, TTL(time.Hour), Do(func(ctx context.Context) (any, error) {
	return fetchData(ctx)
}))
if err != nil {
	// 处理错误
}
```

示例 3: 使用 `SetNX` 进行原子操作 (仅当键不存在时设置)

```go
err := cache.Set(ctx, key, TTL(time.Hour), Value(obj), SetNX(true))
if err != nil {
    // 处理错误，例如键已存在
}
```

## Once 接口

该接口从缓存中获取给定 `Key` 的值，或者执行 `Do` 函数，然后将结果缓存并返回。它确保对于给定的 `Key`，同一时间只有一个执行在进行中。如果出现重复请求，重复的调用者将等待原始请求完成，并接收相同的结果。
通过设置 `Refresh(true)` 可开启缓存自动刷新。

`Once` 接口：分布式-防缓存击穿利器 - `singleflight`

![singleflight](/docs/images/singleflight.png)

> singlefilght ，在go标准库中（"golang.org/x/sync/singleflight"）提供了可重复的函数调用抑制机制。通过给每次函数调用分配一个key，
> 相同key的函数并发调用时，只会被执行一次，返回相同的结果。其本质是对函数调用的结果进行复用。


`Once` 接口：分布式-防缓存击穿利器 - `auto refresh`

![autoRefresh](/docs/images/autorefresh.png)

> `Once`接口提供了自动刷新缓存的能力，目的是为了防止缓存失效时造成的雪崩效应打爆数据库。对一些key比较少，实时性要求不高，加载开销非常大的缓存场景，
> 适合使用自动刷新。下面的代码指定每分钟刷新一次，1小时如果没有访问就停止刷新。如果缓存是redis或者多级缓存最后一级是redis，缓存加载行为是全局唯一的，
> 也就是说不管有多少台服务器，同时只有一个服务器在刷新，目的是为了降低后端的加载负担。

> 关于自动刷新功能的使用场景（“适用于按键数量少、实时性要求低、加载开销非常大的场景”），其中“按键数量少”需要进一步说明。
> 为了确定合适的按键数量，可以建立一个模型。例如，当 refreshConcurrency=10，有 5 台应用服务器，每个按键的平均加载时间为 2 秒，refreshDuration=30s 时，理论上可刷新的最大按键数量为 30 / 2 * 10 * 5 = 750。



函数签名：
```go
func Once(ctx context.Context, key string, opts ...ItemOption) error
```

参数：
- `ctx`: `context.Context`，请求上下文。用于取消操作或设置超时。
- `key`: `string`，缓存键。
- `opts`: `...ItemOption`，可变参数列表，用于配置缓存项的各种选项。 支持以下选项：
    - `Value(value any)`: 设置缓存值。
    - `TTL(duration time.Duration)`: 设置缓存项的过期时间。
    - `Do(fn func(context.Context) (any, error))`: 给定的回源函数 `fn` 来获取值，优先级高于 `Value`。
    - `SkipLocal(flag bool)`: 是否跳过本地缓存。
    - `Refresh(refresh bool)`: 是否开启缓存自动刷新。配合 Cache 配置参数 `config.refreshDuration` 设置刷新周期。

返回值：
- `error`: 如果设置缓存失败，则返回错误。

示例1：使用 `Once` 查询数据，并开启缓存自动刷新

```go
mycache := cache.New(cache.WithName("any"),
		// ...
		// cache.WithRefreshDuration 设置异步刷新时间间隔
		cache.WithRefreshDuration(time.Minute),
		// cache.WithStopRefreshAfterLastAccess 设置缓存 key 没有访问后的刷新任务取消时间
        cache.WithStopRefreshAfterLastAccess(time.Hour))

// `Once` 接口通过 `cache.Refresh(true)` 开启自动刷新
err := mycache.Once(ctx, key, cache.Value(obj), cache.Refresh(true), cache.Do(func(ctx context.Context) (any, error) {
    return fetchData(ctx)
}))
if err != nil {
    // 处理错误
}

mycache.Close()
```

# 泛型接口

```go
// Set 泛型设置缓存
func (w *T[K, V]) Set(ctx context.Context, key string, id K, v V) error

// Get 泛型查询缓存 (底层调用Once接口)
func (w *T[K, V]) Get(ctx context.Context, key string, id K, fn func(context.Context, K) (V, error)) (V, error)

// MGet 泛型批量查询缓存
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V)
```

## MGet批量查询

`MGet` 通过 `golang` 的泛型机制 + `Load` 函数，非常友好的多级缓存批量查询ID对应的实体。如果缓存是 `redis` 或者多级缓存最后一级是 `redis`，
查询时采用 `pipeline`实现读写操作，提升性能。查询未命中本地缓存，需要去查询Redis和DB时，会对Key排序，并采用单飞模式(`singleflight`)调用。
需要说明是，针对异常场景（IO异常、序列化异常等），我们设计思路是尽可能提供有损服务，防止穿透。

![mget](/docs/images/mget.png)

函数签名：
```go
func (w *T[K, V]) MGet(ctx context.Context, key string, ids []K, fn func(context.Context, []K) (map[K]V, error)) (result map[K]V)
```

参数：
- `ctx`: `context.Context`，请求上下文。用于取消操作或设置超时。
- `key`: `string`，缓存键。
- `ids`: `[]K`，缓存对象的ID。
- `fn func(context.Context, []K) (map[K]V, error)`：回源函数。用于给未命中缓存的ID去查询数据并设置缓存。

返回值：
- `map[K]V`: 返回有值键值对 `map`。
