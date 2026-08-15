# Protobuf contract history

This directory contains immutable audit snapshots of superseded protobuf CUE
contracts. They preserve the schema that existed before an intentional breaking
transition without keeping old RPCs, fields, handlers, or readers in the runtime.

The active generators intentionally read only the contracts under
`contracts/proto/*/v1`. Files here are history-only inputs: they are not
compiled, generated, or accepted as compatibility paths.

`apic/v0_0` and `p2p_instance/v0_0` are the source side of the explicit
v0.0-to-v0.1 breaking transitions recorded by their active contracts.

`SHA256SUMS` pins each snapshot byte-for-byte, and
`scripts/verify-contracts.sh` verifies both the manifest and each archived
schema's boundary with its live v0.1 breaking transition. The v0.0 snapshots
were captured from repository commit
`25aac8623d44f521cf1c6e68282e4ca0572d47df`.
