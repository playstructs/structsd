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
`proto/` and run `make proto-gen`. Also generated, from a different source:
`x/structs/types/unicode_tables.go`, via `go generate ./x/structs/types/...`.

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

Because that bit is the whole control, **every handler that moves a player's coins must demand
one**, including the ones where the spend is a side effect rather than the point. `AgreementOpen`
debits the primary address for collateral, so it requires `PermTokenTransfer` even though the
message names no address at all. An access check is not a spend check: a provider's
`PermProviderOpen` says who may contract with it, nothing about whose money may move, and an
agreement's `PermUpdate` says who may change it, not who may fund the change. When a
handler's gate depends on runtime policy, put the fixed part in `PermissionMap` and only the policy
part in the handler — a `DynamicPermissionMessages` entry makes `StructsDecorator` skip
`PermissionMap` entirely, so a message in both maps gets no ante check at all.
`TestArch_PrimaryAddressDebitsRequireTokenBit` in `app/ante` enforces this by walking the handler
sources: pair `GetPrimaryAddress` with a debiting bank call and the message must demand an asset
bit. Sweeping another address *into* the primary is a credit, not a spend, and is allowlisted there.

**Agreement teardown settles exactly once, and consumer collateral outranks provider revenue.**
Agreement teardown destroys its allocation, and allocation teardown settles its agreement, so the
two call each other. `cc.agreements` hands both legs the same `AgreementCache`, so the second leg
would pay out again: `beginTeardown()` claims the one settlement an agreement gets, and
`AllocationCache.Destroy` skips an agreement that is already `IsTearingDown()`. Every teardown path
also checkpoints the provider first, because `Checkpoint()` bills the *current* agreement load
across the whole span since the last checkpoint — change the load before checkpointing and the new
load gets billed over the old span. Skipping it strands money rather than misplacing it: once the
load is decremented and the agreement removed, no later checkpoint can account for the span it was
live, and the revenue it earned sits in the collateral pool where neither `Checkpoint()` nor a
withdrawal can reach it, both being computed from load. The solvency invariant cannot see that —
stranded revenue leaves the pool over-funded, not short — so `TestArch_TeardownPathsCheckpointBeforeMutating`
is the enforcement: it finds every path that calls `beginTeardown()` and requires
`checkpointProvider()` ahead of every payout, `AgreementLoadDecrease`, `SetStartBlock` and
`removeAgreement`. A provider's collateral pool is keyed only by provider, so all
of its agreements share one account: consumer payouts go through `payConsumer` and are exact,
provider revenue goes through `ProviderCache.SweepRevenue` and is clamped to what the pool can
spare. Never reverse that order, or one consumer's collateral funds another's payout. Teardown is
also not transactional — half of it runs in block hooks that can only log and continue — so each
path does everything that can fail (destination validation, allocation teardown) before it moves
money or changes load. The registered `provider-collateral-solvency` invariant
(`x/structs/keeper/invariants.go`) is the standing check;
`x/structs/keeper/agreement_teardown_test.go` is the regression suite.

**An expiry gets one attempt, so it may never return early.** `AgreementExpirations` reads the
expiration index at *exactly* the current height — no range scan, no retry queue — so an agreement
that is not torn down on the one block it comes up is never revisited. Returning early does not
defer the teardown, it cancels it: the agreement keeps its capacity in the provider's load and
`Checkpoint()` goes on billing that capacity against the shared pool every block afterwards, out of
other consumers' escrow. The load decrement and `removeAgreement()` are therefore the parts that
must always happen, and nothing may abort in front of them for a condition that a later block could
not fix. An allocation that is already gone is the case to know: it is nothing to tear down, not a
failure, and `destroyAllocation` treats it that way on both the path where `cc.allocations` has
never heard of it and the path where only `Destroy`'s re-read notices. `Expire()` keeps its error
return for future paths, but no live one can strand today. `agreement-expiry-liveness` is the
standing check — the solvency invariant cannot see this, because it clamps every span to the
agreement's own window and so reads an overdue agreement as a fully served one. Deleting a provider
drains both of its pools for the same reason: their addresses are derived from the provider id, so
whatever is left when the record goes is unreachable for good.

**A brownout is a normal state, so never subtract load from capacity unguarded.** Grid load is
allowed to exceed capacity: an allocation feeding a substation can shrink or disappear mid-block,
and the only thing that pushes load back under capacity is `GridCascade`, which runs in the
EndBlocker and gives up — logging `Grid Queue problem` — when there are not enough allocations to
shed. Every message in between sees the over-subscribed state. In uint64 a bare `capacity - load`
does not report "nothing available" there, it wraps to nearly 2^64, so a gate built on it reads as
unlimited headroom at precisely the moment there is none. Compare first and return zero:
`PlayerCache.GetAvailableCapacity`, `CanSupportLoadAddition`, `SetInitialPower` and
`SetDynamicPower` all do, and `SubstationCache.GetAvailableCapacity` was the one that did not — the
reachable cost being `AgreementCapacityIncrease`, whose only headroom check that was, since
`AllocationCache.SetPower` trusts its caller to have gated. Note what saved the neighbouring path
and what that hides: `AgreementOpen` read the same wrapped value but was never exploitable, because
allocation creation re-checks correctly, so the outer gate being wrong was invisible for as long as
an inner one happened to hold. `x/structs/keeper/substation_overload_test.go` is the regression
suite.

**Delete an index row with the same id you wrote it under.** `SetAutoResizeAllocationSource` keys
the auto-resize hook by *source object id*, and `AllocationCache.Destroy` cleared it with the
allocation id — a `store.Delete` on a key nobody had written, which compiles, runs, returns
nothing and removes nothing. There is no error to notice: the only signal is the row that is still
there. The consequences were both silent too. `SetSource` rejects a new automated allocation on any
source the index mentions *without checking the allocation exists*, so a leaked hook brakes that
source for good; and the infusion capacity path read the hook as proof something was tracking the
source, so it resized a missing allocation and skipped the `AppendGridCascadeQueue` that a capacity
cut owes. When you add an index, write its clear next to its set and key both off the same value —
`Destroy` gets this right one line below, in `RemoveAllocationSourceIndex`. A lookup that decides
policy should also confirm what it found is real: `AutoResizeAllocation` now separates "allocation
missing", which drops the hook and falls through to the cascade, from "resize failed", which must
not, or a transient error starts shedding load. `x/structs/keeper/allocation_autoresize_test.go`
is the regression suite.

**An agreement's service window and the provider's checkpoint clock must start on the same
block.** `Checkpoint()` bills *aggregate* provider load from `checkpointBlock`, while the solvency
invariant measures what each consumer is owed from that agreement's `StartBlock`. `AgreementOpen`
raises the load and checkpoints the provider in the opening block, so `StartBlock` is that same
height — not the next one. Offsetting them by a single block bills the provider for a block of
service no consumer received and leaves the collateral pool short by
`capacity * rate * (1 - providerCancellationPenalty)` per agreement, which is real insolvency, not
rounding. `MigrateAgreementCheckpointOverbill` in `app/upgrades/v0_21_0` is the one-time claw-back
for agreements written before the two clocks agreed.

**Changing an agreement's capacity is settlement too, and re-prices the rest of it.**
`CapacityIncrease` and `CapacityDecrease` release the voided provider cancellation penalty and
re-base the agreement's window, so they obey the teardown rules: validate everything first, then
pay, then mutate. The penalty is priced from `GetDurationPast()` and the *old* capacity, so it
must be swept after validation but before `SetStartBlock` and the capacity write, which reset the
span and the rate it is charged at. Both also re-check the provider's published capacity and
duration ranges through `AgreementCapacityVerify` / `AgreementDurationVerify` — a change re-prices
the unearned span (`rescaledDuration`), so without the duration check a decrease toward capacity 1
multiplies the remaining blocks by the old capacity and escapes the advertised maximum. A change of
zero is rejected rather than treated as a free re-base.
`x/structs/keeper/agreement_capacity_test.go` drives the cache methods directly, without the
transaction rollback that would otherwise mask a payout moving back ahead of validation.

**A destroyed struct is still there, so every handler must reject it.** Destruction does not
delete: `DestroyAndCommit` sets the Destroyed flag, leaves the Built flag alone and deliberately
leaves the struct in its planet or fleet slot, and `StructSweepDestroyed` only clears the slot and
removes the object `StructSweepDelay` blocks later. For those blocks the struct loads normally and
reads as built and offline. That window produced two bugs: destruction replayed through the slot
`AttemptComplete` still walks, releasing one `BuildDraw` reservation and one `typeCount` twice; and
`StructActivate` brought rubble back online, so `GoOnline` re-added the owner's load and the
planet's shield and defensive counters and the sweep then deleted the struct without reversing
them. Both readiness checks now reject a destroyed struct and `DestroyAndCommit` is idempotent, but
neither is a substitute for the handler asking — a handler that only requires the struct to be
online is *incidentally* safe, and stops being safe the moment anything can bring rubble back.
`x/structs/keeper/arch_struct_test.go` requires every handler calling `cc.GetStruct(` to establish
the struct is not destroyed; `x/structs/keeper/struct_destroyed_guards_test.go` is the regression
suite, and `MigrateStructPhantomAggregates` in `app/upgrades/v0_21_0` recomputes what the two bugs
corrupted.

**Commit cache maps in sorted key order, never in map order.** `CommitAll` goes through
`commitCaches`, which sorts. That is not tidiness: Go randomizes map iteration order per process,
and `AddressCache.Commit` reaches `SetPlayerIndexForAddress`, which allocates an auth account
number from the account keeper's *global monotonic sequence* for any address that lacks one. Two
addresses committed in map order therefore got different account numbers on different nodes —
divergent auth state and a different app hash, reachable both from a genesis `AddressList` and
from one transaction carrying two `AddressRegister` messages. Account allocation is the only
order-dependent write among the eighteen caches; every other `Commit` writes keys derived from its
own map key, and the `Append`-named ones are keyed by object id rather than a counter. Note the
split inside that setter: writing the index row is an ordinary KV write keyed by the address string
and happens for any key, while *provisioning the auth account* is the part that consumes a sequence
and is refused for a string that is not a real address. Sorting is
what makes the order a property of the data rather than of the runtime, so a new cache map must be
committed through the helper and never ranged. `x/structs/keeper/arch_commit_test.go` enforces
three things: no bare range over a cache map inside `CommitAll`, every cache map field on
`CurrentContext` actually committed (a map nobody commits drops its writes silently), and
`NewAccountWithAddress` confined to an allowlist, since it is the only call in the module that
consumes a sequence and so the only one whose result depends on when it runs. **The rule is the
general one, not the account-number one**: anything a `Commit` touches that is not keyed by that
cache's own key needs this same scrutiny.

**Consensus code may not ask the toolchain about Unicode.** Go resolves `\p{L}` in a regexp, and
`unicode.Is` against `unicode.L` / `Mn` / `Me` / `Cf`, using the tables of whichever toolchain
compiled the binary, and those tables grow with Go releases — U+088F is unassigned in Unicode
15.0.0 and a letter later. Nothing pins a toolchain hard enough to stop two validators disagreeing:
`go.mod`'s `toolchain` directive is a floor, not a ceiling, and `GOTOOLCHAIN=local` opts out
entirely. So a name made of one boundary code point validated differently on different nodes, only
the accepting node wrote it, and that is an app hash split. Classify against the checked-in Unicode
15.0.0 tables instead, through the helpers in `x/structs/types/unicode_pinned.go`
(`isPinnedLetter`, `pinnedToLower`, `isPinnedSpace`, `asciiToLower`). Note the distinction:
`unicode.Is(pinnedL, r)` is a binary search over data we control and is fine; it is naming the
*standard library's* tables that is not. The same goes for case folding and whitespace —
`strings.ToLower`, `strings.EqualFold` and `strings.TrimSpace` all read those tables — and
`NormalizeName` is the sharpest case because its output is a KV key rather than a comparison value:
`SetGuildNameIndex` writes `"Guild/name/" + NormalizeName(name)`. `x/structs/types/arch_determinism_test.go`
enforces all of this and takes an adjacent `// DETERMINISM_OK: <reason>` comment for a value that
provably cannot reach state. `norm.NFC` is the one allowed external dependency, pinned by `go.sum`;
`TestNFCGoldenVectors` fails if a `go get -u` moves it. **Bumping `PinnedUnicodeVersion` or
`golang.org/x/text` changes which names the chain accepts and needs an upgrade handler** — treat a
Unicode-version review as part of writing one.

**A proto3 enum is an open int32, so a switch on one that decides authorization must end in a
default that denies.** The generated decoder shifts bytes into an int32 and never consults the
enum, so a field typed `guildJoinBypassLevel` holds any number a transaction or a genesis file
puts there. `CanRequestMembership`, `CanApproveMembershipRequest` and `CanInviteMembers` each
switched on one with no default, and a Go switch that matches nothing simply falls through — which
for a `(err error)` named return means `nil`, which means allowed. The undeclared value was
therefore *more* permissive than every declared one: `permissioned` demands `PermGuildMembership`
and `member` demands existing membership, while 500 demanded nothing. A player holding only
`PermGuildJoinConstraintsUpdate` stored it, and any registered outsider then submitted and approved
their own membership request. Note the shape, because it is what makes this class hard to see in
review: the bug is in the branch nobody wrote, and the write that enabled it — the update handler
assigning `msg.GuildJoinBypassLevel` — looks like an ordinary permissioned setter.

Fix it at both ends and know which end does what. The default branches are what covers records
already on disk, and they are the security fix. Validation on the way in —
`GuildJoinBypassLevel.IsValid()`, checked in the cache setters so future callers inherit it, and
again in `GenesisState.Validate` because `GenesisImportGuild` assigns a whole record and skips
them — is what stops new ones. `IsValid` reads the generated `GuildJoinBypassLevel_name` map rather
than listing the constants, so a value added to the proto is storable the moment it is generated
while still being denied by the switches until someone writes policy for it; that pairing is
deliberate. `TestArch_GuildBypassSwitchesHaveDefault` enforces the defaults and
`x/structs/keeper/guild_bypass_level_test.go` is the regression suite;
`MigrateGuildJoinBypassLevels` clamps existing corruption to `closed`. Other enums switched on in
the keeper (`Ambit`, `ObjectType`) mostly dispatch rather than authorize and carry their own
defaultless-switch backlog — the rule is about the ones where falling through grants something.

**Use the typed errors.** Keeper codes (1050–1800) live in
`x/structs/types/errors_structured.go`, ante codes (2000–2050) in `app/ante/errors.go`. The
numbers are a public contract that clients and tests assert on: never `fmt.Errorf` out of a
handler, and never renumber an existing code.

**Adding a new message** means all of:

1. `proto/` change, then `make proto-gen`.
2. `x/structs/keeper/msg_server_<name>.go` plus a `_test.go` beside it.
3. An entry in every applicable map in `app/ante/maps.go`. There are ten —
   `KnownStructsMessages`, `PermissionMap`, `DynamicPermissionMessages`, `ChargeMessages`,
   `ProofMessages`, `SignatureMessages`, `CreatorExtractors`, `ThrottleKeyExtractors`,
   `ThrottleTargetAuth`, `FreeStakingMessages` — plus the `IsFreeTransaction` helpers. Walk the
   whole file. A missing entry fails silently: the message ends up unpriced, unpermissioned, or
   unthrottled rather than erroring.
4. CLI exposure via `x/structs/module/autocli.go`.

**`app/ante` is incident territory.** Any `ctx.IsCheckTx()` or `IsReCheckTx()` short-circuit needs an adjacent `// SKIP_RATIONALE:` comment
or an allowlist entry, or `app/ante/arch_test.go` fails. Route new reject branches through
`observeReject` (see `docs/observability.md`).

**An ante write is the one write a failed message cannot undo, so never reserve on a
transaction's say-so.** The SDK commits the ante cache before it runs messages and only discards
the *message* cache when one fails, so anything the ante wrote outlives the handler's rejection.
`ThrottleDecorator` reserves object-global throttle keys built from fields the transaction
chooses — `proof/<structId>`, `fleet/<fleetId>`, `explore/<playerId>` — and nothing upstream
authorizes the *object*: `StructsDecorator` is Layer 1 only, checking the signing address's own
bits, and a primary address holds `PermAll`, so naming somebody else's struct sails through and
parks that object's slot for the block. Reservation is therefore gated on
`Keeper.ThrottleTargetAuthorized`, which resolves the named struct, fleet or player to its owner
and runs the handlers' own `PermissionCheck` — never a reimplementation of the policy, and never
owner equality, which would stop throttling every delegated action. `ThrottleTargetAuth` in
`maps.go` is the mirror table and `TestArch_ThrottleTargetAuthMatchesHandlers` reads the handler
sources to keep the permission honest.

Note both halves of the shape, because each is load-bearing. **The write cannot move later**: a
post-handler runs against the message cache and is discarded on failure, and a proof slot that a
failed attempt does not consume lets a player grind nonces on-chain rather than mine off-chain.
**The refusal must not reject**: the ante sees pre-transaction state while a handler sees what
earlier messages in the same transaction left, so a transaction that grants a permission and then
uses it is authorized there and not here — skipping the reservation costs one block of throttling,
rejecting would break the transaction. Any future ante-time reservation keyed by a caller-supplied
id inherits all of this.

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
