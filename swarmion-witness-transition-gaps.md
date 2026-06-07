# Swarmion Witness Transition Gaps

Date: 2026-05-26

## Purpose

This note summarizes the Swarmion-facing gaps found while adapting Protos to the latest witness-transition API and running a full mixed local/cloud e2e. It is written for the Swarmion project owner/architect.

The short version: the failures observed do not look like protocol safety failures. They look like liveness and API-boundary hazards around witness-set transitions, especially when the product calls the transition API while peer/rank information is still converging.

## Current Protos Integration Model

Protos now treats witness rank as device-local state stored in the SQL `devices` table. The initial local laptop writes its own static rank during DB initialization. New devices are inserted with rank derived from device type, and when each device comes online it publishes its own rank into the Swarmion protocol path after bootstrapping finalized DB state.

The intended rank ordering is:

- `cloud_vm`: 100
- `local_user_client` / laptop: 50
- `local_vm`: 30
- `phone`: 10

There is intentionally no remote admin call where the original author sets rank directly on a remote peer after provisioning. The flow is:

1. Author adds/provisions a peer.
2. The peer boots and bootstraps DB/protocol state.
3. The peer reads its local device row/type.
4. The peer publishes its own rank through Swarmion.
5. Any eligible reconciler can apply the witness candidate set from finalized DB state.

## E2E Scenario Verified

The mixed witness e2e now performs this sequence:

1. Start with the local laptop/client as the only witness.
2. Add two local macOS VMs.
3. Add one Hetzner VM.
4. Add one Scaleway VM.
5. Verify witness-set transitions after each phase.
6. Deploy two app/container rows using `docker.io/protosio/protos-e2e-probe:latest`.
7. Verify app status reaches `running` on both cloud VMs.
8. Verify container-to-container HTTP connectivity in both directions.
9. Destroy local/cloud resources and verify no leftovers.

The final passing run showed:

- laptop rank `50` published and initial witness set stayed on the laptop;
- local VMs published rank `30` and remained fallback candidates below laptop;
- a single Hetzner VM published rank `100` but did not take over by itself;
- after Scaleway also published rank `100`, Swarmion transitioned to the two cloud VMs as active witnesses;
- all peers matched DB heads before and after app writes;
- both cloud apps reached `running`;
- HTTP connectivity worked Hetzner -> Scaleway and Scaleway -> Hetzner.

## Main Gap: Transition Readiness Is Too Easy To Misapply

The key issue is that the product currently has to infer when it is safe to call `ApplyWitnessCandidates`.

In the failing path, Protos could call the transition API while the candidate formation was still incomplete or drifting. For example, one cloud VM rank might be finalized while the second cloud VM is not yet finalized, or the finalized DB state might include peer rows before all peers have published protocol ranks. If Protos passes that partial view to Swarmion, a child epoch can be created from a formation that is not the product's intended final formation.

This does not look like Swarmion finalizing conflicting histories. The observed behavior was that later writes could end up on a stale tentative branch after another witness transition became active. That is a liveness failure for the application, not a demonstrated safety/corruption failure.

What would help:

- An explicit API/status that says candidate application is ready, not merely possible.
- A structured result explaining missing finalized ranks, missing candidate peers, current active formation, and whether the requested formation is already adopted.
- A documented precondition that callers should only pass candidate sets derived from finalized state at a specific finalized root/epoch.

## Gap: Idempotence Semantics Are Not Clear Enough

`ApplyWitnessCandidates` can be called by multiple peers. That is good, but the product needs stronger clarity on idempotence.

From Protos' perspective, these cases need distinct API results:

- "This exact candidate formation is already active; no transition was created."
- "This formation is accepted and a new child epoch was created."
- "This caller is behind; refresh finalized state before applying."
- "This formation conflicts with an in-flight or newer witness transition."
- "This request is incomplete because candidate ranks are not finalized."

Without clear idempotence/no-op semantics, product code can accidentally treat repeated calls as successful transition work even when it should simply stop applying. In earlier testing, the dangerous shape was repeated or drifting application around the same high-level intent, which could strand subsequent app writes on stale tentative state.

What would help:

- Return an explicit result kind such as `AlreadyActive`, `CreatedTransition`, `RejectedStaleInput`, `RejectedIncompleteFormation`, or `WaitingForTransitionFinalization`.
- Include a stable formation identifier/hash and active epoch in the result.
- Support an optional precondition like `base_finalized_root` or `base_epoch` so stale callers cannot accidentally apply from an outdated view.

## Gap: Safe-To-Write Boundary Is Ambiguous

Swarmion returns transition information such as selected witnesses and `safe_to_disconnect`. That is useful, but Protos also needs a clear signal for when it is safe to author normal application writes after a witness transition.

The app-write failure shape was:

1. A witness transition was applied.
2. The product proceeded with app writes.
3. Another transition/reconciliation path caused a newer active epoch.
4. The app writes were visible as stale/tentative rather than finalized active state.

Protos fixed this by gating locally on the full intended formation and by waiting for remote heads, but the API/docs should make the safe authoring boundary explicit.

What would help:

- A transition status API that answers: "for this peer and finalized root, is the active witness epoch finalized and safe for new product writes?"
- Documentation for whether non-selected formation members can safely author writes immediately after `safe_to_disconnect` includes the old witness.
- A recommended loop for product reconcilers: catch up finalized state, inspect transition status, apply candidates only if needed, then wait for an explicit safe-to-author condition.

## Gap: Concurrent Callers Need A Stronger Contract

The model allows multiple peers to call the witness policy/candidate application API. That is desirable if all callers use the same finalized DB state, but the contract should be documented more rigorously.

Questions that should be answered in Swarmion docs/API:

- If two peers call from the same finalized root with the same candidate set, is exactly one transition created and the other a no-op?
- If one peer is behind and calls with an older candidate set, is that rejected, ignored, or converted to a no-op?
- If two peers call with overlapping but not identical candidate sets, what is the recovery path?
- How should the product distinguish "wait, transition in progress" from "fatal/conflicting transition"?

What would help:

- Require or strongly recommend a finalized-root precondition on transition application.
- Return structured stale-input/conflict statuses instead of requiring the product to infer from logs or branch state.
- Document the expected convergence behavior for duplicate and conflicting proposals.

## Gap: API Packaging / Module Boundary

The recent public API cleanup introduced a dependency on `swarmion.dev/internal/witnesstransition`. For local replace-based downstream development, Protos had to add a local module/replace for that internal package.

This is not a protocol issue, but it is a developer-experience gap. If downstream users need to consume the transition API, the module boundary should be clean and stable.

What would help:

- Publish the witness-transition support package as a normal module, or keep it fully internal to Swarmion so downstream consumers never need a direct replace.
- Document the intended import surface for downstream applications.

## What Protos Changed Locally To Avoid The Liveness Trap

Protos now waits for the complete intended candidate formation before applying a transition. For the tested mixed scenario, that means:

- laptop is present and ranked;
- both local VMs are present and ranked;
- both cloud VMs are present and ranked;
- all candidate ranks are finalized through the protocol path;
- the active formation is checked before reapplying, so an already-satisfied formation is treated as local no-op.

This makes the product behavior stable, but it is still product-side defensive logic. A stronger Swarmion API could prevent downstream integrations from having to rediscover the same rules.

## Recommended Ask To Swarmion Owner

Please clarify and/or extend the witness transition API around these areas:

1. Provide an explicit transition readiness/status API.
2. Make `ApplyWitnessCandidates` result kinds explicit and idempotent.
3. Support stale-input protection using finalized root or epoch preconditions.
4. Provide a documented safe-to-author boundary after witness transitions.
5. Document concurrent caller semantics and conflict recovery.
6. Clean up the public module/import boundary for witness-transition support packages.

The protocol behavior observed so far looks safety-stable. The product-facing risk is that a valid but premature sequence of API calls can cause application writes to stop finalizing under the intended active witness set. That should be treated as an API/liveness gap worth closing in Swarmion, even if the core protocol remains correct.
