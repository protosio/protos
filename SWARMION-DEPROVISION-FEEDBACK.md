# Swarmion Deprovision Feedback

This note tracks the current Protos-side expectations for removing a peer after
the Swarmion checkpoint/provider API cleanup.

## Current Contract

Swarmion no longer exposes product-managed consensus roles. Protos keeps its
own replication-priority metadata in SQL, but that metadata is not pushed into a
Swarmion role-management API.

When Protos deprovisions a device, it should:

1. Bring the local runtime to the latest checkpoint boundary.
2. Ask Swarmion to evict the peer from runtime peer state.
3. Verify the peer is not still reported as a state provider.
4. Remove the Protos device rows and cloud resources.

The important safety boundary is checkpoint convergence, not role membership.
If Swarmion cannot reach the current checkpoint or still reports the peer as a
state provider, Protos should fail the normal deprovision path before deleting
the cloud VM.

## Remaining Product Questions

- Whether Swarmion should expose a stronger structured result from peer
  eviction, instead of requiring Protos to re-read runtime status.
- Whether peer eviction should include a clear "safe to remove cloud resource"
  signal when the peer recently served checkpoint or content data.
- Whether the runtime status surface should separate temporary provider lag from
  fatal deprovision blockers.
