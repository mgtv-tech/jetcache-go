# Examples

This directory provides scenario-oriented examples.

## Index

- [Basic CRUD](BasicCRUD.md)
- [Microservice Pattern](Microservice.md)
- [High Concurrency Hot Key](HighConcurrency.md)

## Run Suggestions

- Start local Redis first.
- Use `go test ./...` to verify core behavior.
- Adapt key naming and TTL policy to your service SLA.
- Check feature availability when running older tags:
  - generic `MGet` load callback + pipeline optimization: `v1.1.0+`
  - cross-process local invalidation (`WithSyncLocal`): `v1.1.1+`
- See [Versioning](../Versioning.md).
