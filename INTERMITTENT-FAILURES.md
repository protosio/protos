# Intermittent Failure Ledger

This file tracks intermittent, suspicious, or reliability-relevant behavior seen
while running Protos tests and e2e workflows. Track issues here even when they
are outside the current fix so they can be revisited later with evidence.

## Architectural Patterns

The repeated failures mostly point to convergence and control-plane fragility
around the declarative-to-imperative boundary. Image transfer bugs, app runtime
stalls, checkpoint mismatches, peer cleanup delays, and APIC deadlines are not
fully separate classes of problems: they often appear when the system is
changing topology, publishing declarative state, reconciling imperative work,
and trying to prove checkpoint durability at the same time.

### Declarative-to-imperative writes must not over-block

Several failures came from operations that had already published state, or had
already caused the correct imperative action, but then failed because the local
node could not immediately prove checkpoint visibility or task feedback
durability. Desired-state writes and runtime feedback should generally return at
the published boundary, while explicit checkpoint waits should be reserved for
verification, cleanup, and other places where durability is the operation being
requested.

### Peer reachability belongs in one lower layer

Peer connectivity, topology class, address reachability, connection state, and
capability checks need one canonical owner. The image resolver, APIC clients,
DB transport, and admin paths should not each infer whether a peer is usable
from raw addresses or libp2p connectedness on their own. Local macOS
`192.168.64.x` addresses being host-reachable but not guest-reachable, and
`network.Limited` connections being valid for some peer APIs, are examples of
topology knowledge that should stay in the peer-management layer.

### Data transfer must not starve control APIs

Large image transfers repeatedly coincided with APIC/status deadlines,
task-watch resets, and slow app-state observation. Bulk transfer should be
isolated from control-plane APIs through bounded streams, explicit backpressure,
resumable chunks, and health checks that do not compete with image blob
transfer for the same fragile connection path.

### Cleanup is an idempotent lifecycle, not one request

Deletion remains a long-running imperative pocket. Provider stop latency,
durable peer removal, host-agent network reconfiguration, and task-watch
deadlines all recur during cleanup. Cleanup should keep being modeled as an
idempotent async lifecycle with explicit phases, bounded waits, stale
observation handling, and resumable reconciliation rather than as a single
request that must hold all control-plane dependencies healthy until completion.

### Diagnostics must not depend on the failing channel

Several RCAs were delayed because protosd logs or VM-side state were difficult
to retrieve once P2P/APIC connectivity failed. Host-agent owned diagnostics
should be available for local VMs even when peer connectivity is broken, and
guest protosd logs should be retrievable through a path independent of the
control channel being investigated.

## Code Verification (2026-06-19)

The fixes recorded below were verified against the working tree. **Verdict: the
ledger is accurate.** Every item marked `fixed in code` is actually present and
matches its description, usually with a backing unit test. No RCA blamed the
wrong subsystem. The notes below refine the *status* of several items: in
multiple cases the fix is real and closes the **observed** path, but a sibling
path to the **same symptom** remains open. These should be read as "fixed
(observed path)" rather than "root cause closed".

Code references use `path:line` against the `core` module.

### Residual issues — fix present, symptom still reachable

1. **Reconcile dedupe race is narrowed, not closed (still reaches
   `stopped (running)`).** The completion-time signature refresh was correctly
   removed; the signature is captured at queue time
   (`internal/app/manager.go:234`) and a regression test exists
   (`internal/app/manager_test.go:156`). But a second path to the identical
   symptom remains: `Notify()` (`internal/app/manager.go:205`) advances
   `am.reconcileSig` to the new signature **before** enqueuing
   (`shouldQueueReconcile` line 234), then `QueueReconcile`
   (`internal/app/manager.go:276`) calls `tasks.EnqueueUnique`, which **dedupes
   against a still-running task and returns `inserted=false` — a bool that is
   discarded** (line 214). When `StartApp` flips desired `stopped→running`
   while the original stopped-state reconcile is mid-flight (a multi-minute
   window during image transfer), the signature advances but no task carrying
   the running state is ever queued. The success path never clears the signature
   (only the error path does, lines 307/311) and there is **no periodic notifier
   for the app manager** (`internal/protosd/protosd.go:402` wraps only
   `dbNotifier`), so nothing recovers it. Recommended fix: only advance
   `reconcileSig` when `EnqueueUnique` reports `inserted=true`, and/or add a
   periodic app-manager safety-net notifier. This is the most important open
   item — it is a latent **hard failure**, and the recurring `stopped (running)`
   symptom across several RCAs points at this edge-triggered convergence model.

2. **DB operation lock still spans a network wait on the error/publish path.**
   The success-path checkpoint wait now runs **after** `unlock()`
   (`internal/db/db.go` `committedWriteContext`, ~2175) — correct and the main
   win. But the retryable-error path calls `resetAfterRetryableWriteError →
   catchUpCheckpointStrict` (a network catch-up, bounded 15s, up to 20 attempts)
   **while still holding `opMu`** (~2154/2197), and the publish phase
   (`swarmion_commit_info`, ~2170) remains under the lock. So the stated goal
   ("the operation lock protects local SQL/staging only") is met for the common
   path but not strictly; under write-error contention, readers (APIC/task
   watchers) can still stall. Recommended: move the retry-path reset/catch-up
   off `opMu`, mirroring the success path.

3. **P2P transfer throughput is structurally serial, not only relay-limited.**
   The ledger attributes ~180 KB/s to the macOS WireGuard relay hairpin (line
   372). That is a real factor, but the limited-relay code path is serial by
   construction: 512 KiB slices (`imageBlobLimitedStreamChunkBytes`,
   `internal/p2p/images.go:30`), **one in flight, concurrency forced to 1** for
   all-limited peers, a **fresh raw stream opened per slice**
   (`host.NewStream` per `downloadImageBlobRangeFromCandidate`), and a
   server-side `Flush` per frame. For a 28 MB image that is ~56 sequential
   stream setups, each paying a full relay RTT before any data flows — low
   throughput on *any* high-latency path, relay or not. Direct peers get 4 MiB
   chunks × 4 parallel ranges × 3 blobs; limited peers get 512 KiB × 1 × 1. This
   is the dominant remaining functional risk to e2e stability (transfers take
   ~158s, close to the harness deadline). Recommended: reuse a single stream
   across slices, pipeline/prefetch the next slice, and raise the in-flight
   window for limited peers.

4. **Peer address ordering is deprioritization, not exclusion.** Connectedness
   semantics are genuinely centralized in one predicate `usablePeerConnectedness`
   (`internal/p2p/p2p.go:348`), and no other package inspects connectedness —
   this realizes the "one lower layer" goal for *semantics*. But the host-only
   `192.168.64.x` address is still merged into one `AddrInfo` that libp2p races
   concurrently, and is still propagated to the DB transport. libp2p does not
   honor slice order as priority, so "internal addresses ahead of provider
   addresses" is largely cosmetic. Stronger would be to *exclude* host-only
   addresses for guest-to-guest dials, or attach a lower-layer reachability
   signal, rather than ordering them last.

5. **Cleanup is idempotent step-wise but not self-resumable.** Phases are
   explicit, bounded, and observable (durable-removal and provider-stop waits are
   capped and report progress), and steps tolerate already-done state — good. But
   the delete task is enqueued with `MaxAttempts: 1`
   (`internal/provisioners/deployment_tasks.go:246`), so a failed delete is **not**
   self-resumable; recovery requires a fresh `RemoveInstance` call. The
   architecture goal ("resumable reconciliation rather than a single request") is
   only partially met. Recommended: allow bounded auto-retry or per-phase
   progress checkpointing so a delete that fails mid-phase resumes itself.

6. **Diagnostics still degrade to SSH-only for the case that matters most.** The
   host-agent fallback and protosd log mirroring are real (`core/Dockerfile` tees
   to `/var/log/protos.log` with a `/dev/console` writability probe and stdout
   fallback). But when `/dev/console` is not writable (the observed
   `Operation not permitted`), protosd logs land only in the guest
   `/var/log/protos.log`, which the host-side `console.log` surface does **not**
   capture. So for a P2P-unreachable local VM — the exact postmortem case — the
   "independent of the failing channel" guarantee degrades to SSH-only.
   Recommended: have the host agent pull `/var/log/protos.log` into its own
   diagnostics surface for local VMs.

### Bugs found in passing (not previously tracked)

- **`GetAllImages` returns at most one image.** The result map is keyed by
  `image.id`, which is never assigned, so every iteration overwrites the `""`
  key (`internal/runtime/containerd_linux.go:336`). Standalone correctness bug,
  cheap fix; compare `GetImage` (~180) which sets `id`.
- **`Stop()` checks the wrong not-found sentinel** —
  `runtime.ErrContainerNotFound` where `GetSandbox` actually returns
  `ErrSandboxNotFound` (`internal/app/app.go:211`). Mostly masked today by
  `GetStatus` swallowing not-found, but a real inconsistency.
- **Image readiness blob check is size-only, not digest.** `contentBlobReady`
  (`internal/runtime/containerd_image_content_linux.go`) compares
  `reader.Size() != desc.Size` but does not re-verify the digest, so a
  correct-length-but-corrupt blob passes (mitigated by digest verification at
  ingest, but readiness itself is not an integrity check). Separately,
  `verifyImageSnapshot` creates and deletes a throwaway snapshot on **every**
  `ImageExistsLocally` call and only logs delete failures, so
  `protos-image-check-*` snapshots can leak under churn.
- **Resume is asymmetric.** Only the limited-relay path resumes per-slice; the
  direct/parallel path fully restarts a blob on any single range failure
  (`downloadImageBlob` truncate-and-reseek in `internal/p2p/images.go`). The
  more expensive restart is on the path the ledger does not flag.

### Test-coverage gaps

The two most consequential fixes are validated only indirectly (predicate-level
unit tests plus e2e runs), not directly:

- No unit test exercises the **resumable slice retry / reconnect loop**
  (`downloadImageBlobLimitedRange` / `ensureImagePeerConnected`); resume
  correctness rests on e2e runs alone.
- No direct **read-your-writes test for the `*Published` boundary**; the
  read-after-write tests use the durable variant. Given the boundary fix is
  central, add a test that an `InsertPublished`/`UpdatePublished` is locally
  readable immediately after return.

## Priorities (2026-06-19)

Ordered by impact on e2e reliability and by whether they cause hard failures vs.
recurring noise. Effort is rough.

### P0 — close the remaining hard-failure paths

1. **Reconcile dedupe residual race** (Residual #1). Latent `stopped (running)`
   hang; cheap fix (gate `reconcileSig` advance on `inserted=true`, plus a
   safety-net notifier). Add a regression test for the "notify during a running
   task" interleaving. *Low effort, high value.*
2. **P2P limited-path throughput** (Residual #3). The dominant reliability risk
   on passing runs — slow transfer consumes the harness deadline and combines
   with any other slow phase to fail. Reuse the stream across slices, pipeline
   slices, raise the in-flight window. *Medium/high effort, high value.*

### P1 — control-plane responsiveness under load

3. **Narrow the DB operation lock** (Residual #2). Move the retry-path
   network catch-up and the publish wait off `opMu`. Removes a recurring source
   of reader/task-watch starvation that has caused cleanup deadlines.
   *Medium effort.*
4. **Fix `GetAllImages` empty-key bug.** Clear correctness defect, trivial fix;
   audit callers for reliance on the buggy single-entry behavior. *Low effort.*

### P2 — robustness, observability, hygiene

5. **Cleanup self-resumability** (Residual #5): bounded auto-retry or per-phase
   checkpointing for the delete task.
6. **Host-agent protosd-log capture for unreachable local VMs** (Residual #6):
   pull `/var/log/protos.log` into the host-agent diagnostics surface so
   postmortems do not depend on SSH/P2P.
7. **Peer reachability: exclude host-only addresses** (Residual #4) for
   guest-to-guest dials instead of merely ordering them last.
8. **Image readiness hardening:** optional digest verification on readiness;
   ensure `verifyImageSnapshot` cleans up its probe snapshot (or gate the probe
   so it is not run on every hot-path call).
9. **`Stop()` not-found sentinel** fix.
10. **Test coverage:** resumable slice retry/reconnect, and `*Published`
    read-your-writes.

## Fixes Applied (2026-06-19)

Implementation pass over the priorities above, organized around three
architectural boundaries the failures kept crossing: the declarative-to-
imperative boundary was too synchronous, peer reachability leaked upward, and
the control plane and data plane interfered under load. All changes below build
for darwin and linux (`CGO_ENABLED=0`, pure-Go tags) and the full `go test ./...`
suite passes.

### Declarative-to-imperative boundary made less synchronous

- **Reconcile dedupe race closed (P0).** `Notify` now advances the reconcile
  signature only when `EnqueueUnique` actually inserts a fresh task
  (`internal/app/manager.go`); a notification that is deduplicated against an
  in-flight reconcile no longer consumes the new desired state. Regression test
  `TestReconcileNotifyDuringRunningTaskStillQueuesFollowup`
  (`internal/app/manager_test.go`) reproduces the mid-flight-notification case
  that previously hung at `stopped (running)`.
- **App-runtime convergence decoupled from the control plane.** The app manager
  now has its own periodic notifier (`internal/protosd/protosd.go`,
  `StartPeriodicNotifier(n.AppManager, 5s)`) instead of relying on the database
  notifier, whose path runs checkpoint catch-up and network/peer reconfiguration
  first and bails under cleanup/load. This is the safety net that recovers any
  reconcile dropped while a task was in flight, independent of control-plane
  health.
- **DB operation lock narrowed to local work (P1).** The retryable-write reset is
  split (`internal/db/db.go`): the local working-set reset
  (`resetWorkingSetForRetry`) stays under `opMu`, while the network checkpoint
  catch-up (`catchUpCheckpointForRetry`) now runs after `unlock()`. Combined with
  the already-correct success-path wait, the operation lock no longer spans a
  network wait on any path, so APIC reads and task watchers are not starved
  during write retries. (The success-path published-write boundary was already
  correct and is unchanged.)

### Control plane and data plane separated under load

- **P2P limited-peer throughput (P0).** The limited/relay blob transfer now reuses
  a single raw data-plane stream across all of a blob's resumable slices instead
  of opening a fresh stream per 512 KiB slice (`internal/p2p/images.go`,
  `limitedBlobStream` + looped `serveImageBlobStream`). This amortizes the relay
  connection setup that dominated the ~180 KB/s measurements while preserving
  slice-granular resumability: a failed slice resets the stream, reconnects
  through the peer layer, and retries from the same offset. The server flushes
  once per range instead of once per frame. The data plane stays isolated from
  the gRPC control channel (dedicated `imageBlobStreamProtocol`). New test
  `TestServeImageBlobStreamServesSequentialRangesOverReusedStream` covers
  multi-range reuse. A per-slice stream deadline now also bounds a stalled relay
  read.

### Cleanup modeled as a resumable lifecycle

- **Self-resumable delete (P2).** The task engine gained bounded auto-retry: a
  stream that opts into `MaxAttempts > 1` is requeued to pending (not terminally
  failed) while attempts remain (`internal/tasks/tasks.go`, `requeueForRetry`).
  The instance delete lifecycle now sets `MaxAttempts = 3`
  (`internal/provisioners/deployment_tasks.go`), so a delete that fails on a
  transient control-plane error (APIC deadline, host-agent network reconfigure)
  resumes itself on the next task tick instead of requiring a fresh
  `RemoveInstance`. Streams that keep `MaxAttempts = 1` still fail terminally.
  Tests `TestTaskRetriesUntilSuccessWithinMaxAttempts` and
  `TestTaskFailsTerminallyWhenMaxAttemptsIsOne` cover both paths.

### Standalone correctness fixes

- **`GetAllImages` empty-key bug fixed** (`internal/runtime/containerd_linux.go`):
  the result map was keyed by an always-empty `id`; it now sets `id = img.Name()`
  so each image gets a distinct key.
- **`Stop()` not-found sentinel fixed** (`internal/app/app.go`): now matches
  `runtime.ErrSandboxNotFound` via `errors.Is`, consistent with the rest of the
  file, instead of the unrelated `runtime.ErrContainerNotFound` code.

### Peer reachability ownership

- Documented `knownPeerIPs` (`internal/p2p/p2p.go`) as the single owner of peer
  transport-address ordering (overlay-first; provider/host-only address last) so
  the image resolver, DB transport, and admin paths consume one ordered list
  rather than infer reachability independently. The dial *set* was intentionally
  left unchanged: the ordering was tuned empirically against the local-macOS
  topology, which is not exercisable from this environment, so hard exclusion of
  host-only `192.168.64.x` addresses is deferred to avoid regressing
  host-to-guest bootstrap (cloud public IPs are unaffected either way).

### Deferred (with rationale)

- **Image-readiness hardening** (digest re-verification on the hot path;
  `verifyImageSnapshot` probe cost/leak) and **host-agent protosd-log capture for
  unreachable local VMs** were deferred. Both live in `//go:build linux` runtime
  / host-agent code that cannot be run-tested in this environment, and both are
  performance/hygiene/debuggability concerns rather than correctness bugs (the
  readiness check is already substantially correct). Adding digest re-hashing to
  every `ImageExistsLocally` call would also regress the hot path. These remain
  P2 follow-ups.

### Cloud e2e validation (run `1781864478190071000`)

Ran the mixed-cloud e2e end-to-end against real cloud instances with all three
provider images rebuilt from the patched source (Scaleway EFI ISO, Hetzner BIOS
raw, local macOS mactest). Topology: 2 local macOS VMs (arm64) + 1 Hetzner VM
(`ash`, amd64) + 1 Scaleway VM (`fr-par-1`, amd64).

- **Result: `passed`** (summary `core/.tmp/mixed-cloud-e2e-summary.json`,
  10:21:18Z to 10:38:47Z, ~17.5 min).
- **P2P image resolution verified**: seed `hetzner-vm` to puller `scaleway-vm`,
  `cloud P2P image resolution verified`, `remote P2P image label observed`
  (`image.source="p2p"`), no registry fallback and no `image.source=""`.
- **No `stopped (running)` hang**: the pull app showed `stopped (running)` for a
  single poll and then immediately reported `running (running)` - the
  reconcile-signature fix plus the independent app-runtime periodic notifier
  drove convergence instead of hanging to deadline. `stopped (running)` occurs
  exactly once in the whole log.
- **Bidirectional app connectivity** verified (HTTP 200 both directions over the
  WireGuard overlay).
- **Clean teardown**: all four instances deleted, every post-deletion checkpoint
  assertion passed, all providers/images/instances `removed`/`deleted`, and no
  leftover VM runner processes or cloud resources (no ongoing spend).
- **No** `DeadlineExceeded`, `RST_STREAM`, `published write did not reach local
  checkpoint`, panic, or durable-peer-removal stall in the run. One recoverable
  remote-runtime checkpoint-root mismatch occurred and converged on the next
  poll (the known, non-fatal checkpoint-view churn).

Scope note: cloud VMs have direct public-IP connectivity, so libp2p used the
direct/parallel blob path - the **limited-relay stream-reuse throughput change is
not exercised by this cloud topology** (it targets the local macOS relay-hairpin
`network.Limited` path). That change is covered by
`TestServeImageBlobStreamServesSequentialRangesOverReusedStream` and would be
measured by the local two-VM image-registry e2e; throughput numbers from the
limited path are still a follow-up to capture there. The probe image here is
small (6.4 MiB, 5 blobs), so this run validates correctness/convergence/cleanup
and P2P resolution on real cloud instances rather than relay throughput.

## Local VM Direct Networking (Design A / "Path B") — 2026-06-19 (in progress)

### Goal and why

The recurring local-macOS relay symptoms (slow `network.Limited` transfer, the
limited-relay throughput work) all trace to one root cause: **the two local
macOS VMs cannot reach each other on the underlay**, so they fall back to a
libp2p circuit relay through the laptop. There are actually two stacked relays:

1. A **WireGuard hub** (the darwin `hairpinTun`): VM1 encrypts inter-VM traffic
   for the *host's* key, the host decrypts, the hairpin loops the inner packet,
   the host re-encrypts for VM2. Necessary today, but it puts the host in the
   crypto path (no e2e VM↔VM) and pays a double-encrypt + host-bandwidth cost.
2. A **libp2p circuit relay** through the laptop: redundant, and the source of
   the `network.Limited` throttling.

If the VMs could reach each other directly on the underlay, WireGuard would form
a single-hop e2e tunnel (VM1 encrypts for VM2's key, no host hub), libp2p's
existing overlay dial would form a `network.Connected`, and **both relays
disappear for local** — making the limited-relay throughput optimization moot
and restoring e2e encryption.

### Why direct underlay isn't possible today

Apple's `VZNATNetworkDeviceAttachment` (used in
`internal/localvm/runner_darwin.go:configureNetwork`) does not give reliable
VM↔VM connectivity on the `192.168.64.x` network — host↔VM works, VM↔VM does
not. The WG peer config encodes this: a NAT-attached node routes other local VMs
through the host relay endpoint (`module_linux.go:319-327`) rather than as direct
peers.

### Chosen direction

Run the local VMs on a **shared `vmnet` network (isolation off)** so they share
one L2 segment and can reach each other, attaching each VM via the public
`VZFileHandleNetworkDeviceAttachment` (needs only the `com.apple.security.virtualization`
entitlement we already have; **no Apple developer account / no `com.apple.vm.networking`**).
`vmnet` shared mode requires **root**, which the host-agent already has. Then the
WG module takes its existing direct-peer branch (`module_linux.go:328-344`) and
the hairpin/relay path is retired for local.

### Investigation findings (VM stack + vz binding `github.com/tmc/apple@v0.6.3`)

- VMs are run by our own Go code (`internal/localvm/runner_darwin.go`) via the
  purego `tmc/apple/virtualization` binding; LinuxKit only builds images. The
  network attachment is fully ours to change. Parent host-agent spawns one child
  `protos-hostagent --run-vm` per VM (`internal/hostagent/daemon/vm.go:247`,
  plain `cmd.Start()`, no fd passing yet).
- Binding exposes `VZFileHandleNetworkDeviceAttachment` (public, no entitlement)
  and a raw `vmnet` framework binding (shared/host modes, `vmnet_read`/`_write`,
  `vmnet_enable_isolation_key`, event callbacks). It is **purego**, so it stays
  `CGO_ENABLED=0`-compatible.
- `VZVmnetNetworkDeviceAttachment` exists but is **private API**
  (`objc.GetClass("VZVmnetNetworkDeviceAttachment")`) and likely entitlement-gated
  — set aside. `VZBridgedNetworkDeviceAttachment` needs `com.apple.vm.networking`
  — set aside.

### Material finding that gates the approach (decision point)

The vmnet binding wraps the **function entrypoints only**. It ships **no xpc
package, no examples, and no working start/read/write sequence**. A pure
in-process vmnet client therefore means hand-writing, via purego:
the xpc interface-description dictionary (to set the isolation key),
`vmnet_start_interface` with a completion block + dispatch queue, result-dict
parsing (assigned MAC/MTU/subnet/max-packet-size), the packets-available event
callback, and `vmnet_read`/`vmnet_write` with correct `vmpktdesc`/`iovec` struct
layout. That is ~200-400 lines of intricate ABI-sensitive interop that **cannot
be validated without root + real VMs** (the agent cannot run sudo
non-interactively), with a slow rebuild-mactest-and-run loop per iteration.

`socket_vmnet` already implements exactly this vmnet client, debugged and proven
(Lima/Colima/Rancher). Using it (root daemon owns the shared vmnet L2; each VM
attaches via `VZFileHandleNetworkDeviceAttachment` with a small stream↔dgram
reframing shim) keeps Design A's architecture while avoiding the hand-rolled
purego vmnet client. The first unknown either way — **does an isolation-off
shared vmnet actually give VM↔VM on this macOS version** — still needs an
empirical spike before investing in the full datapath.

Decision: hand-rolled in-process vmnet (the host-agent already has a NOPASSWD
sudoers entry for `bin/protos-hostagent`, so the root test loop can be driven
directly via `sudo bin/protos-hostagent --vmnet-selftest`).

### Progress: vmnet interop de-risked (selftest passes)

`internal/localvm/vmnet_darwin.go` hand-rolls the missing interop via purego: an
xpc dictionary shim (`xpc_dictionary_create/set_*/get_*`), reading the vmnet xpc
key strings from the framework symbols (the binding mis-types them), and
`vmnet_start_interface`/`vmnet_stop_interface` with the completion handler parsed
inside the callback (xpc params are only valid for the callback's duration).
Exposed as `protos-hostagent --vmnet-selftest`. Run as root it opens an
**isolation-off shared** interface and reports:

```
mtu=1500 max_packet=1514 mac=62:a6:9b:2b:10:8a
subnet_mask=255.255.255.0 dhcp_start=192.168.64.1 dhcp_end=192.168.64.254
```

Two consequences:
1. The hardest, most-uncertain interop (xpc dict + start + result parsing) is
   proven on this macOS version.
2. The shared subnet is **the same 192.168.64.0/24, gateway .1** as the current
   VZNAT, so existing static-IP allocation and gateway detection still apply -
   the IP/WG changes shrink to flipping the WG module to its direct-peer branch.

### Revised (simpler) architecture

vmnet shared mode is a single host-wide L2 switch, so every isolation-off
interface created on it shares that switch. That means each **child**
`--run-vm` process can be fully self-contained:

1. open its own isolation-off shared vmnet interface,
2. create a `socketpair(AF_UNIX, SOCK_DGRAM)`, attach the VM via the public
   `VZFileHandleNetworkDeviceAttachment` on one end,
3. pump frames 1:1 between vmnet and the other socket end.

No parent-owned interface, no `ExtraFiles` fd passing, no MAC-demux - a 1:1 pump
per VM. Both VMs land on the shared switch with isolation off and can reach each
other directly. (If two isolation-off interfaces turn out not to forward to each
other, the fallback is a single parent-owned interface with a demux pump; the
2-VM e2e is the deciding test.)

### Progress: per-child pump + VZFileHandle attachment implemented

`internal/localvm/vmnet_pump_darwin.go` adds `configureSharedVMNetNetwork`: open
an isolation-off shared vmnet interface, `socketpair(AF_UNIX, SOCK_DGRAM)`,
attach the VM via `VZFileHandleNetworkDeviceAttachment` on one end (MTU + MAC
from vmnet), and a 1:1 `vmnetPump` (vmnet->VM via the PACKETS_AVAILABLE event
callback + `vmnet_read`; VM->vmnet via a blocking read loop + `vmnet_write`).
`runner_darwin.go` `configureNetwork` now dispatches NAT (default) vs shared, and
threads a network cleanup through `buildVMConfig`/`Run`. Selection is via
`PROTOS_LOCAL_NET=vmnet-shared` or a flag file (incl. `/tmp/protos-local-net-mode`
so it survives sudo's env stripping; temporary test affordance). Builds clean for
darwin and linux; `go vet` flags only the expected FFI `unsafe.Pointer` uses.

The shared-vmnet datapath is **host-side only** (the VM runner), so validating it
needs no guest-image rebuild - the local 2-VM e2e (which rebuilds the host-agent
from source) exercises it directly.

Validation strategy (staged):
- Stage 1 (no guest change): run the local 2-VM e2e with the flag on and check
  the VMs get network (pump works) and that the previously-failing **direct
  dials over `192.168.64.x`** (DB peer `:10501`, libp2p `:10500`) now succeed -
  proving VM-to-VM underlay connectivity. If libp2p dials the peer's
  `192.168.64.x` directly it may go `network.Connected` over the raw underlay
  immediately, without any WireGuard change.
- Stage 2 (guest change, rebuild mactest): flip the WG module to the direct-peer
  branch when shared-vmnet is active (signal via an appended kernel cmdline
  param), so VM-to-VM app traffic is single-hop WireGuard-encrypted and the
  hairpin/relay is retired.

### Stage 1 run 1: found a binding struct bug (SEGV), fixed

First local 2-VM run with the flag on crashed the `--run-vm` child with
`SIGSEGV fault=0x9` inside the pump. Root cause: the `tmc/apple/vmnet` binding's
`Vmpktdesc` struct has the **wrong field order**. Apple's C layout is
`{size_t vm_pkt_size; struct iovec *vm_pkt_iov; uint32_t vm_pkt_iovcnt; uint32_t vm_flags;}`,
but the binding declares `Vm_pkt_iov` last. So C reads the iov pointer from the
bytes Go placed `vm_pkt_iovcnt(=1)`/`vm_flags(=0)` into → pointer value `0x1` →
dereferences `iov_len` at `0x1+8` → fault `0x9` (exact match).

Fix: bypass the binding's `Vmnet_read`/`Vmnet_write` and call the C symbols
directly via purego with a correctly-ordered `vmpktdesc`/`iovec` and an `int32`
packet count (C `int`). Validated via an extended `--vmnet-selftest` that writes a
broadcast frame and reads without crashing before re-running the e2e.

### Stage 1 PASSED (run `1781871003004453000`)

Extended `--vmnet-selftest` confirmed the struct fix (`write OK`, `read OK,
frames_read=6`, no crash). The local 2-VM image-registry e2e then **passed with
the shared-vmnet flag on**:

- Both VMs deployed `status=running (reachable)` (VM1 `192.168.64.192`, VM2
  `192.168.64.194`) - the pump carries the VMs' traffic.
- Both apps reached `running`; `remote image content ready ... blobs=11`;
  `remote P2P image label observed`; **`Protos P2P image resolution verified:
  seed=VM1 puller=VM2`** - the two local VMs transferred the image directly to
  each other over P2P.
- **Zero** `no route to host` / `connection refused :10501` / `i/o timeout`
  errors (previously a recurring class of noise) - the direct `192.168.64.x`
  VM-to-VM dials now succeed. No `limited=true` relay blob path was logged.
- Clean teardown: both instances deleted, no leftover VM-runner/daemon
  processes. Transient `stopped (running)` recovered immediately (reconcile fix).

**Conclusion: isolation-off shared vmnet gives real VM-to-VM underlay
connectivity**, and that alone makes libp2p connect the two local VMs directly -
fixing the relay/throughput class of problems for the local-macOS topology
without any WireGuard change. (The precise direct-vs-`Limited` connectedness and
throughput numbers live in the guest protosd debug logs, not the harness output;
the 0-dial-failures + successful P2P are the strong signal.)

### What remains (Stage 2, optional)

Stage 1 fixes the libp2p p2p path (which dials the raw `192.168.64.x` underlay).
VM-to-VM **app traffic over the WireGuard overlay** still uses the host hub,
because the guest WG module (`module_linux.go`) still sees `192.168.64.x` and
takes the relay branch. Stage 2 flips it to the direct-peer branch when
shared-vmnet is active (signalled via an appended kernel cmdline param), giving
single-hop WireGuard-encrypted VM-to-VM app traffic and retiring the hairpin for
local. This needs a mactest image rebuild (guest-side change).

### Productionization TODO (before this is default)

- Replace the `/tmp/protos-local-net-mode` test toggle with a manifest field set
  by the provisioner (the parent writes the manifest; the child reads it).
- Confirm the `go vet` `unsafe.Pointer` warnings are acceptable to `task quality`
  (they are expected FFI uses) or annotate them.
- Decide default: keep NAT default + opt-in, or flip to shared-vmnet default once
  Stage 2 lands and soaks.

### Stage 2 implemented (validating)

Guest-side WireGuard now takes the direct-peer branch when shared-vmnet is
active, so VM-to-VM app traffic is single-hop, end-to-end-encrypted, and the
host hairpin/relay is retired for local:

- Host (`runner_darwin.go` `configureLinuxBootLoader`): appends
  `protos.localnet=vmnet-shared` to the guest kernel cmdline when shared-vmnet is
  requested.
- Guest (`module_linux.go`): `sharedVMNetActive()` reads `/proc/cmdline`; when set,
  `relayLocalMacOSPeers` is forced false and the `isLocalMacOSNATIP` relay branch
  is skipped, so a `192.168.64.x` peer falls through to the normal direct-peer
  config (`Endpoint = <peer 192.168.64.x>:wgPort`) - a single-hop WireGuard tunnel
  encrypted for the peer's key (no host in the crypto path).

Builds clean darwin+linux; wireguard unit tests pass. Validation: rebuild mactest
(guest change) + re-run the local 2-VM e2e with the flag on. A pass confirms the
direct WireGuard overlay works (if it didn't, the VMs couldn't reach each other
over the overlay and the run would fail).

### Stage 2 PASSED (run `1781874752625768000`)

Rebuilt mactest with the guest WG change, re-ran the local 2-VM image-registry
e2e with the flag on:

- Both VMs `status=running (reachable)` (VM1 `192.168.64.192`, VM2
  `192.168.64.193`); `Protos P2P image resolution verified: seed=VM1 puller=VM2`.
- **Zero** `through host relay` log lines (and zero `no route to host` /
  `:10501 refused` / `i/o timeout`): the guest WG module took the direct-peer
  branch, yet the VMs still reached each other - confirming **single-hop direct
  WireGuard** VM-to-VM (host hub / hairpin retired for local).
- Both delete tasks `succeeded`; no leftover VM-runner processes.

**Path B complete and validated end-to-end.** Local macOS VMs now share an
isolation-off vmnet switch, reach each other directly at the underlay (libp2p
goes direct), and run single-hop end-to-end-encrypted WireGuard between
themselves with no host in the crypto path. The relay/hairpin and the
limited-relay throughput path are no longer needed for the local topology.

### Remaining productionization (before making this the default)

1. Replace the `/tmp/protos-local-net-mode` test toggle with a manifest field set
   by the provisioner (parent writes manifest, child reads it); drop the
   world-readable flag file.
2. Consider whether to retire the now-unused darwin `hairpinTun` and the
   relay-mode WG branch once shared-vmnet is the default, or keep them as the
   fallback for the NAT path.
3. `task quality`: confirm/annotate the expected FFI `unsafe.Pointer` vet warnings
   in `vmnet_darwin.go`.
4. Report the `Vmpktdesc` field-order bug upstream to `github.com/tmc/apple`.
5. Pump hardening for production: bound the event-callback worker (avoid blocking
   the dispatch queue on a slow VM socket), and add metrics/logging for dropped
   frames.

### Old network code removed — shared vmnet is the only local path

By decision, the NAT/hairpin/relay fallback and the flag were deleted; shared
vmnet is now the sole local-VM network architecture (not opt-in). Removed:

- **Host runner** (`runner_darwin.go`): the `configureNetwork` dispatcher,
  `configureNATNetwork` (VZNAT attachment), the cmdline-signal append, and the
  flag (`sharedVMNetRequested`, `PROTOS_LOCAL_NET`, the flag files incl.
  `/tmp/protos-local-net-mode`). `buildVMConfig` calls
  `configureSharedVMNetNetwork` directly.
- **Guest WG** (`module_linux.go`): the `isLocalMacOSNATIP` relay/roaming branch,
  the relay vars, and the now-dead helpers `localMacOSNATAttached`,
  `isLocalMacOSNATIP`, `localMacOSNATGateway`, `sharedVMNetActive`,
  `peerEndpoints`, `copyUDPAddr`. Local `192.168.64.x` peers always use the
  direct-peer path.
- **Host WG hairpin**: deleted `hairpin_tun_darwin.go` (+ test); `declarativePeers`
  no longer computes `hairpinRoutes`; the TUN is used directly (no
  `hairpinTun` wrapper); `hairpinDevice` field and all uses removed.

Kept intentionally: `provisioners.localMacOSNATGateway` / `originBootstrapIPs` -
that is bootstrap addressing for reaching the host/origin at `192.168.64.1`,
which is still valid (and reachable) under shared vmnet; it is not relay/hairpin
code.

Builds clean darwin+linux; wireguard/app/tasks unit tests pass; a repo-wide grep
for every removed symbol is empty. Re-validated by a local 2-VM e2e with **no
flag set** (run `1781877866270397000`): rebuilt mactest with the new guest,
deleted `/tmp/protos-local-net-mode`, ran the e2e -> **passed (rc=0)**, no crash,
`Protos P2P image resolution verified`, **0** dial-failures / `through host
relay`, both instances deleted, no leftover processes. Shared vmnet is now the
default and only local-VM network path, with the old NAT/hairpin/relay code and
the flag fully removed.

### Validation scope (what is and isn't covered)

Validated with the shared-vmnet flag on: the **local 2-VM image-registry e2e**
(Stage 1 and Stage 2). That covers VM provisioning, the pump, **local VM-to-VM**
connectivity, app deploy, **P2P image transfer between the two local VMs**, app
reachability, and cleanup.

NOT yet validated with the flag on (deferred follow-up): the **full mixed-cloud
e2e** (2 local + Hetzner + Scaleway). The earlier passing cloud e2e ran before
this work and without the flag (local VMs on the old NAT path). So the
**shared-vmnet local-VM <-> cloud-VM** path is unproven:
- internet egress from a shared-vmnet local VM (vmnet shared mode does NAT, so it
  is expected to work, but unconfirmed);
- local<->cloud WireGuard (cloud peers have public IPs and hit the unchanged
  direct-peer path - our changes only touch `192.168.64.x` peers - so expected to
  be unaffected, but unconfirmed in the mixed topology).

To close this: run `task e2e:mixed-cloud` with `/tmp/protos-local-net-mode` set.
Deferred by decision (real cloud spend); the shared-vmnet code only changes the
local-VM network device, and cloud connectivity uses the untouched direct-peer
WG path, so the risk is considered low for now.

## 2026-06-19

### Local two-VM image registry e2e: transient full-mesh misses

- Runs: `1781852687670778000`, `1781853285100853000`,
  `1781854771065167000`.
- Symptom: initial remote runtime full-mesh checks from VM 1 did not see VM 2,
  then subsequent checks recovered and both VMs matched the local checkpoint
  root.
- Evidence: harness printed `runtime does not see ... connected` during mesh
  wait, followed by `remote runtime peers ... connected=[...]` for both VMs and
  matching checkpoint roots.
- Impact: no hard failure in these runs, but it is recurring startup/convergence
  noise.
- Status: open for reliability follow-up. Not currently blocking once the wait
  loop retries.

### Local two-VM image registry e2e: checkpoint root mismatches recover

- Runs: `1781855514859370000`, `1781856404670776000`.
- Symptom: checkpoint verification occasionally sees one remote VM on an older
  or different durable root before a later poll matches the local root.
- Evidence: run `1781855514859370000` had recoverable seed-start and pull-start
  mismatches. Run `1781856404670776000` had recoverable mismatches for VM 1
  during both seed-start and pull-start verification; both VMs later matched and
  the e2e passed.
- Impact: not a hard failure, but recurring checkpoint-view churn adds latency
  and can combine with slower runtime phases.
- Status: monitor.

### Local two-VM image registry e2e: VM 2 WireGuard readiness delay

- Run: `1781854771065167000`.
- Symptom: host WireGuard ping for VM 1 succeeded repeatedly before VM 2 became
  reachable. VM 2 eventually responded and the run continued.
- Evidence: harness output showed four `host WireGuard ping ok` lines for
  `vm-e2e-1781854771065167000-1` before the first successful ping for
  `vm-e2e-1781854771065167000-2`.
- Impact: not a hard failure, but it adds startup variance and can consume e2e
  deadline when combined with slower app/image reconciliation.
- Status: monitor. No code change yet.

### Local two-VM image registry e2e: VM 2 initially running but unreachable

- Run: `1781855514859370000`.
- Symptom: VM 2 was deployed as `running (unreachable)` before later host
  WireGuard checks succeeded for both VMs.
- Evidence: harness bundle
  `/tmp/protos-local-macos-e2e-3588972214`, diagnostics
  `/tmp/protos-local-macos-e2e-3588972214/e2e-diagnostics/vm-e2e-1781855514859370000-2-image-pull-1781855514859370000.txt`.
- Impact: not the hard failure in this run, but it is the same readiness
  variance that can consume app-start and cleanup deadlines.
- Status: open for reliability follow-up.

### Local macOS VM direct peer addresses are host-reachable, not guest-reachable

- Run: `1781851844867497000`.
- Symptom: P2P peer refresh repeatedly tried `/ip4/192.168.64.x/tcp/10500`
  between guest VMs and failed with `i/o timeout` or `no route to host`.
- Evidence: VM logs and diagnostics at
  `/tmp/protos-local-macos-e2e-2644005800/e2e-diagnostics/vm-e2e-1781851844867497000-2-image-pull-1781851844867497000.txt`.
- Impact: the experimental direct-TCP refresh closed usable relay/overlay
  connections and caused image resolution to fall back to the registry.
- Status: fixed by removing the direct-TCP refresh and keeping WireGuard/internal
  addresses ahead of provider addresses. Continue tracking any direct-address
  retries as topology noise.

### Local two-VM image registry e2e: limited libp2p connections misclassified

- Run: `1781852687670778000`.
- Symptom: pull VM reported `protos.io/image.source=""`; app ran from Docker
  Hub even though VMs had a libp2p circuit connection.
- Evidence: VM logs in `/tmp/protos-local-macos-e2e-948797829/` showed
  `new limited connection` followed by image resolution reporting zero
  connected image-capable peers. The gRPC and raw blob dialers already allowed
  `network.Limited`, but `connectedClient` evicted anything other than
  `network.Connected`.
- Impact: valid relay-backed peers were removed before image capability checks.
- Status: fixed in code by treating `network.Connected` and `network.Limited` as
  usable peer-client states. Covered by `TestUsablePeerConnectednessAcceptsLimitedConnections`.

### Local two-VM image registry e2e: concurrent blob streams reset over relay

- Run: `1781853285100853000`.
- Symptom: pull VM found the seed peer and opened P2P blob streams over a limited
  circuit connection, small blobs completed, then larger concurrent streams reset
  with `unexpected EOF`; runtime fell back to the registry and left
  `protos.io/image.source=""`.
- Evidence: VM logs in `/tmp/protos-local-macos-e2e-395210375/` showed
  `opened image blob stream ... limited=true`, several successful small blob
  downloads, then `stream reset: connection closed: unexpected EOF` for larger
  blobs. Seed VM logged matching image blob stream failures.
- Impact: P2P path was selected but not reliable over the relay/circuit topology
  when multiple concurrent raw streams were used.
- Status: superseded by the follow-up single-stream failure in run
  `1781853926501297000`. Current fix now uses smaller resumable slices for
  limited relay peers instead of one large relay stream.

### Local two-VM image registry e2e: single relay blob stream reset

- Run: `1781853926501297000`.
- Symptom: pull VM selected the seed peer and avoided parallel range streams, but
  the first large raw P2P blob stream over a limited circuit reset after roughly
  23 ms. The retry then saw the peer as `NotConnected`, and the runtime fell back
  to Docker Hub with `protos.io/image.source=""`.
- Evidence: VM logs in `/tmp/protos-local-macos-e2e-2077841975/` show small blobs
  of 10333, 2497, and 12334 bytes completing over the limited relay, followed by
  a reset while streaming digest `sha256:d17f077...` with length `4194304`.
  Diagnostics were captured at
  `/tmp/protos-local-macos-e2e-2077841975/e2e-diagnostics/vm-e2e-1781853926501297000-2-image-pull-1781853926501297000.txt`.
- Impact: limited relay connections can describe image content and transfer
  small blobs, but a multi-MB raw stream is not stable enough in the local macOS
  circuit topology.
- Status: fix in progress. Limited relay peers now transfer blobs as smaller
  resumable offset slices and reconnect through the peer layer before retrying a
  failed slice.

### Local two-VM image registry e2e: limited relay slice transfer passes

- Run: `1781854771065167000`.
- Symptom: previous limited-relay stream reset did not reproduce after switching
  limited peers to smaller resumable offset slices. The pull app reached
  `running`, and the harness observed `protos.io/image.source="p2p"`.
- Evidence: harness output reported `remote P2P image label observed` and
  `Protos P2P image resolution verified` for seed
  `vm-e2e-1781854771065167000-1` and puller
  `vm-e2e-1781854771065167000-2`.
- Impact: hard P2P registry-fallback failure fixed in this run. Transfer remains
  slow: pull app start was requested around `2026-06-19T07:41:08Z`, and the app
  reached `running` around `2026-06-19T07:43:46Z`.
- Status: fixed for the observed reset path; keep monitoring throughput and
  relay stability in repeated e2e runs.

### Local macOS VM provider stop takes about 49 seconds during cleanup

- Runs: `1781853285100853000`, `1781853926501297000`,
  `1781854771065167000`, `1781855514859370000`,
  `1781856404670776000`.
- Symptom: VM 1 cleanup stayed at `stopping provider instance` from
  `2026-06-19T07:22:44Z` until `2026-06-19T07:23:33Z`, then volume and instance
  record deletion completed quickly. In run `1781853926501297000`, VM 2 provider
  stop took from `2026-06-19T07:32:10Z` to `2026-06-19T07:32:59Z`, and VM 1 took
  from `2026-06-19T07:34:19Z` to `2026-06-19T07:35:07Z`. In run
  `1781854771065167000`, VM 2 stopped from `2026-06-19T07:43:52Z` to
  `2026-06-19T07:44:41Z`, and VM 1 stopped from `2026-06-19T07:44:56Z` to
  `2026-06-19T07:45:44Z`. In run `1781855514859370000`, VM 2 stopped from
  `2026-06-19T07:58:38Z` to `2026-06-19T07:59:27Z`, and VM 1 stopped from
  `2026-06-19T07:59:41Z` to `2026-06-19T08:00:30Z`. In run
  `1781856404670776000`, VM 2 stopped from `2026-06-19T08:10:24Z` to
  `2026-06-19T08:11:05Z`, and VM 1 stopped from `2026-06-19T08:11:28Z` to
  `2026-06-19T08:11:59Z`.
- Evidence: harness delete task `019edec1-dfca-7e7e-b597-a13bc025f353` progress
  in the preserved terminal output for `1781853285100853000`, and delete tasks
  `019edeca-bdbe-757a-a587-ef46d37c0278` and
  `019edecc-3109-7032-ba0a-9ed6e1417736` for `1781853926501297000`.
- Impact: cleanup succeeded and no local VM runner processes remained, but this
  stop latency is worth trending because slow cleanup has caused deadlines in
  earlier runs.
- Status: monitor. Not blocking the current P2P fix.

### Local macOS VM DB peer dials fail over guest direct addresses

- Runs: `1781852687670778000`, `1781853285100853000`,
  `1781854771065167000`.
- Symptom: VMs repeatedly logged failed DB peer connections to remote VM
  `10501` over derived WireGuard IPv6 and `192.168.64.x`, commonly ending with
  `context deadline exceeded` or `no route to host`.
- Evidence: VM logs in `/tmp/protos-local-macos-e2e-948797829/` and
  `/tmp/protos-local-macos-e2e-395210375/`. The passing run's local-node log at
  `/tmp/protos-local-macos-e2e-278847742/flutter-node.log` also shows failed DB
  peer dials to both local VMs on port `10501`, with `no route to host` for
  derived IPv6 and `connection refused` for `192.168.64.x`.
- Impact: the app/image path recovered through Swarmion/libp2p, but these
  repeated failures are noisy and may affect convergence latency or resource use.
- Status: improved. A follow-up capability gate removed this noise from runs
  `1781855514859370000` and `1781856404670776000`; keep watching to confirm it
  stays gone.

### App runtime task feedback write failed after accepted checkpoint event

- Run: `1781855514859370000`.
- Symptom: the pull app stayed `stopped (running)` until the e2e timed out.
  The pull-side `apps.runtime.reconcile` task failed after trying to persist a
  task/status update: `published write did not reach local checkpoint for
  "update"` with `decision=accepted`, matching `checkpoint_event_root`, but
  `current_covers_event=false`.
- Evidence: diagnostics at
  `/tmp/protos-local-macos-e2e-3588972214/e2e-diagnostics/vm-e2e-1781855514859370000-2-image-pull-1781855514859370000.txt`
  show task `019edede-020f-70e6-adf6-103a04c44103` failed at
  `2026-06-19T07:53:21Z`. The final e2e failure was `app
  image-pull-1781855514859370000 ... did not reach status "running":
  status="stopped (running)"`.
- Impact: hard failure. This is not a P2P blob reset or registry fallback; the
  task feedback path treated local checkpoint lag as fatal after the write was
  already accepted.
- Status: fixed and validated once. Task record/event persistence now publishes
  durable feedback without requiring the caller to wait for local checkpoint
  catch-up. The next run, `1781856404670776000`, passed P2P image-source
  verification and log checks found no repeat of this error.

### Local macOS host-agent network peer reconfiguration deadline during cleanup

- Run: `1781856404670776000`.
- Symptom: after the e2e had passed P2P verification and while VM 1 provider
  stop was in progress, the local node logged `failed to configure network
  peers ... DeadlineExceeded ... RST_STREAM with error code: CANCEL`.
- Evidence: preserved bundle `/tmp/protos-local-macos-e2e-2556146026`,
  `flutter-node.log` around `2026-06-19T11:11:58+03:00`.
- Impact: not a hard failure in this run; VM 1 deletion completed at
  `2026-06-19T08:12:00Z` and no VM runner process remained. This still points
  at cleanup/control-plane responsiveness in the host-agent network
  reconfiguration path.
- Status: open for follow-up if it repeats.

### Cleanup durable peer removal improved but remains worth watching

- Runs: `1781852687670778000`, `1781853285100853000`,
  `1781853926501297000`, `1781854771065167000`,
  `1781855514859370000`, `1781856404670776000`.
- Symptom: cleanup entered `waiting for durable peer removal`, then moved on
  after roughly six seconds in the earlier recent runs. In run
  `1781853926501297000`, VM 2 moved from durable peer removal to provider loading
  after about 36 seconds, and VM 1 took about 67 seconds. In run
  `1781854771065167000`, VM 2 moved past durable removal in under one second,
  and VM 1 took about six seconds, but local logs still showed stale Swarmion
  observations being treated as non-blocking. In run `1781855514859370000`,
  VM 2 durable peer removal took about 36 seconds and VM 1 took about six
  seconds. In run `1781856404670776000`, VM 2 durable peer removal took about
  5.5 seconds and VM 1 took about 10.5 seconds.
- Evidence: harness task progress around delete tasks in both runs.
- Impact: not a failure in these runs, but previous sessions had multi-minute
  waits and cleanup deadlines in this same phase.
- Status: monitor. Recent changes appear healthier, but keep logging exact wait
  durations when cleanup is slow.

### Cleanup DB operation lock waits during provider resource deletion

- Run: `1781854771065167000`.
- Symptom: cleanup succeeded, but the local node logged DB operation lock waits
  around 7.4-10.1 seconds while deleting provider records.
- Evidence: `/tmp/protos-local-macos-e2e-278847742/flutter-node.log` logged
  `committed update publish phase took 10.066955208s`, `select multiple waited
  10.039188292s for db operation lock`, later `committed insert publish phase
  took 7.861879625s`, and similar waits during VM 1 cleanup.
- Impact: not a hard failure in this run, but these waits can starve APIC reads
  and task watchers under tighter deadlines.
- Status: open. Needs follow-up on cleanup write/publish boundaries and DB lock
  scope.

## P2P Image Transfer RCA History

### 2026-06-19 Detailed Run Log

- Baseline observation from the latest local two-VM image-registry e2e: P2P image resolution is correct, but transferring a large nginx layer was slow. A 19 MB blob advanced from blob 2 to blob 3 after roughly 90 seconds while task progress was streamed through APIC. This points at the blob transport path, not Swarmion checkpointing or content closure.
- During the first streaming-transfer e2e run, full-mesh wait saw transient remote runtime-state deadlines for both local VMs: `rpc error: code = DeadlineExceeded desc = context deadline exceeded`. Both VMs later recovered, reported full runtime peer mesh, and matched the local checkpoint root. This looks like admin/runtime-state responsiveness during mesh setup, not image transfer.
- The same run then reached a pull app `running` state, but `GetInstanceImage` observed `protos.io/image.source=""` instead of `p2p`. The pull task had previously reported `refreshing image peers` with one disconnected image peer, so the current hypothesis is peer-client freshness/selection during P2P image resolution, not the new stream itself.
- During cleanup of that failed run, an instance delete task emitted a `succeeded` progress update, then the harness follow-up `GetTask` call timed out: `get task 019edcf2-a4ee-74ca-83b9-1a5d56a0ea51: rpc error: code = DeadlineExceeded desc = context deadline exceeded`. This appears to be APIC/task responsiveness after terminal progress, not a failed delete operation.
- Cleanup then saw additional APIC deadlines while watching the second delete task and removing the provisioner image/provisioner. The VM delete had already reached provider stop, so this appears to be local APIC responsiveness under cleanup/failure pressure. Needs a separate RCA if it repeats on successful runs.
- After fixing synchronous peer-client registration, the next run did use P2P and eventually passed image verification, but the large layer still did not complete within a 90 second watch, and concurrent remote app-status/APIC calls timed out during the transfer. This suggests gRPC-over-libp2p blob transfer is still saturating or blocking the control channel; next change is a dedicated raw libp2p blob data stream.
- The raw libp2p blob stream kept app-status/APIC calls responsive, but the 19 MB layer still took about 106 seconds (`23:13:56` to `23:15:42`). The bottleneck appears to be per-stream throughput rather than gRPC framing alone. Next change is range-based parallel blob transfer over multiple raw streams.
- The same raw-stream run also logged several seed-app status failures while checkpoint state was changing: `catch up swarmion checkpoint view: swarmion checkpoint catch-up retryable: status=target_changed`. The e2e continued into cleanup, so this is tracked as an intermittent API/status-read issue separate from blob transfer throughput.
- Cleanup in that raw-stream run was delayed by checkpoint catch-up churn. `RemoveInstance` for the first cleanup VM was requested at `02:19:08`, logged a checkpoint catch-up deadline at `02:21:23`, retried a stale checkpoint write at `02:21:31`, deferred another target-changed catch-up at `02:21:39`, and only queued the delete task at `02:21:50`. This is a checkpoint/write responsiveness issue during teardown, not a host-agent VM-stop issue.
- The first cleanup delete then stalled at `waiting for durable peer removal`. Logs showed the removed peer still marked `connected`, `active-view`, `state-provider`, `checkpoint-provider`, and `content-provider`; after eviction the remaining blockers narrowed to stale local `connected` and `active-view` observations, which were later treated as non-blocking. The task missed the harness cleanup deadline before that completed, so follow-up APIC cleanup calls timed out.
- Final result for the raw single-stream run: e2e failed because `image-pull-1781824383008839000` on the second VM did not reach `running` before the harness deadline (`DeadlineExceeded`). Cleanup did eventually clear the local VM runner processes through the host-agent fallback. This confirms the single-stream transport is still too slow for the e2e deadline even when the P2P source path is correct.
- First run after rebuilding with parallel range transfer (`1781825188181826000`) did not reach the blob transfer phase. VM 1 reached app `running`, then the first seed-side `GetImageContent` call reset the P2P stream and the VM began refusing connections on port `10500` for both WireGuard IPv6 and local IPv4 TCP. Subsequent dials only saw `connection refused` or QUIC timeouts. The e2e failed waiting for seed VM image content, so this run is evidence of a seed VM/protosd P2P listener failure, not a measurement of the range-transfer code.
- Cleanup for that failed run again showed slow declarative peer removal. VM 2 deletion completed after several minutes, but VM 1 cleanup timed out through APIC and then host-agent fallback cleared the local VM runners. No `protos-hostagent --run-vm` or Virtualization VM processes remained afterward.
- Debuggability issue from the same failed run: once VM 1 stopped answering P2P, APIC `GetInstanceLogs` could only try SSH and did not use the local macOS host-agent/provisioner console diagnostics. The VM console also lacked `protosd` service logs because the LinuxKit targets defined `logwrite` but did not run it. This blocked a precise RCA of whether `protosd` crashed or the listener was otherwise torn down.
- Re-run with local diagnostic fallback and LinuxKit `logwrite` (`1781826396589051000`) recovered from the previous seed-side `GetImageContent` reset: the seed app reached `running` and image content metadata was returned. The pull app then did not reach `running` before the harness moved into cleanup, so the run still did not prove the parallel range-transfer path.
- During cleanup for `1781826396589051000`, APIC removal of VM 2 failed with `failed to insert: catch up swarmion checkpoint view for published write: context deadline exceeded`. This is another cleanup/checkpoint responsiveness issue rather than an image-transfer measurement.
- In the same run, VM 2 initially reported `running (unreachable)` but later recovered, passed host WireGuard ping, matched runtime peers, and matched the local checkpoint root. Treat this as an intermittent runtime/APIC readiness issue unless it repeats as a hard failure.
- The pull-side app request for `1781826396589051000` stalled before image transfer started. The local client logged "Run app" at `02:47:59`, "Creating application" at `02:48:11`, cleanup began at `02:48:30`, and "Created application" only appeared at `02:48:37`. There was no subsequent "Starting app" log for the pull app. Current RCA target: a desired-state write (`CreateApp`) took longer than the e2e helper's 30 second RPC window, likely while waiting for checkpointed write visibility.
- Cleanup for the same run also timed out while VM 1 deletion was waiting for durable peer removal, with blockers including connected/active-view and state/checkpoint/content-provider observations. Host-agent fallback still cleared the VM runner processes.
- After changing `CreateApp` validation to use declared instance state, the next local two-VM run (`1781827249102299000`) passed P2P image-source verification. `CreateApp` and `StartApp` returned immediately, but the pull-side app still took about 2m40s to become running. VM 2 provider logs showed the app shim connecting at `2026-06-19T00:05:01.598Z`, while the local app request started at about `03:02:21` Europe/Athens.
- In the same passing run, APIC diagnostics against VM 2 briefly failed with `could not find RPC client for instance`, even though the VM later started the app and reported the P2P image label. This is an observability/control-plane freshness issue during the transfer window.
- Cleanup for `1781827249102299000` still emitted warnings despite overall e2e success. VM 2 spent about 100 seconds waiting for durable peer removal before provider cleanup started. VM 1 deletion reached record deletion, but the harness task watch hit a deadline, and APIC removal of the provisioner image/provisioner also timed out.
- Follow-up instrumentation goal: report P2P image-transfer bytes through non-durable task progress and log timings for blob download, content import, and unpack. The current blob-level progress hides whether the remaining 2-3 minute gap is network transfer, containerd content import, unpack, or peer-control churn.
- First run after adding transfer/import telemetry (`1781828281780098000`) failed before data transfer. `CreateApp` for the seed app took about 11 seconds, then `StartApp` blocked waiting for the published write to reach the local checkpoint view. The VM 1 console showed the seed app container shim starting at `2026-06-19T00:19:17Z`, but the local APIC call later failed with `failed to update: catch up swarmion checkpoint view for published write: context deadline exceeded`. RCA: the desired-state write had propagated far enough for the remote runtime to act, but the user-facing APIC call was tied to local durable checkpoint visibility. Follow-up fix: make app desired-state CRUD return at the published declarative-write boundary and keep durable checkpoint waits explicit for cleanup/verification paths.
- Cleanup for `1781828281780098000` again timed out through APIC. VM 2 peer removal waited from `03:20:25` to `03:21:02` Europe/Athens before provider cleanup, then provider stop/volume/delete continued past the harness cleanup watch. VM runner processes were gone afterward; only the root host-agent daemon remained.
- After making app desired-state CRUD return at the published-write boundary, the next run (`1781828918774257000`) got past seed start and reached pull app startup. The pull app never reached `running`; it stayed `stopped (running)` until remote status reads began timing out, then the e2e failed with `app image-pull-1781828918774257000 on vm-e2e-1781828918774257000-2 did not reach status "running": rpc error: code = DeadlineExceeded desc = context deadline exceeded`. During the failure window, a direct APIC `GetTasks` call against VM 2 failed to connect to the peer admin API over `/ip4/192.168.64.248/udp/10500/quic-v1`, while earlier runtime-state checks had succeeded. This points at a control-plane responsiveness/connectivity failure during pull-side startup, before the new transfer telemetry was visible.
- The same run did show evidence that the published-write boundary fix worked: seed `StartApp` returned, VM 1 reported remote image content, and the pull-side start checkpoint matched on both local and remote views before the app health wait failed. The next RCA target is why VM 2 becomes unreachable or unresponsive while reconciling the pull app.
- Re-run after adding local provider diagnostics to APIC logs (`1781829744810915000`) reproduced the earlier seed-side failure before pull transfer. Seed app reached `running`, but the first `GetInstanceImage` / `GetImageContent` call against VM 1 reset the stream at `03:43:43` Europe/Athens: `stream reset: connection closed: read tcp6 ... read: connection reset by peer`. The local P2P close handler immediately marked VM 1 disconnected, and every subsequent TCP/QUIC dial to VM 1 port `10500` failed with `connection refused` or QUIC timeout until cleanup. The e2e failed with `image content was not ready on vm-e2e-1781829744810915000-1`.
- Provider-side diagnostics for the same run showed the VM process itself remained alive long enough to start the nginx app shim at `2026-06-19T00:43:38Z`, and both host VM runner processes were present before cleanup. This narrows the failure to the remote `protosd`/P2P listener or its network binding, not the macOS VM process or the guest app container.
- The local provider diagnostics path now works through APIC when P2P logs fail or, for local VMs, as an appended provider section after successful P2P logs. It still does not expose detailed `protosd` logs from inside the LinuxKit service; the provider console shows LinuxKit/containerd and shim events, not Go runtime logs. If the listener failure repeats after the next fix, the VM image logging path needs to persist `protosd` stdout/stderr into the APIC-readable log.
- The same run cleaned both local VMs, with no `protos-hostagent --run-vm` or Virtualization VM processes left afterward. VM 1 delete reached `succeeded`, but the harness follow-up `GetTask` call timed out again after terminal progress, so APIC task-watch responsiveness after cleanup remains a recurring secondary issue.
- After changing the local P2P gRPC serve loop to restart instead of panicking on unexpected `Serve` errors, the next run (`1781830712781852000`) no longer showed the immediate seed-side stream reset / listener crash signature. It still failed before data transfer: VM 2 initially reported `running (unreachable)`, later recovered and reached full mesh/checkpoint match, the seed app reached `running`, then `GetInstanceImage` against VM 1 first hit a deadline and later canceled. Local logs showed VM 1 DB peer port `10501` repeatedly refusing while P2P port `10500` remained unstable enough that the local node lost the RPC client. No `p2p grpc server stopped unexpectedly` log appeared on the local node, so this run points at remote runtime/control-plane readiness or guest-side service logging gaps rather than the local serve-loop panic path.
- Provider-side console for VM 1 in `1781830712781852000` again showed the guest stayed alive and the nginx app shim started (`2026-06-19T01:03:16Z`), but it did not include protosd service logs. The next useful diagnostic improvement is to make the LinuxKit `protos` service write its stdout/stderr to the VM console or another APIC-readable log, otherwise remote protosd stalls/crashes remain inferred from port behavior.
- Cleanup for `1781830712781852000` removed both VM runner processes, but VM 1 cleanup still timed out from the harness after the delete task reached provider stop. The local node also logged `failed to configure network peers: host agent network configure peers ... DeadlineExceeded` during cleanup, suggesting host-agent network reconfiguration responsiveness is another cross-cutting control-plane issue that can interfere with e2e timing.
- Re-run after rebuilding the local mactest image with the P2P serve-loop hardening (`1781831517893112000`) reproduced the seed-side failure. The seed app reached `running`, the first seed `GetInstanceImage` / `GetImageContent` stream reset with `connection reset by peer`, and every subsequent dial to VM 1 port `10500` failed with `connection refused` or QUIC timeout until cleanup. VM cleanup completed cleanly with no leftover VM runner processes. This means the serve-loop recovery patch alone did not resolve the remote peer listener failure; the next run must include guest-side protosd stdout/stderr logging so the failure can be tied to a process exit, panic, or listener teardown.
- Re-run after rebuilding the local mactest image with protosd log mirroring (`1781832239236458000`) did not reproduce the seed-side listener reset. The seed app reached `running`, and seed image content metadata was available with 11 blobs. The run failed before transfer measurement because pull `StartApp` exceeded its 30 second APIC call deadline; the pull app shim later appeared in the VM console at `2026-06-19T01:31:13Z`, after cleanup had already started. This points at app desired-state write/control-plane latency rather than data-transfer throughput. Cleanup again hit APIC deadlines, but host-agent fallback left no VM runner or Virtualization processes behind.
- The same run showed the VM console still does not include protosd application logs, only LinuxKit/containerd/shim logs. The entrypoint log mirroring may be writing `/var/log/protos.log` inside the protos service container without that stream being forwarded to the provider console. APIC `GetInstanceLogs` remains the preferred path while the peer is reachable; for postmortem on an unreachable local VM, the host-agent/provider log surface still needs better protosd log extraction.
- Current re-run after context-aware app writes (`1781833317244846000`) did not reach image transfer. Both local macOS guests showed BTRFS write I/O errors on `/dev/vda1`, forced the data filesystem read-only, and VM 2 then failed to start containerd with `mkdir /var/lib/containerd: read-only file system`. Root cause was host disk exhaustion: the macOS filesystem had only about 182 MB free while many old `/private/tmp/protos-local-macos-e2e-*` bundles occupied roughly 100 GB. After deleting stale generated bundles and keeping only the active run bundle, free space returned to about 102 GB. This run is invalid for transfer measurement and should be repeated from a clean bundle.
- Cleanup for `1781833317244846000` ran while the host filesystem was still full, so APIC cleanup lost the local-node socket and host-agent fallback initially failed to write a temporary manifest with `no space left on device`. After freeing disk space, no `protos-hostagent --run-vm` or Virtualization VM processes remained. Treat this as a disk-pressure cleanup artifact, not a VM lifecycle bug unless it repeats with sufficient free space.
- Fresh re-run with sufficient disk (`1781833831050584000`) failed before image transfer telemetry appeared. Seed app reached `running` and seed image content was available with 10 blobs. Pull app declarative state reached both VMs, but the pull app stayed `stopped (running)` until the e2e deadline. The local DB logs showed the deeper cross-cutting issue: committed writes hold the DB operation lock through slow apply/publish phases, including seed `StartApp` publish 19.5s, pull `CreateApp` apply 4.7s, pull `StartApp` publish 15.6s, and cleanup writes with reader waits as high as 46.9s. This starved APIC/runtime status calls and cleanup task reads. Next fix: shrink the DB operation lock so it protects local SQL/staging only, not network checkpoint publish/catch-up waits.
- Cleanup for `1781833831050584000` again showed terminal task progress followed by a timed-out `GetTask` call, then VM 1 cleanup timed out through APIC. Host-agent fallback cleared all local VM runner and Virtualization processes. Disk remained healthy at about 102 GB free, so this is not the earlier disk-pressure failure.
- After implementing long-lived image blob readers and moving checkpoint observation outside the DB operation lock, `task -t core/Taskfile.yaml test` passed. The follow-up local image rebuild with `task -t cloud-provisioning/Taskfile.yml mactest` failed before Docker build started because the configured Docker endpoint `unix:///Users/al3x/.orbstack/run/docker.sock` was unavailable. This is an environment blocker; restore or switch Docker context, then rerun the same Taskfile build.
- In the next e2e run (`1781835154375253000`), starting OrbStack restored Docker and `task -t cloud-provisioning/Taskfile.yml mactest` passed. During remote checkpoint verification after seed app start, VM 1 briefly returned `catch up swarmion checkpoint view: swarmion checkpoint catch-up retryable: status=target_changed`, then the following runtime checkpoint check matched the local root. Track as intermittent checkpoint-view churn rather than a hard failure unless it repeats or causes a deadline.
- In the same run, the first two local checkpoint waits for pull app start hit `rpc error: code = DeadlineExceeded desc = context deadline exceeded`; the third poll recovered and both VMs later matched the pull app start checkpoint root. This is intermittent local APIC/checkpoint responsiveness, not a hard convergence failure in this run.
- The same e2e failed before blob-transfer logs appeared because the pull app stayed `stopped (running)`. VM 2 cleanup then waited from `02:18:41Z` until about `02:20:52Z` in durable peer removal before provider cleanup progressed. During that wait the removed peer remained marked connected/active/state/checkpoint/content provider in Swarmion observations, then eventually cleared enough to continue. Cleanup then hit APIC task-watch/provisioner cleanup deadlines and logged a host-agent network peer reconfiguration deadline, but no VM runner or Virtualization processes remained after the harness exited. This is a pre-transfer app convergence/control-plane issue plus a cleanup/convergence delay, not a measurement of the long-lived blob reader optimization.
- After adding direct runtime-change app notifications, run `1781836193418666000` reached the pull-side app reconcile path. APIC task inspection showed VM 2 queued and ran `apps.runtime.reconcile` for peer `12D3KooWQEqz524b9iaSbzUfRGTS9PEmykWvuJvXTku6c1tHKMs7`, so the earlier "stopped (running)" symptom was not a missing reconcile notification in this run.
- The same run failed after the first pull reconcile task ran from `02:30:51Z` to `02:33:54Z`. The task failed creating the sandbox with `parent snapshot sha256:a92d827df71b2cfc2905c8311eeb158911eb4e604e298b716908673368de49d2 does not exist`; subsequent reconcile retries failed faster with missing content digest `sha256:e3573bbeaf27fc648133d1c35b5d981f962f332fab1b4e0c653173fdabb73366`. Root cause candidate: partial P2P content import/unpack left a containerd image record that `ImageExistsLocally` treated as usable, so retries skipped P2P repair and went straight to broken sandbox creation.
- After adding runtime-side image readiness verification, run `1781836978604049000` avoided the immediate broken containerd retry loop and entered P2P transfer. A live APIC `WatchTask` on VM 2's `apps.runtime.reconcile` task showed `bytes_downloaded=24043879`, `bytes_total=28722771`, `completed_blobs=5`, `elapsed_ms=139063`, and about `172898` B/s before the harness app-start deadline moved into cleanup. This confirms the next blocker is raw transfer throughput, not missing reconcile notification or skipped repair after a partial image record.
- The same run had transient VM readiness/control-plane noise before transfer: VM 2 took several host WireGuard ping attempts, full-mesh runtime checks briefly hit APIC deadlines, and local checkpoint verification had one APIC deadline before matching both VMs. Cleanup eventually deleted VM 2, VM 1 cleanup/provisioner cleanup timed out through APIC, and host-agent fallback left no VM runner or Virtualization processes. Because VM 2 was deleted during cleanup, only VM 1's console remained in the preserved bundle; VM-side postmortem logs for the failed pull were lost.
- First run after switching the raw P2P blob stream to binary frames and concurrent blob downloads (`1781837884799856000`) failed before transfer measurement. The seed app container started in VM 1, but local `StartApp` returned `failed to update: merge staged SQL root=4fc1b1a9 current=2652ddb0 base=d6ab8dd8: merge roots ours=2652ddb0 theirs=4fc1b1a9 ancestor=d6ab8dd8: context deadline exceeded` after a notifier checkpoint catch-up deadline. Cleanup then stuck waiting for VM 2 durable peer removal while the removed Swarmion peer stayed marked `connected`, `active-view`, and provider-bearing. This run is a checkpoint/write and peer-removal blocker, not a data-transfer measurement.
- After moving broad DB callbacks to published checkpoint-root events, run `1781838655053226000` got past the seed `StartApp` APIC deadline but exposed an app-runtime dedupe race. The seed app was created with desired `stopped`, which queued a VM 1 reconcile task. `StartApp` changed the row to desired `running` while that task was running; the task reconciled one stopped app without emitting `starting app`, then refreshed its dedupe signature to the newer `running` state at completion. The local APIC SQL view showed the app row as desired `running`, but no second reconcile task was queued and the app stayed `stopped (running)` until cleanup. Fix: a reconcile task must only claim the signature that caused it to be queued, not a later desired-state signature written while it was running.
- After removing reconcile completion signature refresh, run `1781839361627398000` confirmed the dedupe race fix: VM 2 queued a second `apps.runtime.reconcile` task for the pull app after the initial stopped-state reconcile. The task completed the P2P image transfer and runtime verification path through APIC task streaming: events reached `unpacking image`, `unpacked image`, `verifying image snapshot`, `image content ready` with `downloaded_blobs=11`, and `resolved image from peers`. Immediately afterward, the task watch stream reset with `connection reset by peer`, subsequent APIC `GetTasks` to VM 2 failed to connect to peer `12D3KooWBAHKVPby2uo14k4QX3TG8banofZXaJ79wMKCj4LUC5im`, and the harness failed waiting for the pull app to report `running`. This run proves the raw concurrent transfer can finish before the e2e timeout, but exposes a post-transfer/control-plane failure while starting or reporting the pull app.
- Run `1781840162426988000` used the bounded diagnostics path and failed because the pull VM fell back to the registry instead of using P2P. VM 2 logs show the pull reconcile attempted P2P, refreshed one disconnected known image peer, then `GetImageContent` against seed peer `12D3KooWPYD35MtpVYWUdwZF3cuQ62gKXNCJd8RQsrcfDYara87W` reset with `connection reset by peer`; VM 2 then logged `Pulling image 'docker.io/library/nginx:alpine'` and started the app from Docker Hub, leaving `protos.io/image.source=""`. The harness captured diagnostics at `/tmp/protos-local-macos-e2e-276381463/e2e-diagnostics/vm-e2e-1781840162426988000-2-image-pull-1781840162426988000.txt`. VM 1 was already unreachable through P2P by then, so APIC could only return provider console logs, which lacked protosd logs. Follow-up: mirror protosd logs to the VM console so the next seed-side reset can be tied to a remote protosd/server failure instead of inferred from dial behavior.
- Run `1781840918684143000` used the rebuilt local image with protosd output mirrored to the provider console. It again failed before P2P transfer because VM 2 fell back to Docker Hub and left `protos.io/image.source=""`. The new diagnostics show neither protosd daemon crashed or exited: both VMs kept serving logs through cleanup, but VM 2's image peer refresh observed `known=2 disconnected=1 connected=0`, then attempted registry pull. VM 2 logs show repeated dials to seed peer `12D3KooWNqp9hWg241FpF8MMEw6puyknREiiVSDJQjwdqhtN4YgG` failing with identify/ping deadlines and one remote stream reset (`error reading server preface: stream reset (remote): code: 0x1002`). VM 1 logs show the symmetric problem: repeated failed dials to VM 2 over WireGuard IPv6, local IPv4, and circuit addresses, followed by a disconnect notification. This narrows the next fix to lower-level peer connectivity freshness and usable-address selection, not `ResolveImage` itself or image blob transport.
- Run `1781841727815439000` passed the two-local-VM image-registry e2e after the peer layer was changed to aggregate all destination addresses for a peer and let libp2p race them as one dial attempt, with a longer RPC handshake window. VM 2 reached `running`, the harness observed `protos.io/image.source="p2p"`, and cleanup completed without host-agent fallback or leftover VM runner processes. Transfer evidence from VM 2 logs: the pull app started at `04:04:18Z`, blob downloads from seed peer `12D3KooWJqXeWWdzLEZLT12zjWyurKjey52K6haGwhXPHa76Whhe` began at `04:04:50Z`, and the harness observed the running app plus P2P image label before cleanup at `04:07:31Z`. Intermittents in the passing run: initial full-mesh verification briefly saw VM 1 missing VM 2, pull runtime checkpoint verification had a transient root mismatch, the remote reconcile task sat at `loading app state` from `04:04:14Z` until logs showed `Syncing apps` at `04:04:18Z`, VM 2 durable peer removal waited roughly 66 seconds, and the protos service logged `tee: /dev/console: Operation not permitted`. The Dockerfile was then adjusted to write the persistent `/var/log/protos.log` and fall back to service stdout when `/dev/console` is not writable; `task -t cloud-provisioning/Taskfile.yml mactest` rebuilt successfully with that fix.
- Investigated refinement: local macOS VM addresses are recorded in the generic `PublicIP` field even though they are private host-only `192.168.64.x` addresses. A peer-layer experiment preferred those private provider addresses before the derived WireGuard address while keeping true public cloud addresses behind WireGuard. This was later invalidated by VM-to-VM tests: the addresses are host-reachable, not reliably guest-reachable.
- Run `1781851159409706000` passed the two-local-VM image-registry e2e with P2P image-source verification, but transfer throughput was still poor: VM 2 reported about 28.7 MB downloaded in roughly 159 seconds, around 180 KB/s. Import, unpack, and app start after the transfer were fast. Cleanup completed without leftover VM runner processes.
- Run `1781851844867497000` tested a more aggressive direct-TCP preference and failed because VM 2 fell back to the registry, leaving `protos.io/image.source=""`. Diagnostics at `/tmp/protos-local-macos-e2e-2644005800/e2e-diagnostics/vm-e2e-1781851844867497000-2-image-pull-1781851844867497000.txt` showed repeated attempts to refresh connected peers onto `/ip4/192.168.64.x/tcp/10500`, followed by `i/o timeout` and `no route to host`. The refresh loop closed usable overlay/relay connections while chasing host-only local VM addresses that guest VMs cannot reliably reach directly.
- RCA from that failure: `192.168.64.x` local macOS VM addresses are reachable from the macOS host but are not a globally valid guest-to-guest transport. The peer layer must not promote those addresses or recycle working connections without a lower-layer reachability signal. The direct-TCP preference and connection refresh were removed; the peer layer now keeps the aggregated libp2p dial path and orders internal/WireGuard addresses before provider public/private addresses. The remaining throughput issue belongs in the lower network path: the Linux WireGuard module intentionally hairpins local macOS VM-to-VM internal traffic through the local user-device relay at the macOS NAT gateway when local NAT attachment is detected.
- Run `1781852687670778000` reproduced registry fallback after removing the direct-TCP refresh. The app reached `running`, cleanup completed without leftover VM runners, but the pull VM reported `protos.io/image.source=""`. Logs showed the deeper peer-client bug: both VMs had only a libp2p `Limited` connection to each other through the circuit path, and `connectedClient` treated anything other than `network.Connected` as unusable. The gRPC and raw image stream dialers already allowed `network.Limited`, so image resolution was deleting otherwise valid clients before checking image capabilities. Fix: centralize peer connectedness semantics and treat `network.Connected` and `network.Limited` as usable peer-client states.
- Run `1781853285100853000` confirmed the limited-connection peer-client fix: the pull VM found the seed peer and opened raw P2P image blob streams over the circuit path (`limited=true`) instead of immediately reporting zero image-capable peers. The run still fell back to the registry because concurrent blob streams over that limited relay connection reset with `unexpected EOF`; small blobs completed in milliseconds, then larger concurrent streams failed. Fix in progress: limited relay peers now use one blob stream at a time and avoid range-parallel blob streams, while direct peers keep the existing concurrent transfer path.
- Run `1781853926501297000` showed that simply reducing limited relay peers to one large raw stream was not enough. The pull VM successfully downloaded three small blobs over the limited circuit, then the first 4 MiB stream for digest `sha256:d17f077...` reset after about 23 ms. The retry saw the seed peer as `NotConnected`, so image resolution fell back to Docker Hub and the final label remained `protos.io/image.source=""`. Cleanup completed through product delete tasks and left no local VM runner processes, but durable peer removal took about 36 seconds for VM 2 and about 67 seconds for VM 1, and each provider stop took roughly 49 seconds. Next fix: for limited relay peers, transfer blobs as smaller resumable offset slices and reconnect through the peer layer before retrying a failed slice; keep direct peers on the faster parallel range-stream path.
- Run `1781854771065167000` passed after the limited-relay slice change. The pull app reached `running`, the harness observed `protos.io/image.source="p2p"`, cleanup completed through product delete tasks, and no local VM runner processes remained. This confirms the previous hard reset/registry-fallback path is fixed for the local two-VM relay topology. Remaining concerns from the same run: transfer/app startup still took roughly 158 seconds from pull `StartApp` to the P2P label, VM 2 host WireGuard ping had several readiness retries, the first full-mesh check missed VM 2 before converging, DB peer dials to VM port `10501` still fail over direct guest addresses, cleanup still logs stale Swarmion peer observations, and each local provider stop still takes about 49 seconds.
- Run `1781855514859370000` failed after the limited-relay slice fix and after the DB peer capability gate, but not in the P2P blob-transfer path. VM 2 initially deployed as `running (unreachable)`, later passed host WireGuard and full-mesh checks, and two checkpoint-root mismatches recovered. The pull-side `apps.runtime.reconcile` task then failed while persisting task/app feedback: `published write did not reach local checkpoint for "update"` even though the checkpoint event decision was accepted and the event root matched; the current checkpoint did not cover the event before the wait deadline. There was no evidence of a blob reset or registry fallback in this run. The current RCA target is durable task feedback writes waiting on local checkpoint visibility, not image transfer.
- Run `1781856404670776000` passed after task record/event persistence was moved to the published boundary. The pull app eventually reached `running`, the harness observed `protos.io/image.source="p2p"`, and cleanup completed through product tasks with no leftover VM runner process. Log checks in `/tmp/protos-local-macos-e2e-2556146026/flutter-node.log` found no repeat of `published write did not reach local checkpoint`, no DB peer direct-dial noise, and no registry fallback label. Remaining concerns from the passing run: seed-start and pull-start checkpoint verification each had a recoverable VM 1 root mismatch, the pull app still spent roughly three minutes in `stopped (running)` before reporting `running`, VM 1 cleanup logged one host-agent network peer reconfiguration `DeadlineExceeded`, and provider stop still took tens of seconds.
