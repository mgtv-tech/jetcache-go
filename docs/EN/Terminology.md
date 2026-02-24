# Terminology

This page standardizes core terms used in `jetcache-go` docs.

| Term | Definition |
| --- | --- |
| Cache penetration | Repeated requests for non-existing data bypass cache and hit backend. |
| Cache breakdown (hot-key breakdown) | A hot key expires and high concurrency causes burst load to backend. |
| Cache avalanche | Many keys expire/fail in a short period and overload backend. |
| Singleflight | One key, one in-flight loader execution; concurrent callers share result. |
| Auto-refresh | Background refresh for selected keys to reduce expiration shock. |
| Degradation | Fallback behavior to keep partial service when cache/backend fails. |
| Two-level cache | Local cache + remote cache layered read/write path. |
| Placeholder (not-found placeholder) | Sentinel value for not-found result to prevent penetration. |

Related docs:

- [CacheAPI](CacheAPI.md)
- [Config](Config.md)
- [BestPractices](BestPractices.md)
- [Troubleshooting](Troubleshooting.md)
