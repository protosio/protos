---
name: swarmion-integration-tests
description: Run Swarmion integration tests against the simulator, a local Docker engine, or a reusable cloud VM Docker context. Use when Codex needs to execute, debug, or explain Swarmion integration-test workflows, including deploying or destroying reusable cloud test VMs, reusing Docker cache volumes, running Docker-backed tests from inside the runner container, cleaning up peer containers, and copying artifact volumes back locally.
---

# Swarmion Integration Tests

## Core Rule

Keep cloud VM lifecycle separate from test execution.

- Use `integration-tests/tooling/cloud_vm.sh` only to `deploy`, `status`, `list`, or `destroy` a reusable cloud Docker host.
- Use `integration-tests/runtime/docker/run-docker-backend.sh` to run Docker-backed tests on either the current Docker context or a named remote Docker context.
- Do not re-provision a VM for each test run. Reuse the Docker context until the user explicitly asks to destroy the VM.
- The helper image from `integration-tests/tooling/Dockerfile.cloud` is infra-only: it creates/destroys VMs and installs/verifies Docker on the VM over SSH. It must not contain the Docker CLI or create/remove local Docker contexts.
- The host-side `cloud_vm.sh` wrapper owns local Docker context creation, reachability checks, and context removal using the laptop's Docker CLI.

## Local Docker Run

From the Swarmion repo root:

```sh
task test:integration-runtime
```

For direct runner use:

```sh
integration-tests/runtime/docker/run-docker-backend.sh -v -count=1 -timeout 30m ./...
```

The runner builds the test-runner image on the selected Docker context, uploads the current `swarmion`, `dolt/go`, and `doltsqldriver` workspace into a named Docker volume, mounts named Go cache volumes, runs `go test` inside the runner container, stores artifacts in a named artifact volume, then copies that volume locally with `docker cp`.

## Local Simulator Run

From the Swarmion repo root:

```sh
task test:integration-simulator
```

For direct runner use:

```sh
integration-tests/runtime/simulator/run-simulator-backend.sh -p=1 -v -count=1 -timeout 30m ./...
```

The simulator runner executes host-side `go test` from `integration-tests/`,
sets `SWARMION_TEST_BACKEND=simulated`, writes a run bundle under
`perf-artifacts/simulator-runs/<host>-<timestamp>/`, and runs the same
`swarmion-observe summarize` analyzer as the Docker runner unless
`SWARMION_OBSERVABILITY_ANALYZE=0` is set.

Simulator network delay can be enabled with Go duration values:

```sh
SWARMION_SIMULATOR_LATENCY=2ms \
SWARMION_SIMULATOR_JITTER=1ms \
integration-tests/runtime/simulator/run-simulator-backend.sh -v -count=1 -timeout 30m ./...
```

## Ranged Full-Suite Scaling Sweep

Use `TestRangedFullIntegrationSuite` when the user wants one integration test
to exercise the protocol scenario suite across a sequence such as
`10,20,30,...,100` peers and produce one suite report plus one logical report
per peer count.

Default Docker full sweep:

```sh
SWARMION_TEST_STRESS=1 \
SWARMION_ARTIFACT_COPY_MODE=summary \
integration-tests/runtime/docker/run-docker-backend.sh \
  -p=1 -v -count=1 -timeout 150m \
  -run '^TestRangedFullIntegrationSuite$' . \
  -args -start-peer-range=10 -end-peer-range=100 -peer-count-step=10
```

Short Docker sanity sweep:

```sh
SWARMION_TEST_STRESS=1 \
SWARMION_ARTIFACT_COPY_MODE=summary \
SWARMION_DOCKER_LIVE_PPROF=1 \
SWARMION_DOCKER_PPROF_CPU_SECONDS=10 \
integration-tests/runtime/docker/run-docker-backend.sh \
  -p=1 -v -count=1 -timeout 90m \
  -run '^TestRangedFullIntegrationSuite$' . \
  -args -start-peer-range=10 -end-peer-range=30 -peer-count-step=10
```

Short simulator sanity sweep:

```sh
SWARMION_TEST_STRESS=1 \
integration-tests/runtime/simulator/run-simulator-backend.sh \
  -p=1 -v -count=1 -timeout 30m \
  -run '^TestRangedFullIntegrationSuite$' . \
  -args -start-peer-range=10 -end-peer-range=30 -peer-count-step=10
```

Important behavior:

- The ranged suite is stress-gated and runs on both Docker and simulator.
- Each peer-count step runs the protocol scenarios as subtests and resets/rebootstraps the active peers before every scenario.
- `-start-peer-range`, `-end-peer-range`, and `-peer-count-step` are Go test flags, so pass them after `-args`.
- Docker grows one compose project by starting missing peer containers as the peer count increases.
- Docker orchestration uses each peer's container-local admin HTTP endpoint for status, queries, commits, reset, hardening epoch installation, and observability flushes. This keeps harness control traffic out of the protocol libp2p overlay.
- Existing active peers are reset in place. Reset clears repository state, increments runtime generation, and reboots the `swarmiond` app into a fresh namespace without restarting the container.
- The simulator uses the same scenario orchestration and artifact schema, but recreates in-process peer hosts for logical resets rather than resetting long-lived containers in place.
- For bounded-connectivity simulator sweeps, new in-process peers are admitted into the bounded overlay before runtime bootstrap so their initial bootstrap RPCs use direct prepared neighbors instead of overfilling an already-saturated peer.
- In bounded-connectivity stress runs, the fresh-bootstrap scenario seeds history through the init peer, then bootstraps later peers from the active overlay instead of forcing every joiner to connect directly to one seed.
- Peer-range phases set a lightweight observability scope through the admin observability-flush RPC. Lifecycle summaries, fast metrics, protocol RPC events, and pprof metadata can carry `scenario_id`, `run_id`, `step_id`, `phase_id`, `phase`, `logical_peer_count`, and `active_peer_count` without enabling raw lifecycle output.
- Runtime instrumentation is centralized in `runtime/observe`; core packages should use `observe.Event`, `observe.Scope`, `observe.RecordLifecycle`, and `observe.Do`. Default builds keep this enabled, while `-tags swarmion_observe_disabled` compiles pprof labels and lifecycle recording to no-ops without removing cheap admin fast metrics.
- The peer lifecycle writer/aggregator lives in `runtime/observe/lifecycle` and is configured through `observe.ConfigFromEnv`. Fast-metrics snapshots include `observability_overhead` counters for lifecycle records, exemplar/raw bytes, summary bytes, and flush durations.
- The Docker compose project is still torn down when the test completes.

## Reusable Cloud VM Run

Deploy the VM once:

```sh
integration-tests/tooling/cloud_vm.sh --build-image deploy
```

Use the Docker context printed by deploy for every remote test run:

```sh
integration-tests/runtime/docker/run-docker-backend.sh --context <docker-context> -v -count=1 -timeout 150m ./...
```

Run one stress scenario:

```sh
SWARMION_TEST_STRESS=1 \
SWARMION_TEST_PEERS=50 \
integration-tests/runtime/docker/run-docker-backend.sh \
  --context <docker-context> \
  -v -count=1 -timeout 150m -run '^TestLateJoinerCatchesUp$' ./...
```

Destroy only when done reusing the VM:

```sh
integration-tests/tooling/cloud_vm.sh destroy --context <docker-context>
```

Destroy removes both the cloud VM and the local Docker context.
The VM destruction happens inside the helper container; local context removal happens in the host wrapper after the helper returns.

## Artifact Handling

Expect each Docker runner invocation to create a local bundle under:

```text
perf-artifacts/docker-runs/<context>-<timestamp>/
```

Expect each simulator runner invocation to create a local bundle under:

```text
perf-artifacts/simulator-runs/<host>-<timestamp>/
```

Both backends use backend-neutral per-test raw artifacts under
`tests/<test>__<backend>-<timestamp>/`. The analyzer still accepts legacy
`docker-tests/...` bundles for older runs. Docker artifacts originate in a named
Docker volume on the selected Docker engine and are copied back locally with
`docker cp`; simulator artifacts are written locally during the host-side test
run.

Use these overrides when needed:

```sh
SWARMION_ARTIFACT_DIR=/tmp/swarmion-it-artifacts \
SWARMION_KEEP_ARTIFACT_VOLUME=1 \
integration-tests/runtime/docker/run-docker-backend.sh --context <docker-context> ./...
```

## Observability Reports

Both backend runners run the analyzer by default after `go test`:

```sh
go run ./cmd/swarmion-observe summarize /workspace/code/swarmion/perf-artifacts
```

For simulator, the same command is run locally against the simulator run
directory. After artifacts are available locally, open the run directory and
expect:

- `suite-summary.json` and `suite-report.html`: the common comparative layer across peer counts. It includes data coverage/validation, metric definitions, latency sample counts and statuses, scaling decomposition, resource saturation, and outlier exemplars. A single peer-count run still gets a one-row suite summary/report.
- `summary-<n>-peers.json` and `report-<n>-peers.html`: the individual run summary/report for the logical peer-count step. Each pair contains headline KPIs, work units, histograms, peer summaries, data coverage, data-quality warnings, resource saturation, profile coverage, finalization timelines, content root closures, protocol amplification, protocol RPC events, CPU/profile index, outlier exemplars, and outliers; the suite report links back to these files.
- `report-data/*.json`: suite-level chart extracts such as `suite-metrics-coverage.json`, `suite-scaling-decomposition.json`, `suite-outlier-exemplars.json`, `suite-resource-saturation.json`, and `suite-metric-dictionary.json`.

The analyzer no longer writes root-level `data.json`, `summary.json`, `summary.md`, or `report.html`; those names were the old aggregate/report aliases and should not be expected in new bundles.

CPU fields in suite summaries are nullable when pprof was not collected for that logical peer count. Do not interpret missing pprof as zero CPU; check `metrics_coverage_by_run` and `profile_coverage_by_run` first.

Aggregate run directories currently include only suite-level chart extracts under `report-data/`, including `suite-run-table.json`, `suite-normalized-metrics.json`, `suite-latency-quantiles.json`, `suite-histograms.json`, `suite-top-outliers.json`, `suite-metrics-coverage.json`, `suite-scaling-decomposition.json`, `suite-outlier-exemplars.json`, `suite-resource-saturation.json`, `suite-metric-dictionary.json`, and `suite-report-validation.json`.

Per-test raw observability lives under `tests/<test>__<backend>-<timestamp>/`:

- `manifest.json`: `swarmion.run_manifest.v1` with backend, peers, artifact mode, selected env, git metadata, and schema names.
- `harness.ndjson`: `swarmion.harness_event.v1` wait, phase, assertion, and live-profile records.
- `peer-fast-metrics/<peer>.json`: `swarmion.metrics_snapshot.v1` from the cheap admin fast-metrics endpoint. These snapshots include protocol progress, queues, content counters, Go runtime gauges, network counters, durable head, protocol counters, and protocolnet envelope/RPC metrics.
- `peer-lifecycle-summary/<peer>.json`: compact `swarmion.lifecycle_summary.v1` copied from each peer's observability directory. This is the default analyzer input and carries bounded counters, duration histograms, time-window snapshots, and top slow/error exemplars.
- `peer-lifecycle-exemplars/<peer>.ndjson`: retained raw diagnostic lifecycle records: errors, retries, blocked states, slow-tail records, and configured successful samples.
- `peer-lifecycle/<peer>.ndjson`: optional raw `swarmion.lifecycle.v1` copied from `/data/observability/lifecycle.ndjson` only when lifecycle mode is `sampled` or `full`. Raw records carry event/root lifecycle stage, purpose, IDs, provider, attempt, result, duration, and error fields.
- Peer-range tests also copy step-scoped lifecycle summaries under `peer-range-runs/peers-<n>/peer-lifecycle-summary/` after calling the admin observability-flush RPC. These are the source for lifecycle stage P99 and hot-window charts in `report-<n>-peers.html`; missing step summaries should show lifecycle-not-collected placeholders rather than blank charts.
- `peer-pprof/<peer>/`: Docker-only pprof snapshots for selected diagnostic peers. Live profiles/traces are under `peer-pprof/<peer>/live-wait-<phase>/`. Simulator reports mark pprof as not collected.
- `peer-status/`, `peer-trace-status/`, and `peer-content-trace/`: deep diagnostics for selected peers, every peer on failure, or every peer in full diagnostics mode.

Key controls:

```sh
SWARMION_OBSERVABILITY_ANALYZE=0              # skip run-level analyzer
SWARMION_OBSERVABILITY_TOP_K=5               # default 3, capped at 20
SWARMION_OBSERVABILITY_FULL_DIAGNOSTICS=1    # collect deep diagnostics for all peers
SWARMION_DOCKER_ARTIFACT_MODE=full           # equivalent full-diagnostics mode
SWARMION_LIFECYCLE_MODE=aggregate            # aggregate|sampled|full, default aggregate
SWARMION_LIFECYCLE_SUCCESS_SAMPLE_RATE=0     # deterministic success sampling in sampled mode
SWARMION_LIFECYCLE_SLOW_THRESHOLD_MS=1000    # slow-tail exemplar threshold
SWARMION_LIFECYCLE_TOP_N=50                  # bounded slow/error exemplar count
SWARMION_LIFECYCLE_SNAPSHOT_INTERVAL=10s     # compact time-window snapshots
SWARMION_LIFECYCLE_MAX_RAW_BYTES_PER_PEER=0  # optional raw lifecycle cap in sampled/full
SWARMION_DOCKER_PPROF_CPU_SECONDS=10         # CPU profile duration, capped at 30s
SWARMION_DOCKER_COLLECT_RUNTIME_TRACE=0      # disable short runtime traces
SWARMION_DOCKER_RUNTIME_TRACE_SECONDS=5      # runtime trace duration, capped at 30s
SWARMION_DOCKER_LIVE_PPROF=0                 # disable first-wait live profile collection
SWARMION_DOCKER_PPROF_TRIGGER_QUEUE_AGE_MS=1000      # peer-range live profile trigger; 0 disables
SWARMION_DOCKER_PPROF_TRIGGER_ROOT_INFLIGHT=128      # peer-range live profile trigger; 0 disables
SWARMION_DOCKER_PPROF_TRIGGER_REPAIR_RPCS=200        # peer-range live profile trigger; 0 disables
SWARMION_SIMULATOR_LATENCY=2ms                       # simulator peer-to-peer transport delay
SWARMION_SIMULATOR_JITTER=1ms                        # simulator peer-to-peer jitter
```

To regenerate reports after a run, use a full artifact bundle with
`tests/<test>/` raw inputs present:

```sh
cd integration-tests
go run ./cmd/swarmion-observe summarize ../perf-artifacts/docker-runs/<context>-<timestamp>
go run ./cmd/swarmion-observe summarize ../perf-artifacts/docker-runs/<context>-<timestamp>/tests/<test>
go run ./cmd/swarmion-observe summarize ../perf-artifacts/simulator-runs/<host>-<timestamp>
go run ./cmd/swarmion-observe compare <run-dir> [<run-dir>...]
```

The analyzer prefers compact `peer-lifecycle-summary/*.json` lifecycle inputs
and reads raw `peer-lifecycle/*.ndjson` only when sampled/full lifecycle mode
produced them. In peer-range reports it prefers the step-scoped
`peer-range-runs/peers-<n>/peer-lifecycle-summary/*.json` files first.

Persist a suite-level scaling explorer from separate run directories:

```sh
cd integration-tests
go run ./cmd/swarmion-observe compare \
  --out ../perf-artifacts/suite-summary.json \
  ../perf-artifacts/docker-runs/<run-10> \
  ../perf-artifacts/docker-runs/<run-20> \
  ../perf-artifacts/docker-runs/<run-30>
```

`compare --out` writes the requested `suite-summary.json`, a sibling `suite-report.html` scaling explorer, and suite-level `report-data/*.json` comparison extracts.

Use `integration-tests/tooling/observability_matrix.sh` for peer-count sweeps. It defaults to Docker, peer counts `5 10 25 50 100`, and startup/batch/concurrent-write scenarios, then writes `compare.json` when analyzer-compatible run directories exist. Override `SWARMION_OBSERVABILITY_MATRIX_PEERS`, `SWARMION_MATRIX_BACKEND`, `SWARMION_MATRIX_RUN`, `SWARMION_MATRIX_TIMEOUT`, or `SWARMION_MATRIX_ARTIFACT_DIR`. Local Docker runs above `SWARMION_LOCAL_MATRIX_PEER_LIMIT` default to skip unless `SWARMION_OBSERVABILITY_MATRIX_LARGE=1` is set or a non-default Docker context is selected.

Fast-metrics and trace-probe admin RPCs are defined in `runtime/adminrpc/adminrpc.go`, implemented in `runtime/app/app.go`, and used by the integration harness through `integration-tests/harness/control.go`. Prefer fast metrics for repeated checks and trace-probe for targeted event/root debugging instead of full node snapshots.

## Cache And Cleanup Expectations

The runner reuses these named volumes on the selected Docker engine:

- `swarmion-go-build-cache`
- `swarmion-go-mod-cache`
- `swarmion-it-workspace`

The current checkout is uploaded into the workspace volume on every run. Go dependency and build caches remain warm across repeated runs on the same local engine or remote VM.

Each Docker-backed test tears down its peer compose project after collecting artifacts. After the package run, the runner also removes leftover `swarmion-it-*` containers and networks as a safety cleanup.

## Useful Commands

List reusable cloud VM state:

```sh
integration-tests/tooling/cloud_vm.sh list
```

Check a context:

```sh
integration-tests/tooling/cloud_vm.sh status --context <docker-context>
docker --context <docker-context> info
```

Run the simulator backend:

```sh
task test:integration-simulator
```

Never run the Docker backend with host-side `go test`; the harness rejects it intentionally because the control peer must run inside the Docker runner container.
