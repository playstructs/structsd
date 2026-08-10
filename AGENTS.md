# AGENTS.md

Cosmos SDK chain for Structs. Game logic lives in `x/structs/{keeper,types,module,client}`;
`app/` wires the chain together and `app/ante/` holds the custom fee, charge, permission and
throttle gating. `proto/` is the source of truth for wire types; `api/` and every `*.pb.go` are
generated. Run `make help` for the task list, `readme.md` for build and dev setup.

The Go module is literally named `structs`, so imports are `structs/x/structs/types`, not a
GitHub path.

**Nothing runs your tests but you.** The only CI workflow is a goreleaser release on `v*` tags.

## Rules

**Don't mass-format or mass-lint.** `make format` (gofumpt) would rewrite ~213 non-generated
files, and `make lint` currently fails with a 148-issue backlog. Both predate your change. Match
the style of the file you're in and touch only the lines you're changing. If you want lint
feedback, scope it: `golangci-lint run ./x/structs/keeper/...`.

**Never hand-edit generated code**: `*.pb.go`, `*.pb.gw.go`, `*.pulsar.go`, `api/`. Change
`proto/` and run `make proto-gen`.

**Follow the handler pattern.** Every `msg_server_*.go` has the same shape: unwrap the SDK
context, `cc := k.NewCurrentContext(ctx)`, load entities through `cc.GetX(...)`, check permission
and charge (usually via methods on the loaded objects), mutate, then `cc.CommitAll()`. Read a
neighbouring handler before writing a new one. Don't write to the KV store directly when a cache
exists — caches deduplicate loads per operation and commit once.

**Load the signer with `cc.GetSigningPlayer(msg.Creator)`, and nothing else with it.** A player
owns many addresses, each with its own permission bitfield, but `cc.players` is keyed by player
id, so every address of a player resolves to one shared `PlayerCache`. The acting identity
therefore lives on the `CurrentContext`, written once by `GetSigningPlayer`, and is what every
address-level permission check reads. Any other address a message names (`FromAddress`,
`DelegatorAddress`, `Address`) is a subject, not an identity: load it with the pure
`cc.GetPlayerByAddress(...)`. `x/structs/keeper/arch_signer_test.go` enforces this. The signing
key's own bits are a hard ceiling — a key may never exercise a permission it does not itself
hold, and when a handler overwrites a permission bitfield it must check the bits being destroyed
as well as the bits being written.

A subject address deliberately does **not** have to sign. `PlayerSend` debits `FromAddress`, and
the reactor handlers stake against `DelegatorAddress`, on the authority of the signer's own
`PermTokenTransfer` / `PermTokenInfuse` bit — every address of a player is one wallet, and that
bit is what authorizes moving its coins. This reads like a missing signature check and is not one:
audits keep proposing that the debited address be a required signer, which would break delegating
token operations to a secondary key. The bit on the signing key is the control, so grant it as
narrowly as that implies.

**Agreement teardown settles exactly once, and consumer collateral outranks provider revenue.**
Agreement teardown destroys its allocation, and allocation teardown settles its agreement, so the
two call each other. `cc.agreements` hands both legs the same `AgreementCache`, so the second leg
would pay out again: `beginTeardown()` claims the one settlement an agreement gets, and
`AllocationCache.Destroy` skips an agreement that is already `IsTearingDown()`. Every teardown path
also checkpoints the provider first, because `Checkpoint()` bills the *current* agreement load
across the whole span since the last checkpoint — change the load before checkpointing and the new
load gets billed over the old span. A provider's collateral pool is keyed only by provider, so all
of its agreements share one account: consumer payouts go through `payConsumer` and are exact,
provider revenue goes through `ProviderCache.SweepRevenue` and is clamped to what the pool can
spare. Never reverse that order, or one consumer's collateral funds another's payout. Teardown is
also not transactional — half of it runs in block hooks that can only log and continue — so each
path does everything that can fail (destination validation, allocation teardown) before it moves
money or changes load. The registered `provider-collateral-solvency` invariant
(`x/structs/keeper/invariants.go`) is the standing check;
`x/structs/keeper/agreement_teardown_test.go` is the regression suite.

**Use the typed errors.** Keeper codes (1050–1800) live in
`x/structs/types/errors_structured.go`, ante codes (2000–2050) in `app/ante/errors.go`. The
numbers are a public contract that clients and tests assert on: never `fmt.Errorf` out of a
handler, and never renumber an existing code.

**Adding a new message** means all of:

1. `proto/` change, then `make proto-gen`.
2. `x/structs/keeper/msg_server_<name>.go` plus a `_test.go` beside it.
3. An entry in every applicable map in `app/ante/maps.go`. There are nine —
   `KnownStructsMessages`, `PermissionMap`, `DynamicPermissionMessages`, `ChargeMessages`,
   `ProofMessages`, `SignatureMessages`, `CreatorExtractors`, `ThrottleKeyExtractors`,
   `FreeStakingMessages` — plus the `IsFreeTransaction` helpers. Walk the whole file. A missing
   entry fails silently: the message ends up unpriced, unpermissioned, or unthrottled rather
   than erroring.
4. CLI exposure via `x/structs/module/autocli.go`.

**`app/ante` is incident territory.** Any `ctx.IsCheckTx()` or `IsReCheckTx()` short-circuit needs an adjacent `// SKIP_RATIONALE:` comment
or an allowlist entry, or `app/ante/arch_test.go` fails. Route new reject branches through
`observeReject` (see `docs/observability.md`).

**State-breaking changes need an upgrade handler** in a new `app/upgrades/vX_Y_Z/`, added to `upgradesList` in `app/app.go`. Never change field numbers or the meaning of
a stored proto field without a migration.

**Events are a public API.** Keeper events flow to `GRASS` and `structs-pg`, then every client.
Add fields; don't rename, remove, or repurpose them.

## Tests

Two separate worlds. Don't confuse them.

**Go tests — `make test`.** Runs `go test ./...` over all 154 `*_test.go` files. No chain, no
daemon; keeper tests build an in-memory keeper via `testutil/keeper.StructsKeeper(t)`. These are
not purely unit tests — the suite also covers upgrade migrations and an in-process app with real
staking and slashing wired up. Takes about four minutes cold. It passes today, so any failure is
yours. Run it before reporting done. (`make test-unit` is the same packages with a different
timeout; ignore it.)

**Live-chain scripts — `tests/*.sh`.** These broadcast real transactions against a running local
chain and cannot run concurrently. Ask before starting one.

- `test_chain.sh` — full lifecycle, the big one, and the only script worth running for general
  verification. `make test-integration` runs it *without* flags, so proof-of-work makes it take
  hours; invoke it directly with `--skip-mining` instead. Needs `alice` and `bob` in the keyring.
- `ante_regression.sh` — `make test-ante-regression`, but needs four env vars (`PLAYER_KEY`,
  `DEFENDER_STRUCT_ID`, `PROTECTED_STRUCT_ID_1`, `PROTECTED_STRUCT_ID_2`) *and* existing state
  from `test_chain.sh`.
- `test_upgrade.sh` — boots its own chain from versioned binaries. Not a Makefile target.

See `tests/README.md` for the full flag and prerequisite reference.

## Don't

Commit keys or mnemonics. Push a `v*` tag (that triggers the release workflow). Change genesis or
params in `config.yml` / `chain.json` unless that is the task. Broadcast a transaction against
anything but a local dev chain without being asked. The chain has no undo.

## Game rules and the docs corpus

`structsd` enforces the rules of Structs but does not explain them. If your task touches
observable game behaviour — charge costs, permission bits, combat resolution, energy,
proof-of-work difficulty — load the game corpus first: <https://structs.ai/llms.txt>, or clone
<https://github.com/playstructs/structs-ai> into `.references/`, which is gitignored, read-only
scratch space. Otherwise skip it. It is a player-facing corpus and most chain work doesn't need
it.

**This repository wins.** If `structs-ai` and the code disagree, the code is correct and the doc
is a bug — fix it upstream, don't bend the chain to match a document. When you change a game
rule, say so explicitly in your summary, because `structs-ai` describes that rule to players and
will need updating there too.
