<!-- TOC -->
* [介绍](#介绍)
  * [LogStats 日志默认输出如下格式信息：](#logstats-日志默认输出如下格式信息)
  * [Prometheus 统计插件可视化大盘](#prometheus-统计插件可视化大盘)
<!-- TOC -->

# 介绍

`jetcache-go` 默认提供了内嵌 `LogStats` 及 [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin) 提供的`Prometheus`统计插件。

## LogStats 日志默认输出如下格式信息：

```shell
2024/09/25 18:45:49 jetcache-go stats last 1ms.
cache                   |         qpm|   hit_ratio|         hit|        miss|       query|  query_fail
------------------------+------------+------------+------------+------------+------------+------------
any                     |           2|      50.00%|           1|           1|           1|           1
any_local               |           2|      50.00%|           1|           1|           -|           -
any_remote              |           2|      50.00%|           1|           1|           -|           -
test_lang_cache_0       |           2|      50.00%|           1|           1|           1|           1
test_lang_cache_0_local |           2|      50.00%|           1|           1|           -|           -
test_lang_cache_0_remote|           2|      50.00%|           1|           1|           -|           -
test_lang_cache_1       |           2|      50.00%|           1|           1|           1|           1
test_lang_cache_1_local |           2|      50.00%|           1|           1|           -|           -
test_lang_cache_1_remote|           2|      50.00%|           1|           1|           -|           -
test_lang_cache_2       |           2|      50.00%|           1|           1|           1|           1
test_lang_cache_2_local |           2|      50.00%|           1|           1|           -|           -
test_lang_cache_2_remote|           2|      50.00%|           1|           1|           -|           -
------------------------+------------+------------+------------+------------+------------+------------
```

## Prometheus 统计插件可视化大盘

![stats](/docs/images/stats.png)
