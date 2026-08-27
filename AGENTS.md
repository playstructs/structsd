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

**Efficiency is a correctness concern here, because every read and write is metered.** This is a
chain: a keeper read is a KV store read and somebody pays for it. On a transaction path that is
gas, and for the free Structs messages it is a ceiling rather than a bill — `GasRouterDecorator`
swaps in a 20M meter (`DefaultFreeGasCap`), so a handler that walks too much state does not charge
the player more, it stops working. In a block hook there is no meter at all: `AgreementExpirations`,
`GridCascade` and `ProcessInfusionMaturitySweep` run every block against an infinite one, so their
cost is block time that every validator pays, and work there that a transaction can grow without
bound is a halt rather than a slow query. Size a loop by asking what an attacker can put in it, not
what a normal player would.

**Answer a question about one object with an index, never by scanning every object.** The store
layout already does this: `AllocationSourceKeyPrefix`, `AllocationDestinationKeyPrefix` and
`SubstationPlayerKeyPrefix` are scoped prefixes, so iterating one costs what is attached to that
object rather than what is on the chain — deleting a substation reaches its connected players, its
outbound allocations and its inbound allocations through exactly those three, and never looks at a
provider at all. Add a new index beside the write that populates it, and clear it under the same key
it was written under. `GetAllX(ctx)` is a full store scan and belongs only where every row genuinely
is the question: upgrade migrations, genesis import and export, `EventAllGenesis` (guarded to block
1), and the registered invariants, which the crisis module runs on demand. The trap is that both
spellings exist — `Keeper.GetAllPlayerBySubstation` iterates every player and filters, while
`CurrentContext.GetAllPlayerBySubstation` reads the index, and only the second is on a live path.

The caches are part of this rather than a convenience: `CurrentContext` deduplicates loads within an
operation and writes once at `CommitAll`, so going around it to the store pays again for reads it
already has, on top of breaking the commit-ordering rules below.

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

**A staking removal hook fires before the row is deleted, so it may not reconcile.** Every other
reactor path refreshes an infusion by reading the delegation and writing what it finds, and
`BeforeDelegationRemoved` is the one place that is wrong: `RemoveDelegation` calls the hook five
lines before its `store.Delete`, and the delegation still carries its *pre-decrement* shares, so a
reconcile there rewrites exactly the stale value it was called to clear. `ReactorInfusionDelegationRemoved`
therefore zeroes `Fuel` explicitly. It must also be the hook that does it, because staking gives no
other signal: `Unbond` routes a delegation whose shares reach zero through `RemoveDelegation` and
deliberately *skips* `AfterDelegationModified`. That is what made the leak invisible — a full
redelegation left the source infusion's fuel, power and grid capacity installed while `Delegate`
granted the destination capacity for the same stake, and the one compensating path,
`AfterUnbondingInitiated`, fires with a redelegation id that does not resolve to an unbonding
delegation. Note that the neighbouring case, a full *undelegate*, was always safe, because it hands
us a real unbonding id — the two differ only in which id staking passes, which is why one leaked and
one did not. `Defusing` stays untouched throughout: it tracks unbonding balances, which outlive the
delegation record, and is derived from staking rather than trusted from the row. **This was not
mainly an attack.** `GuildMembershipJoin` redelegates a player's entire infusion, so honest players
minted capacity every time they joined a guild on a different reactor;
`MigrateReconcileReactorInfusions` clears what they accumulated, and the grid cascade that follows
is the intended consequence rather than a bug in the migration.

Know the other half of that pair, because fixing the hook is what exposed it: **`SetDelegation`
fires nothing.** `AddressRevoke`, `AddressRegister` and `PlayerUpdatePrimaryAddress` all sweep a
player's stake onto their primary address with a `RemoveDelegation` followed by a `SetDelegation`,
and only the first of those is hooked — so the moment the removal hook started clearing the source,
the move stopped misattributing the player's capacity and started destroying it, the destination
being credited by nobody. All three now go through `MoveDelegationsToAddress`, which reconciles the
source *and* the destination explicitly rather than relying on either hook, which is also what lets
the mock exercise it. Two things about that function are load-bearing. The reconciles run after the
loop, not inside it: the hook commits a `CurrentContext` of its own, and `SetGridAttributeDelta`
reads through `cc`'s cache, so a reconcile interleaved with the hook leaves `cc` holding a capacity
read from before the hook's write and clobbers it at `CommitAll` — **a nested context that commits
is invisible to an outer context that has already read the same attribute.** And reconciling the
source is not redundant with the hook: it is what makes the repair independent of whether the hook
ran at all.

**There is no such thing as rekeying a delegation, so that sweep is an assembled transfer.** Cosmos
has no operation for handing a delegation to another account, and the pair of calls above was not
one. `SetDelegation` is keyed by `(delegator, validator)`, so a destination that already delegated
to the same validator had its record *replaced* while `Validator.DelegatorShares` went on counting
both — orphaned tokens and a skewed redemption ratio, unrecoverable once written, since nothing on
disk says which address lost the shares. Shares therefore merge, and `DelegatorShares` is
deliberately left alone: the sum is conserved and `LegacyDec` addition is exact. Firing no hooks was
the other half, and the one with no symptom at the time: the destination pair never received the
`DelegatorStartingInfo` that prices its rewards, and without it every later withdraw, undelegate and
redelegate fails on `ErrEmptyDelegationDistInfo` forever. `MoveDelegationsToAddress` drives that
lifecycle by hand through `x/distribution`'s hooks, in an order fixed by `initializeDelegation`,
which reads the delegation back out of staking (so `AfterDelegationModified` comes last) and prices
from `Period-1` (so something must have incremented the period first). **A store write that a
neighbouring call fires hooks for is not the same operation minus the hooks** — it is a different
operation that happens to leave similar-looking bytes.

Two states a delegation cannot be moved out of at all, and the limit is reachability, not caution:
an in-flight redelegation and an in-flight unbonding delegation each carry queue rows
(`RedelegationQueue` DVVTriplets, `UBDQueue` DVPairs) and id indices that no public keeper API can
rewrite, so the delegator address cannot follow the delegation. Leaving a redelegation behind is the
dangerous one — `SlashRedelegation` resolves the delegation it slashes through the redelegation
record's *own* delegator address and continues on a miss, so moving out from under one makes that
stake unslashable. What to do about a blocked delegation is a policy the caller passes, and the
difference is a security property rather than a preference: `AddressRegister` and
`PlayerUpdatePrimaryAddress` refuse the message, both addresses still belonging to the player, while
`AddressRevoke` never refuses. **A revoke is the response to a compromised key, so anything that key
can sustain must not be able to block it** — rolling redelegations otherwise let an attacker prevent
their own eviction, and refusing gains the player nothing, since that key could always have
undelegated the stake outright. The blocked delegation stays bonded and its infusion is destroyed
instead. `x/structs/keeper/delegation_transfer_test.go` covers the assembly against the mock,
`app/delegation_transfer_test.go` against real staking and distribution — including the SDK
accounting checks that `AllInvariants` used to provide before v0.53 retired the crisis module.

The reason all of this survived so long is worth its own line: **`testutil/keeper`'s
`MockStakingKeeper` fires no hooks at all.** Every keeper test that "exercises" a staking path is
really calling our own handler directly, which proves the handler and says nothing about whether
staking ever reaches it. Anything whose correctness depends on *when* the SDK calls us needs a
real-app test with real staking wired up — `app/delegation_transfer_test.go`,
`app/delegation_migration_test.go`, `app/reactor_redelegation_test.go`,
`app/address_revoke_infusion_test.go` and `app/reactor_jail_gate_test.go` are the five, all built
on `setupJailGateApp`. `x/structs/keeper/reactor_delegation_removed_test.go` covers the keeper
method, including a case that pins why reconciling in that window does not work.

**An infusion is owned by whoever owns its address, not by whoever owned it first.** `PlayerId` is
the only field on an infusion that can go stale — `DestinationId` and `Address` are the cache key,
`DestinationType` follows the id prefix — and `UpsertInfusion` used to write it on creation only.
An address changes hands (`AddressRevoke` clears the index, `AddressRegister` binds an unindexed
address to any player on a key proof) while the record, keyed by (destination, address), survives
intact, so the former player kept being credited the capacity and, through `GuildMembershipJoin`'s
`infusion.PlayerId` check, kept the authority to redelegate stake they could no longer reach.
`UpsertInfusion` now re-homes through `InfusionCache.SetPlayerId`, which withdraws the delegator
share from the outgoing owner and credits the incoming one. Re-homing rather than erroring is
deliberate: every caller is a staking hook that cannot refuse without desynchronising staking from
structs, and whoever controls the address controls the stake. **An ownership field consulted for
authorization needs a live resolution behind it**, so `GuildMembershipJoin` also resolves
`infusion.Address` through `cc.GetPlayerByAddress` — the record catching up eventually is no help
for a row carrying only a `Defusing` balance, which staking never touches again.
`MigrateInfusionOwnership` re-homes what the live bug wrote, walking `GetAllInfusion` because struct
generator infusions carry the same field.

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

**The checkpoint for a capacity change lives in the handler, so test it there.** Unlike a teardown,
which checkpoints inside `agreement_cache.go` and is held to it by
`TestArch_TeardownPathsCheckpointBeforeMutating`, `CapacityIncrease` and `CapacityDecrease` are
checkpointed by their message handlers. That put the ordering outside everything that was watching
it: the arch test only walks functions calling `beginTeardown()`, and every capacity test drove the
cache method with its own local checkpoint, so deleting `Checkpoint()` from either handler left the
suite green while the next checkpoint billed the new capacity across the span the old one served.
`TestCapacityChange_HandlerCheckpointsBeforeRaisingLoad` and its `LoweringLoad` sibling go through
the message server for exactly that reason. Note that only one direction is visible to the solvency
invariant — an increase overbills and leaves the pool short, while a decrease underbills and leaves
it over-funded with the revenue stranded — so both assert the amount swept rather than relying on
the invariant.

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

**A context getter is a cache allocator and says nothing about existence, so resolve
message-supplied ids with `cc.GetExistingPlayer`.** `cc.GetPlayer` does not read the store. It used
to return an error anyway, always nil, and twenty-eight callers guarded on it — a dead branch that
reads exactly like an existence check, which is how `AllocationTransfer` came to accept any string
as a controller. Nothing else covered it: `CanBeTransferBy` asks only whether the caller may
transfer. The error return is gone, so the compiler now rejects those guards, and an id a
transaction chose goes through `cc.GetExistingPlayer`, which loads and returns `ErrObjectNotFound`.
`cc.GetPlayer` stays correct for an id read off something already loaded from state — an owner, a
controller, a substation's connection list — because that id exists by construction.

What a phantom cache does is worse than a nil pointer, because it behaves: it loads as a zero value,
and since a failed load leaves `PlayerLoaded` false, every getter reloads and quietly reverts what
the handler wrote. What survives is whatever was written under a *different* key — a substation
index row and connection count keyed by the `PlayerId` field — plus a commit against `Player.Id`
`""`, which the KV store panics on. That panic is why the guild membership version of this rejected
the transaction and `AllocationTransfer` did not: its write is keyed by allocation id, so nothing
tripped, and it committed. **A refusal that only happens because an unrelated write happens to
panic is not a check.** `TestArch_HandlersResolveMessagePlayerIdsThroughGetExistingPlayer` is the
enforcement, and it follows a message field through a local variable, a loop over a repeated field
being one. Ante and permission-resolver reads are deliberately outside it: neither mutates, and a
phantom owns nothing and holds no permission row, so the following permission check refuses it —
which in the ante must decline the throttle reservation without rejecting the transaction.
`MigrateOrphanedAllocationControllers` re-homes what the live bug wrote.

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

**A synthesize-on-miss loader belongs to creation paths only, and the authorization
belongs outside the synthesis branch.** Guild membership is two-sided: a player files a request
and the guild approves it, or the guild issues an invite and the player accepts it. The stored
`GuildMembershipApplication` is the *only* evidence the first leg happened, so
`GetGuildMembershipApplicationCache` inventing a `proposed` one on a store miss handed every
approval path exactly the consent it was about to verify. `GuildMembershipRequestApprove` was a
force-join: `CanRequestMembership` takes no player argument and only asks whether the guild has
requests open, so any member of a recruiting guild named any victim and `ApproveRequest` overwrote
their `GuildId`, reset their `GuildRank` and moved their substation connection, with the victim
never transacting. Hence two loaders —
`GetOrCreateGuildMembershipApplicationCache` for the three handlers that open an application,
`GetPendingGuildMembershipApplicationCache` for the six that consume one — plus `requirePending()`
as the backstop on the mutators. `DirectJoin` and `Kick` are deliberately outside it, each
creating and consuming its own record and carrying its own authorization.
`TestArch_MembershipTransitionsRequirePendingApplication` holds the pairing and
`guild_membership_consent_test.go` is the regression suite.

Note the second half, which is the part that generalizes past this bug. A **get-or-create is two
code paths and a check written inside the create branch guards only new records.**
`GuildMembershipInvite` has no `Verify` call of its own, so the loader's `CanInviteMembers` was its
whole authorization, and it sat in the synthesis branch: an invite already on file was unguarded,
and an outsider could name one and carry on into `SetSubstationIdOverride`, which validates rights
on the destination substation and nothing about the guild, redirecting the invite to a substation
they own. Amending a record needs the authority that creating it needed. Also note why the fix
needed the handlers repaired to work at all: all six consumption handlers assigned the mutator's
error and dropped it, which was invisible while every mutator returned `nil`.

That authority is deliberately *equal* to the authority to create, not stricter, so at bypass level
`member` any member may retarget a colleague's pending invite — `CanInviteMembers` is the whole
boundary and there is no separate gate for amending. A guild that wants fewer amenders sets
`JoinInfusionMinimumBypassByInvite` to `permissioned`, which demands `PermGuildMembership` on the
guild object; `TestGuildMembershipInviteAmendmentFollowsInviteAuthority` pins both halves so the
intra-guild case is not re-reported as the outsider bug. Note the related gap that lever does not
close: a pending invite's destination can change under an invitee who is about to accept, and they
cannot pin it, because `substationId` on `MsgGuildMembershipInviteApprove` is a setter gated by
`CanManageConnectionsBy` rather than an assertion — naming the guild's own entry substation needs
connection rights there that a prospective member will not hold.

**A clawback token may only live where clawback is coherent.** Native guild tokens may be sent to
registered player addresses, the structs module, and indexed provider pools — nowhere else. This is
a bank send restriction rather than an ante check because ordinary bank sends and ICA message
execution bypass the structs ante, and it makes IBC escrow fail closed without tracking ibc-go's
address derivations. The restriction is destination-only, so a pre-upgrade balance at an unusual
address can always leave for a player. Provider pools are not safe-deposit boxes: earnings are
fully confiscatable, and collateral is protected only up to
`ProviderCollateralObligation`, the same helper the solvency invariant uses. Excess deposits remain
confiscatable. `ProviderPoolAddressKey` is derived state but load-bearing — write both rows beside
provider pool creation, clear them only after deletion drains both pools, rebuild them during
genesis import, and backfill them before any upgrade migration moves balances. Pre-upgrade IBC
escrow holding guild tokens is protect-only because it backs vouchers already in circulation; its
marker is derived too, and `ProtectLegacyGuildEscrowBalances` rebuilds it from exported channel and
bank state during both the upgrade and genesis import.
`x/structs/keeper/guild_bank_holder_test.go` and `app/guild_bank_send_restriction_test.go` cover the
policy and the full guild-denominated agreement lifecycle.

**Founding a guild has two identities, and every check belongs to the founder.** `GuildCreate`'s
signer is the *solver*: it is half of the work preimage and gets recorded as `charterSolverId`,
and that is all. The *founder* — `msg.FounderPlayerId`, or the solver when it is empty — owns the
guild, so the membership move, the entry substation's `CanManageConnectionsBy`, the reactor's
`CanCreateGuildBy` and the ownership all resolve against them. The two are the same player in the
ordinary case, which is exactly why getting this backwards would not show up in the common path.
A third-party founding is authorized by the founder's offline consent signature and nothing else,
so **that signature has to bind the whole shape of the guild** (`GuildCharterConsentInput`:
founder, reactor, entry substation, endpoint, anchor). It is a bearer token shared with a whole
mining pool — unbound, any holder could name their own substation as the guild's entry point,
which is every future member's power supply. It is collected *before* the grind, because a charter
is a race and a winning nonce that had to travel to the founder for a signature would lose to the
pool with the most attentive leader, not the most hashpower.

**The charter anchor is the replay protection, for the proof and the consent alike.** Both
preimages bind it, and founding a guild on a proof moves it to the current height, so every nonce
anyone was grinding dies at once — including the winning one. Block heights never repeat, so
nothing comes back. Do not add a counter: the obvious candidate, `GuildMembershipJoinProxy`'s
`proxyNonce`, is bumped by unrelated events and would expire a consent that has to survive weeks
of mining. The reactor entitlement path deliberately does *not* move the anchor, or a validator
collecting a perk would wipe every pool's work. Two things follow for anyone touching this:
`CharterAnchor` falls back to the current height rather than reading an unset key as zero (zero
would make the age the whole chain's history and sell a guild for one leading zero), and
`CharterAge` clamps rather than subtracting, since an anchor ahead of the height would wrap to a
maximally easy puzzle. Because moving the anchor is the *whole* of what retires a consent, only a
path that moves it may spend one: `GuildCreate` refuses a third-party founding on the entitlement
path outright, since that path leaves the anchor alone and would otherwise leave the signature live
until an unrelated player proof-founded — long enough to mine a fresh proof and replay it, dragging
the founder into a second guild at the entry rank. That refusal costs nothing, the entitlement path
already requiring reactor permission from the founder, who can therefore sign for themselves.
Anything added later that accepts a consent inherits the same either-or. The proof path also checks
that the named reactor's validator exists and is not jailed, matching `GuildUpdatePrimaryReactor`,
because `AppendGuild` writes `PrimaryReactorId` unconditionally and `GuildMembershipJoin`
redelegates every joiner's infusion to it; that check cannot strand a solution, since the work
preimage binds no reactor and a refused solver renames it and re-submits the same nonce.
`guildCharterDifficultyRange` and `guildCharterReactorAge` are params
because production values make a fresh dev chain unusable — see `config.yml`, and note that
`AppendReactor` stamps `guildCharterEligibleHeight` from the param at creation and never revisits
it, so changing the param does not re-age existing reactors. **A genesis validator's reactor is
created before this module has params**, which is the trap that made every fresh dev chain unable to
found its first guild: staking and genutil precede `structs` in `genesisModuleOrder`, so
`AfterValidatorCreated` reaches `AppendReactor` while `GetParams` still returns a zero `Params` and
`CharterReactorAge` substitutes the production month — the stamp on disk ignored the genesis file
while the params query went on reporting what the file asked for, which is why it reads as a
permission bug. `InitGenesis` restamps through `RestampReactorCharterEligibility` straight after
`SetParams`, before the import loop, so a restored chain keeps the clock it exported. The fallback
itself stays: a zero age would make every reactor eligible at once, which is the worse direction to
fail in. Anything else that derives state from a param inside a staking hook inherits this.
`x/structs/keeper/guild_charter_test.go`
is the suite; `app/guild_charter_reactor_test.go` is the one that proves real staking's bonded and
jailed states reach the free path's gate, since the mock returns whatever the test handed it, and
holds the genesis ordering with `TestGuildCharterGenesisReactorHonoursParams`.

**Guilds are property, so ownership and membership are separate and a handler must take a
`guildId`.** `GuildCache.SetOwner` moves `guild.Owner` and the `PermGuildAll` row and touches
neither player's `GuildId`, so every sale produces an owner who is not a member — and membership is
singular while ownership is not, a buyer being free to found or join elsewhere while holding what
they bought. This is deliberate and `TestCharterOwnedGuildNeedNotBeJoined` pins it, because the
obvious "fix" is worse: forcing a buyer into the guild would evict them from their own or make a
purchase impossible while they are in one. Authorization was never the gap —
`PermissionCheck`'s owner shortcut passes an owner on their own guild regardless of membership — but
a handler that resolves the guild from the *signer's* membership cannot be addressed to it at all,
which is how `GuildBankMint`, `GuildBankConfiscateAndBurn`, `GuildUpdateEntryRank` and
`PlayerUpdateGuildRank` came to be unusable by the person who owned the guild. All four now take an
optional `guildId` with membership as the fallback, and a supplied id needs `CheckGuild()` beside
it or it is the phantom-cache class above, `cc.GetGuild("")` being an allocator. Two rules had to be
restated rather than merely widened, and both times the new wording is identical for a member
caller: `PlayerUpdateGuildRank` asks whether the target is in the *named* guild instead of whether
it shares the caller's, and rank-derived authority — the entry-rank ceiling, and the
"outrank the target" fallback when `PermAdmin` fails — applies only to a caller who is a member of
the guild being edited, since a rank in some other guild is an unrelated number and a guildless
caller's zero would outrank everyone. `GuildBankRedeem` needs nothing, resolving the guild from the
token denom, which is what makes the token tradeable; the membership handlers already worked this
way. `x/structs/keeper/guild_non_member_owner_test.go` covers all of it, fallback included.

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
`observeReject` (see `app/ante/observability.go`).

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

**Combat is seeded from the block, so no test may name the survivors.** `StructCache.IsSuccessful`
draws evasion and each individual shot from the block `AppHash` plus an incrementing per-player
nonce, so the same attack sequence leaves different structs standing on different runs. Per-hit
damage in `struct_type.go` *is* fixed, which is what makes the outcome look predictable enough to
hardcode. A script that records one run's survivors and aims a later phase at them decays into a
phase that never runs: `test_chain.sh`'s AR3 defender matrix pointed at the EB-era structs, and by
AR3 most were rubble, so twenty-six `struct-defense-set` calls came back as permission denials — a
destroyed struct still loads, as a phantom owned by the empty string, so the permission check is
what refuses it — and thirteen attacks skipped without anything saying so. Resolve combatants at
use time through `ar_live` / `ar_defense_set` / `ar_defense_clear`, build a phase its own fresh
targets when it needs one to live, and report the count of attacks that actually executed, or lost
coverage is silent.

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
