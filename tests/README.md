# Structs integration tests

Bash scripts that broadcast real transactions against a live `structsd` chain. They are separate
from the Go test suite (`make test`), which needs no chain and no daemon.

Prerequisites for all scripts: a chain running locally, with `alice` (genesis validator) and
`bob` (faucet) in the keyring. Scripts assume home `~/.structs` and `--keyring-backend test`.

Not safe to run concurrently — one test runner per chain. Each script exits non-zero if an
assertion fails.

## test_chain.sh

Full lifecycle: player setup, guild membership, planet exploration, struct building, mining and
refining, fleet movement and combat. A fresh chain is recommended.

`make test-integration` runs this with no flags, which includes the proof-of-work phases and
takes hours. Invoke it directly when you want flags:

```bash
bash tests/test_chain.sh --skip-mining
```

- `--skip-mining` — skip ore mining, refinery build/refine, and planet raid. Avoids the slowest
  proof-of-work operations, and is usually the flag you want.
- `--extended-battle` — build all 13 fleet struct types and exercise every combat mechanic.
- `--log-battle` — write full `EventAttack` details as JSONL under `tests/battle_logs/`.
- `--resume-from N` — skip to phase N, recovering IDs by querying the running chain.

## ante_regression.sh

Replays the incident 2026-05 shape: a single transaction carrying two `MsgStructDefenseSet`
messages from the same registered player. Asserts the broadcast is rejected with `structs-ante`
code 2020 (`ErrDuplicateChargeInTx`) — a success is a regression.

Needs a player with a built combat struct and two protectable structs, which phase 1+ of
`test_chain.sh` produces:

```bash
PLAYER_KEY=player_3 \
DEFENDER_STRUCT_ID=5-1702 \
PROTECTED_STRUCT_ID_1=5-1657 \
PROTECTED_STRUCT_ID_2=5-1658 \
make test-ante-regression
```

Optionally set `NODE` (default `tcp://localhost:26657`) and `CHAIN_ID` (default
`structslocal-1`). Exits 0 when the fix holds, 1 on regression, 2 on a missing prerequisite.

## test_upgrade.sh

Exercises the full v0.15.0 to v0.16.0 governance upgrade path: bootstraps a chain on the old
binary, populates state, submits and votes the proposal, halts at the upgrade height, swaps in
the new binary, and verifies the migration. It boots its own chain, so it is not a Makefile
target and does not use an already-running one.

```bash
bash tests/test_upgrade.sh --auto-build
```

- `--auto-build` — build any missing versioned binary from the right branch.
- `--skip-populate` — skip the state population step.
- `--keep-alive` — leave the upgraded chain running at the end.

## test_chain_v0.15.0.sh

Frozen copy of `test_chain.sh` as it stood at v0.15.0. Used only by `test_upgrade.sh` to populate
pre-upgrade state; don't run it against a current chain.
