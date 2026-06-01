# Swarmion deprovision feedback

Date: 2026-05-30

## Context

This note records the Swarmion-facing issues found while testing Protos VM deprovisioning after the libp2p init work.

The Protos-side intent is:

1. A new VM starts `protosd` in init mode.
2. It initializes from the deploying peer over libp2p.
3. The VM becomes a normal Protos peer and may become witness-eligible.
4. When the VM is deleted, Protos removes the VM from cloud resources, local peer state, and Swarmion witness eligibility.

## What Protos changed locally

Protos deletion now asks the local Swarmion runtime to remove the deleted peer from witness eligibility before deleting cloud resources and DB peer rows.

The local sequence is:

1. If the peer is an active witness, build the remaining witness candidate formation and apply a Swarmion witness transition that excludes the peer.
2. Confirm the peer is no longer active.
3. If the peer is still eligible, publish a rank-zero witness update for that subject.
4. Confirm the peer is no longer eligible.
5. Continue provider deletion, peer removal, and DB cleanup.

Protos also filters user-facing runtime `StateProviders` through the current Protos peer DB. This avoids presenting stale Swarmion control-plane provider cache entries as current product peers after deprovisioning.

## Swarmion behavior that caused confusion

Swarmion witness eligibility is durable protocol state. Removing a peer from the Protos DB, bootstrap list, or cloud provider does not implicitly remove Swarmion eligibility.

That behavior is correct for protocol safety, but it means embedding products need an explicit deprovision sequence. Omitting a peer from a later candidate set is not enough, because `ApplyWitnessCandidates` publishes missing positive ranks for requested candidates; it does not publish rank-zero tombstones for peers absent from the request.

## Swarmion API feedback

### 1. Add a first-class peer deprovision API

The product currently has to combine several low-level calls:

- `PreviewWitnessCandidates`
- `ApplyWitnessCandidates`
- `PublishWitnessRankUpdateForSubject(peer, 0)`
- status polling

It would be safer if Swarmion exposed a single operation like:

```go
PreparePeerRemoval(ctx, peerID, remainingFormation)
```

or a documented helper that:

1. detects whether the peer is active;
2. applies the required transition if needed;
3. publishes the rank-zero tombstone if authorized;
4. reports whether the peer is safe to remove from app peer tables and transport.

### 2. Make rank-zero tombstone confirmation explicit

After publishing rank zero, the embedding app needs to know when the tombstone is finalized and visible at the current boundary. A structured result would be better than requiring callers to re-read status and infer success from `EligibleWitnessIDs`.

Useful result fields:

- subject peer ID;
- tombstone event ID;
- finalized root/commit where the tombstone is visible;
- whether the local caller was authorized as subject or root owner;
- whether the peer remains active and still requires a witness transition.

### 3. Expose or clean stale operational peer caches

`StateProviders` currently behaves like an operational cache. It can continue to report a deleted peer because the control plane records providers it has seen and only removes them in narrow cases such as unsupported provider errors.

That may be fine internally, but product-facing status needs one of these:

- an explicit `ForgetPeer(peerID)` / `EvictPeer(peerID)` API that clears provider and compatibility caches;
- TTL or connected/current-only semantics for `StateProviders`;
- separate fields for historical providers versus currently usable providers.

Without that distinction, applications can confuse stale operational memory for desired product membership.

### 4. Bootstrap should not synthesize finalized commit identity

For VM init, Protos expects the new VM to initialize from the remote peer, not create an independent Protos genesis.

Swarmion does hydrate from bootstrap state, but the fresh repository path still creates local Dolt storage state before hydration. The important requirement is that any advertised finalized commit identity must be imported or reproduced exactly. A fresh peer should not synthesize a different local Dolt commit for an advertised finalized boundary.

Recommended Swarmion direction:

1. Include finalized commit IDs, roots, and commit object availability in the bootstrap manifest.
2. Fetch/import the exact finalized Dolt commit closure from the bootstrap peer before materializing protocol state.
3. Have `EnsureFinalizedLineageMaterialized` reuse already-imported finalized commits instead of reconstructing them locally.
4. If exact commit import is unavailable, fail bootstrap as an incompatible or incomplete history instead of silently creating a different local commit for the same boundary.

## Protos follow-up

The Protos-side deprovision path is now stricter: if Swarmion cannot safely move an active witness out of the active set or cannot tombstone eligibility, deletion should fail before the cloud VM is removed. That is intentional. A force-local cleanup path can still be added later, but it should be visibly unsafe and should not pretend protocol deprovision succeeded.
