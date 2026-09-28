# Substation upkeep: proposed consolidated design and implementation plan

**Superseded September 28, 2026:** The user selected a fixed 10% substation
operating loss for v0.22.0 and set aside this upkeep design. See the
[implemented rule and migration behavior](v0.22.0-substation-operating-loss.md).
The rest of this document is historical design research, not current release scope.

Updated September 25, 2026. **Start here for the current proposal.** This revision
incorporates the discussion and chooses solutions for the previously open design
questions. User-confirmed directions remain constraints; the concrete defaults and
policies selected below are recommendations made in response to the request to
resolve the remaining questions. They are not represented as previously approved.

Earlier reports retain their value as research, but their alternatives are not
cumulative requirements. This is a design document, not an implemented chain
feature or a certification of the wider v0.22.0 branch.

## 1. The design in plain language

A substation charges for **how many players it serves and how much power it exports**.
Disconnecting its incoming power does not make its upkeep cheaper. Upkeep restores
health and buys whole days of a temporary output benefit; health keeps declining
during those days. There is no manually managed incoming-power rating.

When someone adds more customers or outgoing power than the last purchase covered,
they fund the difference. Existing customers may leave and return without paying
twice. This requires two coverage totals, not billing histories for every player.

Ordinary player service is calibrated around **10 kW usable per player**. Large
generators are expected to support their additional demand with personal generation
or separately arranged supply. This is a pricing reference, not a limit on delivery
or a promise that every connected player receives 10 kW. Surplus power is allowed.

Owners maintain and develop their infrastructure; rivals can expose unsupported
outgoing commitments, claim collateral and force correction. No age-based scan or
scheduled shutdown enforces upkeep. Actual allocation changes and accepted audits
use bounded GridCascade processing.

## 2. Decisions carried forward and proposed resolutions

| Topic | Resolution in this draft | Basis |
| --- | --- | --- |
| Health | Keeps reducing output during prepaid upkeep. | User confirmed. |
| Age enforcement | No scheduler, due-height shutdown index or station scan. | User confirmed. |
| Pricing | Base + connection count + aggregate outgoing-power tariff. | User direction; parameters selected below. |
| Payment manipulation | Charge growth beyond purchased service coverage. | Recommended; arithmetic modeled. |
| Player demand | Use 10 kW for ordinary-service calibration; allow extra supply freely. | Recommended from current loadout calculations. |
| Funding | Whole purchases from an explicitly authorized payer; no contribution buckets. | Recommended simplification. |
| Automated output | Sole output, retained at zero, excludes temporary benefits. Growth also respects funded coverage. | Exclusivity confirmed; funding rule recommended. |
| Audits | Oldest ordinary allocation first; accepted suppliers may withdraw; self-audits allowed. | User confirmed. |
| Proof preparation | Allow work prepared within the current upkeep/audit cycle; require a real imbalance on submission. | Recommended explicit tradeoff. |
| Disrepair | Exhausted collateral requires authorized intervention; final reward includes dust. | User confirmed; amounts selected below. |
| Development | Repeatable upgrades with increasing cost and diminishing efficiency gains. | Recommended implementation of the long-game objective. |
| Migration | Mint collateral shortfall into live existing stations; use the same rules afterward. | Mint direction confirmed; eligibility and activation policy proposed below. |
| Release | Recommend a separate upgrade after v0.22.0; do not silently add this to its handler. | Scope recommendation, not a release decision already made by the user. |

## 3. Initial parameters

These are a coherent **playtest configuration**, not a claim of economic equilibrium.
The supplied testnet observations were bot-dominated and are not independently
revalidated by this work. Current-source loadout calculations establish technical
demand, not organic player income or willingness to pay.

| Parameter | Proposed initial value |
| --- | ---: |
| Ordinary player service reference | 10 kW usable |
| Fixed upkeep fee per purchased day | 100,000 ualpha / 0.10 Alpha |
| Per connected player per purchased day | 60,000 ualpha / 0.06 Alpha |
| Outgoing-power tariff, first 100 kW | 9,500 ualpha per kW per purchased day |
| Outgoing-power tariff, next 9,900 kW | 5,900 ualpha per kW per purchased day |
| Outgoing-power tariff, above 10 MW | 3,800 ualpha per kW per purchased day |
| Durable efficiency at development level zero | 70% |
| Efficiency during prepaid upkeep | 100% before applying health |
| Health curve | Quadratic; zero at 30 upkeep days |
| Wear buffer for outgoing commitments | 7 upkeep days |
| Maximum prepaid time | 7 upkeep days ahead |
| Blocks per upkeep day | 17,280 |
| Founding payment | 1 Alpha: 0.5 burned, 0.5 collateral |
| Restart reserve | 0.5 Alpha |
| Final-claim dust threshold | 0.01 Alpha |

One upkeep day is a **block unit**, approximately 24 hours at five-second blocks.
The resulting wear horizon is 518,400 blocks and the buffer/prepayment horizon is
120,960 blocks. Slower blocks lengthen the real-world duration. Earlier models used
15,500 blocks per day; do not mix their block constants with this configuration.

The 0.06 player fee is near the previous bulk-price reference of 0.057 Alpha for
10 kW sustained through seven days of wear. The smaller base reduces the penalty
on solo stations. For connection-only stations, one purchased day costs:

| Connected players | Whole station payment | Payment per player |
| ---: | ---: | ---: |
| 1 | 0.16 Alpha | 0.16 Alpha |
| 10 | 0.70 Alpha | 0.07 Alpha |
| 100 | 6.10 Alpha | 0.061 Alpha |
| 1,000 | 60.10 Alpha | 0.0601 Alpha |

These are **purchase prices**, not automatic daily debits. A station with substantial
headroom may repair less often. We deliberately do not charge excess incoming power
merely to force scarcity or hit an issuance-burn target.

## 4. Health, delivered power and safe commitments

Use elapsed blocks since the last successful upkeep, with nonnegative age `a`:

```text
H(a) = max(0, W² - a²) / W²                 W = 518,400 blocks
current output = accepted supply × H(a) × current efficiency
safe outgoing budget = accepted supply × H(a + B) × durable efficiency
                                                    B = 120,960 blocks
```

Round power down after exact integer arithmetic. Current efficiency is 100% while
prepaid time remains, otherwise the station's durable efficiency. There are no
floating-point calculations in consensus. Output never exceeds accepted supply.
Upkeep resets health and extends prepaid time by a positive whole number of days,
subject to the seven-day horizon. An addition to service coverage alone resets
neither health nor prepaid expiry.

**Player connections use the station's projected remaining output**, divided by
connection count after outgoing load, with subtraction clamped at zero. Cache this
projection once per CurrentContext and station. This is a local read, not a recursive
supply proof, a station scan or a transaction that destroys allocations. It makes a
connection-only station's local service reflect its health without inventing an
allocation audit bounty where there are no outgoing obligations to correct.

Existing outgoing allocations remain installed until a real mutation or successful
audit corrects them. Downstream stations may therefore retain credited input from a
neglected upstream station until that correction. This is the accepted consequence
of player-enforced maintenance; do not describe health reaching zero as a guaranteed
network-wide shutdown on that block.

New outgoing commitments must fit the durable safe budget. The temporary benefit
is for direct player headroom and cannot be exported through a new or enlarged
allocation. The sole automated allocation follows that same durable budget.

## 5. Exact payment and coverage rules

Let `N` be connected-player count, `O` be aggregate outgoing allocation power, and
`T(O)` the continuous piecewise-linear outgoing tariff from section 3:

```text
Q(N, O) = base fee + N × player fee + T(O)
```

Calculate `T` on total power, not individual allocation rows. Retain fractional
ualpha internally as an integer numerator and round the final payable amount up.
Splitting allocations must not obtain cheaper brackets or rounding discounts.

Store just `covered_connections` and `covered_outgoing_power` alongside upkeep
height and prepaid expiry. They say what service was paid for; they do not cap
incoming power or require a player to manage an operating-capacity rating.

### Adding service

For proposed totals `Nnew, Onew`, calculate the two increases independently:

```text
delta = max(0, Nnew - covered_connections) × player fee
      + max(0, T(Onew) - T(covered_outgoing_power))

charge = ceil(delta × max(one day, remaining prepaid blocks) / one day)
```

After payment, increase each covered total to the maximum of its old and proposed
value. A reduction in one component cannot subsidize an increase in the other.
The one-day minimum prices the benefit of the existing health reset even after the
temporary benefit expires. New service inherits the station's current age; it does
not receive another health reset for free.

Reducing actual service does not immediately erase coverage or produce a refund.
Returning within that coverage is free until an authorized smaller upkeep purchase
replaces it. A station can purchase extra connection slots for upcoming guild joins;
it is not necessary to process a separate payment for each player.

### Buying upkeep and deliberately downsizing

Default upkeep covers at least both current service and existing covered service.
A permissionless sponsor therefore cannot erase coverage someone else purchased.
An administrator may explicitly request a smaller coverage purchase, but neither
requested total may be below actual service after the transaction's changes.
Reducing coverage gives up the old larger entitlement; there is no refund.

For the selected totals, charge:

```text
Q(selected N, selected O) × newly purchased whole days
+ positive growth of each coverage component × remaining prepaid time
```

Then replace coverage with the selected totals, reset health and extend prepaid
expiry. The carried-time charge prevents buying six cheap days followed by one
large-service day and receiving seven large-service days at the cheap price.
Round rational terms consistently with the arithmetic model.

For automated output, the upkeep quote includes the intended automated power after
health restoration, as computed from the safe budget. An authorized operator may
choose a smaller funded output. There is no surprise automatic treasury debit.

### What this fixes, and its deliberate cost

With the proposed fees, 100 players and 60 MW of outgoing allocations cost
**255.46 Alpha per purchased day**. Shrinking output to 1 kW, buying upkeep and
restoring 60 MW costs the same total before rounding. Buying seven empty days and
immediately adding those services costs **1,788.22 Alpha**, the normal seven-day price.
Disconnecting only incoming power changes neither bill.

The extra coverage rule is necessary if coordinated detach/buy/reattach discounts
are to be closed. It is additional state, but only two totals. It also means an
expansion on a nearly worn-out station may be a poor purchase. Clients should offer
one combined expansion-and-upkeep quote; paying only for growth must not secretly
repair the station.

## 6. Payers, joins and automatic changes

**A payment always has an explicit payer and maximum spend.** A player can sponsor
from their own authorized vault without station control. A guild-vault payment
requires that vault's authority. In both cases the signing key needs the applicable
asset permission; station-management authority by itself cannot spend a treasury.
Paying grants no permissions, rank, ownership or right to connect.

Use one quote/payment helper from all service-growth paths. Provide an explicit
service-funding action for reserving connections or outgoing power without changing
health. Funded expansion and upkeep can also be composed with normal changes in one
transaction; any failure rolls back the bank and object changes together.

| Operation | Funding behavior |
| --- | --- |
| Connect, reconnect or migrate a player | Free inside coverage; otherwise require explicit funding of the difference. |
| Guild join, invitation acceptance or join-by-rank-purchase | Reuse available entry-station coverage. If connection is required and cannot complete, reject the entire transaction with a specific error; never silently leave the player unconnected or debit a treasury. |
| Ordinary output creation or dynamic increase | Check source authority, safe budget and funded coverage; explicitly fund any excess. |
| Provider/agreement capacity increase | Same coverage rule, plus existing checkpoint, collateral and settlement requirements. |
| Supply decrease or output reduction | No payment. Preserve existing cascade behavior and unused coverage. |
| Automatic output growth | Grow only to `min(safe budget, covered outgoing power)`; no bank call from a staking hook or block worker. |
| Automatic output reduction | Reduce normally, including to zero; retain the exclusive allocation object. |

An automated output waiting for funding is a visible condition, not a failed hook.
An explicit funding/upkeep action retriggers its resize. Supply recovery up to
already funded coverage remains automatic. Only authorized station changes can
increase outgoing commitments; funding them does not bypass that authority.

Whole purchases are the default. Players can transfer contributions to their guild
through existing wallet operations; no on-chain waterfall bucket is introduced.
Rank purchases pay the guild's token, so the guild still needs actual ualpha to fund
upkeep. The interface must not imply that rank revenue automatically provides it.

## 7. Accepted input and avoiding unwanted obligations

Keep open allocation contributions, but require station authority to accept a new
outside input. Acceptance belongs to the allocation identity at that destination.
Moving it away clears acceptance; moving it back requires approval again. New rows
cannot inherit a deleted row's acceptance. An accepted row's later power changes
follow its normal allocation rules, including hostile withdrawal.

Unaccepted input is stored and indexed but contributes no usable capacity. Accepted
supply and outgoing load are cached aggregates updated with the corresponding
allocation writes; pricing and readiness must not scan the graph. Pending/deleted
station checks apply to acceptance, connection and new commitments.

Increasing accepted supply alone costs nothing under service pricing. More output
for the same customers is allowed. Exporting that increase still goes through the
outgoing coverage rule. Keeping one exclusive automated output, including at zero,
must be enforced by object/index existence in both creation directions.

## 8. Audits, proof preparation and settlement

### Eligibility and proof

An audit requires installed outgoing load greater than current projected output.
Connected-player count is a billing quantity, not a power obligation added to that
load. Zero-health stations with positive outgoing load remain auditable. At zero
output use the maximum-severity branch rather than divide by zero.

Use a canonical, length-delimited hash preimage containing chain ID, an audit domain
tag, station ID, last-upkeep height, upkeep revision, audit nonce, beneficiary player
ID and worker nonce. Increment the upkeep revision on every upkeep, including two
in the same block. Consume the audit nonce when a proof is accepted, before queuing
work. Emit/query every public challenge field; bind payment to the beneficiary so a
copied proof cannot redirect the reward.

**Prepared work within a cycle is allowed.** Including the previous upkeep does not
make it unpredictable. A prepared proof must still pass live eligibility and required
difficulty; another upkeep or accepted audit invalidates the old cycle. Do not add a
randomness service or pretend the race starts only when an imbalance appears.

Retain the existing game difficulty function and a bounded integer severity table.
The one-hour severe / one-day mild targets are tuning targets, not settlement SLAs.
The old simulator's block constants and unprepared-racer assumptions do not validate
them under this draft; recalibrate with prepared competitors before launch.

Maintain a single accounting checkpoint for changes to accepted supply, outgoing
load or the health/efficiency schedule. Before changing it, derive whether and when
the old schedule became imbalanced. A bounded search over block heights can find the
first crossing without logs or per-block work. Carry forward an existing imbalance
start if the change leaves it imbalanced; set it to the current height for a newly
created imbalance, and clear it when corrected. Harmless edits must not continually
restart difficulty. Temporary-benefit expiry is part of that projection.

### Snapshot the reward once

At acceptance capture outgoing load `L`, safe budget `S`, unreserved collateral `C`
and the **full achieved** leading-zero difficulty `d`. The existing checker that
stops counting after the required difficulty is insufficient for a reward scale.

```text
correction fraction = (L - min(L, S)) / L
scale(d) = 25% for d <= 5; +18.75 percentage points per zero; 100% at d >= 9
reward = min(C, max(1 ualpha, floor(C × 50% × correction fraction × scale(d))))
```

If the remaining reserve would be at most 0.01 Alpha, include it in this final reward.
Use the required correction fraction, not the size of a large indivisible allocation
that happens to be oldest. Thus coarse allocation sizing does not inflate the bounty.
This is an intentional refinement of the earlier actual-power-shed proposal.

Reserve this amount at acceptance. Later deposits, supply movement or allocation
overshoot cannot increase that claim. Reward reservation, nonce consumption and the
pending settlement record are atomic. Only one accepted audit may be pending for a
station. The claimant has earned one fixed claim; completion pays it exactly once.

### Bounded correction, including failures

During pending settlement, reject new station commitments, connections, upkeep,
development, ownership transfer and deletion. Allow reductions, collateral deposits
and unavoidable upstream changes; those cannot revoke the accepted claim. No hook
may reject an upstream withdrawal because the destination is being audited.

Shed ordinary outputs by numeric `Allocation.Index`, oldest first, until they fit
the safe target. Resize the sole automated output instead of deleting it. Recompute
connection capacity after correction. The target may tighten as supply/health falls
while processing; a subsequent improvement must not reopen growth during settlement.
If the reserved reward exhausts available collateral, the target is zero and the
station is already in disrepair: drain ordinary outputs, retain automated output at
zero and expose zero connection capacity.

Each work item must be bounded, resumable and use the existing allocation/agreement
teardown machinery. Checkpoint providers before changing load, pay consumers before
provider revenue, and never remove an allocation while its agreement settlement
failed. Roll back a failed item through the supported store/cache transaction
boundary and do not retain stale outer-cache state or a claimed teardown flag.
Do not revisit the failed entry within the same block's worker allowance.

A failed item keeps its cursor and reserved reward for retry. Surface a specific
error/event; do not silently forgive debt, pay twice or delete the evidence needed to
finish. Successful completion removes the transient settlement record and pays the
bound beneficiary's valid vault atomically. Validate that destination at acceptance
and use a stable recipient identity. A bank failure retains the pending claim.
This is operational settlement state, not an accumulating financial audit ledger.

The completed target must fit the **fresh** safe budget, giving the seven-day wear
buffer if supply is unchanged. A second transaction cannot collect again against
the same unchanged imbalance. New supply loss can create a new opportunity, as the
user intends. Bounded downstream propagation may finish later; do not emit a claim
that the entire network is already repaired.

## 9. Collateral, creation and recovery

Creation burns 0.5 Alpha, deposits 0.5 Alpha into the station's collateral account
and buys at least one upkeep day for its initial covered service. An empty new
station therefore costs **1.10 Alpha** before any paid service expansion. Validate
and commit creation and payments atomically. A prefunded predictable collateral
address is a donation, never grounds for refusing creation.

Collateral stays with the station and has no ordinary withdrawal operation.
Voluntary deletion burns its unreserved remainder after obligation settlement;
pending audit claims must finish first. This prevents deleting and recreating a
station to reclaim its reserve. Emit the account-to-station association and real
asset movements using existing conventions.

A nonzero reserve is normally sufficient. Exhaustion through a reserved claim sets
an explicit disrepair flag. A donation alone does not clear it. To recover, a station
authority must bring the free reserve to at least **0.5 Alpha** and fund at least one
whole upkeep day. That upkeep restores full health; no additional arbitrary recovery
fee or health-floor formula is needed. A sponsor may fund it, but cannot force the
authorized recovery transition. Pending teardown must complete before recovery.

Zero projected health without collateral exhaustion is ordinary neglect: a sponsor
can restore it through upkeep unless an audit is already pending. This avoids two
competing notions of permanent disrepair. No zero-health scheduler is introduced.
Collateral never pays upkeep automatically, and upkeep is 100% burned.

## 10. Lasting development without a final upgrade tier

Recommend one repeatable development track that improves durable efficiency with
diminishing returns. For completed level `l`:

```text
durable efficiency(l) = 70% + 15% × l / (l + 10)
next level burn        = (l + 1) Alpha
```

| Level | Durable efficiency | Total development burn |
| ---: | ---: | ---: |
| 0 | 70% | 0 Alpha |
| 1 | about 71.36% | 1 Alpha |
| 10 | 77.5% | 55 Alpha |
| 100 | about 83.64% | 5,050 Alpha |

There is no designed final level. The efficiency approaches 85%, preserving loss
per hop and a temporary-upkeep benefit. Returns eventually become small: this is
ongoing optional investment, not a promise that every distant level is equally
valuable. Health still reaches zero after 30 upkeep days at every level.

Compute the rational efficiency directly when calculating power; do not first round
it into a small fixed set of basis-point tiers. Use checked/bounded integer inputs
and reject numeric overflow. Development costs and benefits belong to the station
and survive ownership transfer. Upgrading neither resets health nor extends prepaid
time. It changes the current/safe projection; any automatic export growth still
needs funded outgoing coverage. Only an authorized operator can initiate development,
with separate authority over the payer's funds.

This curve is a new proposed solution, not part of the earlier 80-test evidence.
Test its effects on consolidation, paid-benefit demand and high-level rounding before
shipping. It does not turn a station into an energy generator or solve graph loops
by itself; retain the grid's accounting invariants and cycle regression scenarios.

## 11. Migration and release placement

Recommend shipping this as a **separate subsequent upgrade** because it adds a new
lifecycle, payments, proof settlement and grid policy. Keep the v0.22.0 guild/vault/
reactor/security work independent. If release scope keeps this in v0.22.0, the same
acceptance gates apply; this document itself does not change release wiring.

For the chosen upgrade:

1. Enumerate existing stations in the migration, where a full scan is appropriate.
   Closing stations continue their existing teardown; do not mint a new reward for
   an object already being deleted.
2. Create/associate collateral accounts for live stations. Preserve any existing
   balance and mint only `max(0, 500,000 ualpha - existing eligible collateral)`.
   Publish the exact total issuance and per-station funding events. Handle account
   types and prefunding without reintroducing predictable-address denial of service.
3. Accept currently connected, valid input allocations so existing supply does not
   disappear merely because it predates the acceptance rule. Future inputs use normal
   approval. Rebuild/verify the corresponding aggregates and scoped indices.
4. Set full health, development level zero and service coverage to the existing
   connected count and outgoing load. Give no temporary paid benefit. Existing
   coverage avoids charging again merely to restore a previously covered connection;
   future larger service still costs the difference.
5. Apply the standard 70% durable projection and new admission rules. Recalculate
   shared player capacity. Do not silently delete installed outgoing commitments in
   the migration; normal real mutations and player audits handle their correction.
   Existing commitments above projected output are immediately auditable.
6. Emit initialization, collateral/address associations, coverage and resulting
   capacity fields sufficient to reconstruct the transition. Verify export/import
   and zero-height export semantics for the new clocks and transient work.

**This proposed migration accepts an initial reduction in local usable power.**
Full health and minted collateral do not preserve the old 100% output. Avoid hiding
that behind a nominal seven-day gift: already oversized commitments could still be
auditable as soon as health decays. Rehearse and publish the affected stations,
players, agreements and total mint before release. This is a new migration policy
recommendation; the mint direction alone was the user's earlier confirmed decision.

## 12. Implementation order and acceptance gates

1. **Lock down pure arithmetic.** Add the proposed parameter set, exact quote and
   renewal rounding, development curve, health/difficulty checkpoint transitions,
   reward snapshot and dust rules to the research tests. Retain the adversarial
   detach/repair/reattach, dynamic restore, carried-prepayment, station rotation,
   self-audit, prepared-work and supply-withdrawal cases. Recheck affordability and
   audit timing with the new constants; do not rerun unrelated sweeps by default.
2. **Add state and pure projections.** Define protobuf fields/messages/events, then
   regenerate. Add service coverage, accepted-input aggregates, current challenge
   state and at most one pending audit record per station. No per-block histories.
   Keep CurrentContext reads cached and commits deterministic; prove nested writes
   cannot be overwritten by an already-loaded outer cache.
3. **Integrate every funded growth path.** Share the coverage helper across station
   connection/migration, guild entry, allocation creation/update, provider changes
   and automatic resizing. Keep object authority separate from payer authority.
   Test delegated keys, guild rank permissions, failed payments and atomic rollback.
4. **Implement audits and recovery.** Reuse teardown and bounded GridCascade. Test
   payout failure, retry after nested rollback, oldest-index order, automated zero,
   transient disrepair, deposits during settlement, reward replay and blocked recovery.
5. **Bound and measure production work.** Include the new queue in a shared work
   allowance, with bounded per-item cost and progress for other queues despite one
   failure. Select the allowance from real measurements on the supported node setup,
   not the toy model's 256-operation default or an assumed gas-to-time conversion.
   Measure readiness in real combat, large connection sets, dust input graphs and
   multi-block cascades under the 20M transaction ceiling.
6. **Rehearse migration and public behavior.** Use a current testnet export where
   available, real-app staking/grid/agreement tests, mint/bank invariants and
   deterministic replay. Update `test_chain.sh` in detail for the full lifecycle,
   events, failed transactions and restart/export paths. Review event contents with
   indexer needs in mind; indexer implementation remains the user's separate work.

Every stage needs explicit tests for player-triggered enforcement: simply advancing
blocks must not enqueue age-based teardown. Read projections must not mutate state.
Queries must distinguish projected output, installed outgoing commitments, funded
service coverage, unreserved collateral and pending correction.

## 13. What remains uncertain after these choices

The policy questions above now have proposed answers. The remaining work is evidence:

- **Affordability and adoption:** organic income, willingness to sponsor, repair
  cadence, concentration and prepared-auditor participation need playtesting. The
  sink is voluntary/strategic, so no model establishes a fixed share of issuance.
- **Integration and liveness:** the full feature has not been implemented or exercised
  through SDK transactions, hooks, retries and bank failures.
- **Production work bounds:** neither gas nor validator block-time limits can be
  certified from a Python economic model.
- **Migration impact and release timing:** quantify the 70% transition and complete
  the same acceptance gates whichever release contains it.

Verification for this revision: reran the research suite with
`python3 -m unittest discover -s tools/substation_model -p 'test_*.py' -q`;
**all 80 tests passed**. Separately checked the new fees against the coverage
calculator, including shrink/restore and seven-day empty-purchase expansion,
and verified the block constants, development examples and document links.
These checks cover prior models and selected arithmetic, not this entire proposed
specification. Permission rules, progression behavior, migration, settlement and
the complete new parameter configuration still require the validation above.

## Evidence and superseded alternatives

- [Player calibration and coverage arithmetic](substation-player-calibration-and-payment-guards-2026-09-25.md)
- [Connection-count/outgoing-capacity model](substation-service-pricing-model-2026-09-25.md)
- [Discount-delay model](substation-discount-delay-model-2026-09-25.md)
- [Fixed repair alternative](substation-fixed-repair-model-2026-09-25.md)
- [Operating-rating alternative](substation-operating-rating-model-2026-09-25.md)
- [Broader adversarial scenarios](substation-adversarial-model-2026-09-25.md)
- [Supplied-report review and user clarifications](substation-upkeep-audit-review-2026-09-25.md)

Manual incoming-capacity ratings, delayed discounts as the only protection, forced
scarcity, scheduled age-based shutdowns and historical billing logs are not part of
this proposal. Historical reports should not be read as adding them back.
