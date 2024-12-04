<!-- TOC -->
* [Introduction](#introduction)
  * [LogStats Default Output Format](#logstats-default-output-format)
  * [Prometheus Plugin Visualization Dashboard](#prometheus-plugin-visualization-dashboard)
<!-- TOC -->

# Introduction

`jetcache-go` provides built-in `LogStats` and a `Prometheus` statistics plugin via the [jetcache-go-plugin](https://github.com/mgtv-tech/jetcache-go-plugin).


## LogStats Default Output Format

The default log output from LogStats follows this format:

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

## Prometheus Plugin Visualization Dashboard

![stats](/docs/images/stats.png)
