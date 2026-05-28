# Swarmion manifest mismatch root cause

## Summary

The observed `manifest_mismatch` is not caused by the fresh Protos SQL schema or by migration history drift. It is caused by Swarmion deriving the swarm manifest initial boundary from the repository's current finalized head at startup when no explicit manifest is supplied.

That makes the swarm identity mutable across restarts. If one peer starts when the finalized head is "fresh DB with user only" and another later restarts after the VM rows are finalized, they compute different manifest digests even though the SQL schema is identical.

## Observed evidence

Local fatal marker:

```text
heartbeat manifest mismatch:
local=35e300c570d4a9fbc39ff76a5214ef61d0c343e9529f669bdb233cc71efe6b2d
remote=426e15df6ca030bb97ba7630116cd650aa986782d15327d63ccd033be8b4bfb6
remote_peer=12D3KooWCaeoKyXUMPk59QCWqx9XANKeUUXNEt3Xdnqq8UuqUNSh
derived_policy_digest=310d3d88149a813eced492829eacea4ad60ee0926a32f8b0ab7b3b8063bf7943
```

Remote fatal marker was the inverse: remote local digest was `426e15df...` and its observed peer digest was `35e300...`.

The merge policy digest matched on both peers:

```text
310d3d88149a813eced492829eacea4ad60ee0926a32f8b0ab7b3b8063bf7943
```

The migration applied on both sides was the fresh collapsed migration:

```text
protos_01_tables.sql checksum=58d258...
```

The manifest digests were reproducible from the different initial boundaries:

| Peer state | Initial root | Initial finalized commit | Manifest digest |
| --- | --- | --- | --- |
| Local after VM row existed | `v9qhld6goh53p6nou2sis47sb24rtjpc` | `a6lr3tsj7qvl31d1ujss9uvtg041k11l` | `35e300c570d4a9fbc39ff76a5214ef61d0c343e9529f669bdb233cc71efe6b2d` |
| Remote/fresh join boundary | `nm009lkt0trrb3ca8kepvoi4t1rjibkl` | `v20lel65t359pb07eua8v4ovkbig2e72` | `426e15df6ca030bb97ba7630116cd650aa986782d15327d63ccd033be8b4bfb6` |

The remote parentless boundary contained only the fresh local origin data: `users`, the local origin peer, and local device metadata. The local parentless boundary used later by the manifest contained those rows plus the VM/remote peer rows. That data difference is enough to change the finalized root and commit, and therefore the manifest digest.

The `.swarmion/protos.empty-root` and `.swarmion/protos.genesis-head` marker files matched on both machines, but those are not the manifest boundary being compared in heartbeats.

## Why a fresh schema can still mismatch

Swarmion's manifest digest is not just a schema digest. `protocol.SwarmManifest.Digest()` includes:

- manifest version
- `InitialRootHash`
- `InitialCommitID`
- schema digest
- merge policy manifest

So the schema can be fresh and identical while the manifest still differs because the initial finalized root/commit differ.

In this failure, the schema and merge policy were not the differentiators. The differentiator was the selected initial finalized boundary.

## Code path that creates the drift

Protos opens Swarmion with a fixed namespace:

```go
Namespace:      fmt.Sprintf(swarmionNamespaceTemplate, db.name),
AdminNamespace: fmt.Sprintf(swarmionAdminNamespaceTemplate, db.name),
```

It did not pass a persisted `SwarmManifest`.

In `swarmion/runtime/app/app.go`, startup normalizes the configured manifest. With no explicit initial boundary, the manifest has schema/merge policy identity but no `InitialRootHash` or `InitialCommitID`.

The namespace-specific auto-completion path only runs when the namespace is empty and there are no bootstrap peers:

```go
if configuredManifestDigest == ([32]byte{}) &&
    strings.TrimSpace(cfg.Namespace) == "" &&
    len(cfg.BootstrapPeers) == 0 {
    completeFoundationManifestFromRepository(...)
}
```

Because Protos supplies a namespace, this does not run.

Later, `foundationProtocolStateFromRepository` fills the missing manifest boundary from the current repository finalized head:

```go
_, initialRoot, err := a.repo.FinalizedHead(repoCtx)
initialCommit, err := a.repo.MaterializeFinalizedCheckpoint(...)
manifest.InitialRootHash = materializedRoot
manifest.InitialCommitID = initialCommit
manifestDigest, err := manifest.Digest()
```

That means each standalone startup can choose a different "initial" boundary if the repository finalized head has moved since the last startup.

In the observed run:

1. The remote peer joined while the local finalized state was the fresh DB/user-only boundary.
2. The remote computed and retained digest `426e15df...`.
3. The VM registration rows were then added and finalized.
4. The local daemon restarted without a persisted manifest.
5. Local startup recomputed the manifest using the newer finalized head containing VM rows.
6. Local computed digest `35e300...`.
7. Heartbeats compared `35e300...` against `426e15df...` and correctly entered `manifest_mismatch`.

## Contract mismatch wording

This is a protocol identity mismatch, not a Protos SQL contract mismatch. The fresh collapsed schema works as expected. The problem is that the protocol identity currently depends on a moving repository head unless the embedding application supplies and persists a completed manifest.

## Fix direction

The swarm identity needs a single immutable foundation boundary.

The practical fix is:

1. When Swarmion starts from a bootstrap state and the configured manifest has no initial boundary, complete the in-memory manifest from `protocol.FindInitialFinalizedBoundary(bootstrapState)`.
2. Expose the completed runtime manifest to the embedding application.
3. Have Protos persist the completed manifest under `.swarmion/protos.swarm-manifest.json`.
4. On every later Protos startup, load that persisted manifest and pass it into `swarmionapp.Config.SwarmManifest`.
5. When a configured manifest already has an initial boundary, Swarmion must use that boundary instead of rematerializing the current finalized head as a new initial boundary.
6. If the repository is deleted for a fresh start, Protos should also delete the persisted manifest to avoid pinning a stale swarm identity onto a new repo.

This preserves the declarative model: DB state still drives peer behavior, while the swarm manifest becomes stable protocol metadata for that DB instead of a value inferred imperatively from the latest local repository state.

## Questions for the Swarmion architect

- Should Swarmion itself own persistence of the completed `SwarmManifest`, or should embedding apps persist it after `Open()`?
- Should fixed namespaces be allowed with an incomplete manifest, or should Swarmion reject that and require either an explicit completed manifest or an empty namespace so it can derive one safely?
- Should the default schema digest continue to be `swarmion:default-schema:v1`, or should schema-engine callers be able to include the actual schema contract digest in the manifest?
- Should the startup fatal record include the local and remote initial boundary roots/commits, not just the digest, to make this class of mismatch immediately diagnosable?
