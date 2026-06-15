# Swarmion Checkpoint Provider Feedback

This note replaces the older transition-role feedback. The current integration
path treats Swarmion as a checkpointing and provider runtime; Protos no longer
authors or reconciles Swarmion role membership.

## Product Boundary

Protos still stores device class and replication priority locally. That data is
used for product policy and diagnostics, while Swarmion owns checkpoint
materialization, state-provider status, content repair, and peer eviction.

The mixed local/cloud end-to-end test should therefore assert:

1. Local and cloud peers converge on the same checkpoint root.
2. Runtime state reports usable state providers.
3. Deprovision catches up to a checkpoint before removing peer state.
4. Protos replication metadata persists without being interpreted as Swarmion
   role membership.

## Open API Questions

- A structured "checkpoint caught up" result would make product deprovision
  flows easier to reason about.
- Peer eviction could return whether the target still has provider obligations
  or content-transfer work in progress.
- Runtime status should keep provider compatibility and fatal checkpoint errors
  explicit so products do not need to infer them from logs.
