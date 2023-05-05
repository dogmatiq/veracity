<div align="center">

# Veracity

Veracity is an event-sourced [Dogma](https://github.com/dogmatiq/dogma)
[engine](https://github.com/dogmatiq/dogma#engine) with a focus on horizontal
scalability and "shardability" of data.

[![Documentation](https://img.shields.io/badge/go.dev-documentation-007d9c?&style=for-the-badge)](https://pkg.go.dev/github.com/dogmatiq/veracity)
[![Latest Version](https://img.shields.io/github/tag/dogmatiq/veracity.svg?&style=for-the-badge&label=semver)](https://github.com/dogmatiq/veracity/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/dogmatiq/veracity/ci.yml?style=for-the-badge&branch=main)](https://github.com/dogmatiq/veracity/actions/workflows/ci.yml)
[![Code Coverage](https://img.shields.io/codecov/c/github/dogmatiq/veracity/main.svg?style=for-the-badge)](https://codecov.io/github/dogmatiq/veracity)

</div>

- interop api
  - discoverspec - serves list of running applications
  - eventstreamspec - serves events in real-time
    - update to support partitioning / multiple streams per app
    - mesh router - routes API requests to appropriate node
- cluster
  - registry - queries which nodes are present
  - sharding - allocates "work" to specific nodes (rendezvous hashing)
    - aggregate instances
    - process instances
    - projection "consumers" (one per event stream per projection)
  - cluster api - gRPC API used to communicate between nodes in the same cluster
- persistence
  - postgres - implement journal and kv store
  - dynamodb - kv store
- command executor
- aggregates
- processes
- integrations
- projections
