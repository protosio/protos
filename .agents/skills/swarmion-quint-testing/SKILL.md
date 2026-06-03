---
name: swarmion-quint-testing
description: Use when working in the Swarmion repo and needing to run or explain Quint sampling profiles, Quint TLC/Apalache finalized-boundary proofs, cloud profile sampling, or Go ITF conformance tests.
---

# Swarmion Quint And Conformance Testing

Use this skill from `/Users/al3x/code/protos/code/swarmion`. It is the single entry point for the model-level test spectrum: Quint sampling profiles, cloud sampling, finalized-boundary TLC/Apalache checks, and Go ITF conformance replay.

## Sampling Profiles

The profile manifest is `specs/profiles/profiles.json`. It names each scenario, peer count, root witnesses, witness-selection limit, profile module, default samples/steps, and invariant subset.

Profile workspaces are generated into temporary directories by `specs/tooling/quint_profiles.py`. The generator copies the specs and patches the copied `specs/model/swarmion_model.qnt`; it does not mutate checked-in Quint specs. Patched fields include `PEERS`, witness lists/sets, `WITNESS_SELECTION_LIMIT`, configured witness ranks, initial witness ranks, and the finite child-formation set helper.

Common local commands:

```sh
specs/tooling/quint_profiles.py list
specs/tooling/quint_profiles.py typecheck --quiet
specs/tooling/quint_profiles.py sample --samples 2 --steps 4 --jobs 4 --quiet
specs/tooling/quint_profiles.py sample --profile p4_join_recovery --samples 10 --steps 12 --jobs 4 --quiet
```

Use low `--samples` and `--steps` for smoke checks. The baseline `p3_core` profile intentionally runs every invariant from `specs/swarmion_verify.qnt`, so even tiny sample counts are much slower than targeted larger-peer profiles.

The manifest defaults are the selected-suite cloud confidence tier. They keep `p3_core` broad while capping the expensive p4/p6 join profiles so a 12 VM UpCloud run remains practical. Use explicit `--samples`/`--steps` or a separate manifest for deeper stress/nightly runs.

Add new scenarios by adding one focused `specs/profiles/<name>.qnt` module plus one manifest entry. Keep profile modules small: `profile_init`, `profile_step`, and a handful of named scenario actions. Prefer targeted invariant subsets for non-baseline profiles, split positive progress/adoptability scenarios from negative fatal-state scenarios, and avoid all-formation-set-pair invariants at larger peer counts unless that cross-product is the point of the profile.

## Cloud Sampling

Use the cloud runner when local sampling is too slow or when running all profile defaults:

```sh
python3 specs/tooling/quint_cloud_runner.py \
  --profiles-manifest specs/profiles/profiles.json \
  --profile all \
  --provider upcloud \
  --vms 1 \
  --samples 100 \
  --steps 20
```

Omit `--samples` and `--steps` to use per-profile manifest defaults. Repeat `--profile <name>` for a subset. Repeat `--invariant <name>` to narrow the invariant set across selected profiles.

Use `--all-profile-invariants` only when the goal is the full Cartesian check of every `inv_*` from `swarmion_verify.qnt` against each generated profile. For large UpCloud runs, use `--run-worker-slots-per-vm <cores>` to use more than the conservative default four run workers per VM.

Cloud profile mode uploads a generated bundle with one isolated patched workspace per profile, then runs profile-qualified invariant jobs so logs and JSON results show both profile and invariant.

For overnight sampling, use detached resilient mode. It starts per-VM systemd workers and writes `<run-output-dir>/run-state.json`; SSH drops only pause local polling, not remote jobs. Resume with:

```sh
python3 specs/tooling/quint_cloud_runner.py --resume-state <run-output-dir>/run-state.json
```

Use `--poll-interval <seconds>` to reduce polling chatter and `--remote-job-timeout <seconds>` for long per-invariant jobs.

Credential loading:

- `--env-file <path>` loads dotenv-style `KEY=VALUE` lines before provider initialization.
- `QUINT_CLOUD_ENV_FILE=<path>` is the equivalent environment control.
- If `--provider upcloud` is used and `.env-upcloud` exists at the repo root, it is loaded automatically.
- Existing shell environment values win over the env file.

## TLC And Apalache Proofs

Finalized-boundary proof harnesses live under `specs/proofs/finalized_boundary`.

Local smoke/full harness:

```sh
VERIFY_TIMEOUT_SECONDS=120 specs/tooling/finalized_boundary_local_verify.sh
```

Cloud proof harness:

```sh
BACKEND=apalache STEPS=2 VMS=1 specs/tooling/finalized_boundary_cloud_verify.sh
BACKEND=tlc STEPS=2 VMS=1 specs/tooling/finalized_boundary_cloud_verify.sh
```

Use `CASE_FILTER=<substring>` to run one proof case. Use `INCLUDE_PROFILE_CASES=1` when the profile proof cases should be included. Use `ENV_FILE=<path>` to pass an explicit env file through this wrapper; otherwise the underlying runner auto-loads `.env-upcloud` for UpCloud.

## ITF Conformance

Run conformance commands from `tools/itf-conformance`. This module is outside the repo's Go workspace, so use `GOWORK=off`. Native Go tests should use the pure-Go tags from `AGENTS.md`:

```sh
cd tools/itf-conformance
GOWORK=off go test -tags=dolt_purego_zstd,gms_pure_go ./...
GOWORK=off SWARMION_ITF_TYPECHECK=1 go test -tags=dolt_purego_zstd,gms_pure_go . -run TestQuintITFGeneratedProfilesTypecheck -count=1
GOWORK=off SWARMION_ITF_CONFORMANCE=1 SWARMION_ITF_SAMPLES=2 SWARMION_ITF_STEPS=8 go test -tags=dolt_purego_zstd,gms_pure_go . -run TestQuintITFConformance -count=1
GOWORK=off SWARMION_ITF_CONFORMANCE=1 SWARMION_ITF_PROFILE_CONFORMANCE=1 go test -tags=dolt_purego_zstd,gms_pure_go . -run TestQuintITFManifestProfileConformance -count=1 -v -timeout=20m
```

Useful opt-ins include `SWARMION_ITF_MATRIX=1`, `SWARMION_ITF_NIGHTLY=1`, `SWARMION_ITF_GENERATED6=1`, `SWARMION_ITF_PROFILE_CONFORMANCE=1`, `SWARMION_ITF_KEEP_TRACES=1`, and `SWARMION_ITF_REQUIRE_ACTIONS=<comma-separated actions>`. `TestQuintITFManifestProfileConformance` replays bounded p5/p6/p7 manifest profiles through the Go implementation, including the p5 witness-selection-limit override and the p7 sibling-epoch profile wrapper. See `tools/itf-conformance/README.md` for the full scenario list.
