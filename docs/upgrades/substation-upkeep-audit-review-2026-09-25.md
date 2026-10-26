# Review: Substation Upkeep and Audit, revision 2

September 25, 2026. Review of the supplied **Substation Upkeep and Audit Report**
and its **Economy Equilibrium** companion. This is feedback, not implementation
approval or a replacement formula specification.

**Decision received during this review:** the user chose a one-time migration mint
of ualpha to fund each existing substation's minimum collateral. This supersedes
both the supplied report's zero-collateral migration and the exemption alternative
raised during review. Exact reserve sizing remains to be specified.

**Second clarification received:** this mechanic concerns substations, not reactors.
The user confirmed that substation output continues to decline with health even
during the paid period. Retain `accepted supply × health` during that period and
`70% × accepted supply × health` otherwise. “Prepaid upkeep” is the recommended
replacement for the confusing word “fuel”; the terminology is a recommendation,
while continued health-based decline is now confirmed.

**Further clarification:** a substation's automated output is exclusive: it is the
only outgoing allocation. This resolves the shedding-order concern. The user also
agreed that cheap prepayment followed by supply expansion must be addressed, and
suggested binding audit work to the station and its last-upkeep block height.

**Enforcement clarification:** the user rejected scheduled health-zero shutdown.
Collateral rewards and hostile players motivate audits. Time changes projected
condition and eligibility; it does not schedule allocation teardown. Reuse bounded
GridCascade after an audit or another actual allocation/supply mutation.

## Overall assessment

This is a stronger gameplay proposal than the previous maintenance draft. Fuel
gives players a concrete purchase, accepted supply makes operating costs something
owners can control, and allocating only durable output separates reliable contracts
from temporary power for connected players. Burning upkeep in full also makes the
economic effect much easier to explain.

I support developing this as a separate upgrade after v0.22.0. I would not yet call
it internally consistent, simulated end to end, or ready for implementation. Several
rules conflict, and the simulation expressly omits the multi-station and adversarial
cases that determine whether the mechanism works on this chain.

The next step should be a short, authoritative rule specification and a reproducible
simulation revision, not additional economic features.

## What is particularly promising

- **Players can rescue shared infrastructure.** Permissionless payment creates a
  useful role for members when an owner is absent, without granting the payer control.
- **Temporary power has a distinct purpose.** Fuel benefits connected players while
  contracts are admitted against durable output. This is a clearer product than
  selling a temporary boost and letting it back long-lived contracts.
- **Risk has a consent boundary.** Accepting outside supply makes withdrawal traps
  a deliberate infrastructure risk. Their destructive consequences remain accepted
  gameplay; this review does not recommend removing them.
- **Infrastructure scale becomes a strategic choice.** Lower operating costs for
  larger stations compete with the resilience of smaller ones. Oldest-first shedding
  gives that risk an understandable order.
- **The report admits when the sink does not work.** Separating abundant-energy and
  scarce-energy scenarios is much more useful than presenting one headline burn rate.
- **The release recommendation is sensible.** Keeping this out of an already large
  migration provides room to test the actual network behavior.

## Rules that need resolution before implementation

### 1. Fund legacy collateral during migration; still assess the initial output cut

The supplied migration gave legacy stations full health and zero collateral while
ordinary rules shut down a station with zero collateral. **Resolved by the user's
subsequent decision:** mint the minimum necessary ualpha during the upgrade and fund
each existing station's collateral. Use the same operating/disrepair rules afterward;
do not introduce a permanent legacy exception or unpaid-audit mode.

The proposed migration also imposes an immediate change from existing throughput to
70% unfuelled throughput. The example station can absorb that because it is heavily
overprovisioned. A fully committed legacy station cannot. Marking all existing inbound
allocations unaccepted would be an even larger immediate cut.

**Recommendation:** implement the chosen migration independently of the two large
testnet examples:

- Use the normal minimum restart reserve as the proposed funding target. It must
  exceed the final-claim dust threshold; its exact amount is still pending. Treat
  this as collateral, not a purchase of fuel or a founding burn.
- Mint only the shortfall to that target if an existing station account already has
  usable ualpha. Predictable-address donations must not block migration. Protect
  reserved funds if any exist; minting must not make them available twice.
- Derive the finite total from existing eligible station records at the upgrade
  height, in deterministic order. Specify how pending-deletion stations are treated;
  recommended default is to finish deletion rather than fund a station already
  closing. Rehearse that rule against actual exported state.
- Emit truthful mint/funding and collateral-address association events, reconcile
  the total bank-supply increase, and include these coins in subsequent economy
  measurements. Execute the upgrade atomically and verify rollback/replay cannot
  double-mint. This is one-time bootstrap issuance, not recurring bounty issuance.
- Backfill existing connected inputs as accepted if preserving their supply is the
  intention. Do not silently accept unrelated future inputs.
- Measure the day-zero 30% cut, existing commitments above the safe budget, and
  downstream impact. “No immediate total shutdown” is a more accurate claim than
  “no cliff” unless a separate transition actually preserves existing output.

### 2. Retain health-based decline and correct the full-output promise

The formula says fuelled output is `supply × health`, and health decays continuously.
After one day, health is approximately 99.889%; after seven days, 94.556%. Output
therefore changes inside a fuel-day and can cross a player's readiness threshold
before fuel expires. The fuel switch is discrete; the full projection is not
piecewise constant.

**Resolved by the user:** keep the formula and its continuous decline. Do not freeze
health effects or make output equal to supply throughout the paid period. This is
about substations only and makes no change to reactor fuel or reactor output.

Describe the purchase as **prepaid upkeep** that removes the 30% baseline penalty
for a period while health still applies. For example, at 95% health the station
delivers 95% of accepted supply with active upkeep and 66.5% without it. Remove
“full output for a whole day,” “never fades within a day,” and “a struct stays online
until fuel runs out” from the report and client promises. Update the diagram and
projection description accordingly. Supply withdrawal, new outgoing commitments
or additional connected players can also reduce an individual's share.

An end-of-paid-period capacity forecast would be more useful than an uptime promise:
show whether the player's current load fits through that period under unchanged
supply, commitments and connection count. It needs no extra on-chain history.

### 3. Prepaid fuel needs a rule for changing accepted supply

The user confirmed this concern must be addressed. The pricing mechanism below
remains a recommendation rather than a selected implementation.

A fuelled-until height says how long the station is paid for, but not how much supply
was paid for. As written, an owner could buy seven days while accepting very little
power, then accept substantially more power and receive its fuel benefit at the old
price. Temporarily reducing accepted power before each purchase creates the same
problem. Resetting health on such a cheap purchase adds another benefit.

Acceptance alone also does not bound later increases of an accepted dynamic or
automated input. An accepted flag could become permission to raise tomorrow's bill
by an arbitrary amount.

**Recommendation:** use an explicit accepted-power ceiling and a paid fuel capacity
alongside the expiry height. Extra supply receives only durable treatment until its
remaining fuel interval is paid for. A decrease does not produce a refund that can
be recycled through a decrease/increase loop. State exactly how health restoration
is priced when accepted capacity changes; the cheap-purchase/full-health loop must
be covered too. Compare this design with prepaid power-time credits in the simulator
and choose one, rather than leaving price fixed at purchase and benefit unlimited.

Acceptance should be scoped to a destination and allocation identity. Reconnection
to another station must not carry consent with it. Define what ownership/source
changes do to that consent, and show the maximum bill the permission holder accepts.

### 4. “Anyone can fuel” and whole-station, whole-day purchases are different economic promises

At the recommended brackets, one day for the 29.6 MW station costs **133.84 Alpha**.
The quoted **0.043 Alpha per member** is a division of that bill, not a purchase the
described transaction supports. Most players in the supplied holdings data cannot
buy even one day for their shared station.

The user raised guild-funded upkeep as the simpler alternative to pooled contributions
or individually prorated payments. **Updated recommendation, not yet a final decision:**
retain whole upkeep purchases, with the guild treasury as the normal payer for guild
infrastructure. Any player can still sponsor a complete purchase. Guild-vault spending
requires its existing authority and the signing key's asset permission; this does
not give outsiders permission to spend guild funds.

Members can send ordinary Alpha contributions to the guild vault, and an authorized
operator makes the upkeep purchase. No contribution counter, automatic funding
threshold, refund mechanism or per-contribution health reset is needed. Receiving a
vault deposit does not itself trigger upkeep. Show the total payment in the interface;
the per-member amount is an economic average, not an available purchase size.

Important currency distinction: current rank purchases pay the guild's **own token**
into its vault (`guild_rank_purchase.go` uses `guild.GetBankDenom()`), while station
upkeep burns **ualpha**. Rank revenue is not directly an Alpha upkeep budget. Guilds
need liquid Alpha contributions or another explicit funding/conversion route; do not
silently spend token-backing collateral or add automatic conversion to this feature.

Concrete example: a station costs 134 Alpha for one day and has roughly 3,116 members.
Dividing produces about 0.043 Alpha per person. But a whole-day-only transaction
still asks its caller to pay 134 Alpha. Sending 0.043 Alpha does not buy one day
unless the protocol or an existing guild mechanism collects the other contributions.
“Anyone is permitted to pay” does not mean “anyone can pay just their share.”

Pooled threshold payments and smaller proportional purchases remain possible future
alternatives, but neither is recommended for the initial implementation. Both add
rules around incomplete funding, changing prices/capacity and health restoration.
The guild-payer approach is simpler and still permits social cost-sharing through
ordinary vault transfers; it assumes a funded treasury and an available operator.

### 5. Proof binding does not by itself prevent mining before the imbalance

The station ID, next audit nonce and prior settlement anchor may already be known
while a station is healthy. Predictable wear makes the future imbalance predictable
too. A supplier can also prepare work before deliberately withdrawing supply.
Binding a proof to these fields prevents some replay and redirection; it does not
prove the work began after the problem arose.

The user's proposed last-upkeep height is a reasonable simple cycle anchor, in line
with the existing work mechanics. Recommended canonical proof fields are chain ID,
`AUDIT` tag, substation ID, last-upkeep height, upkeep revision, audit nonce, beneficiary
player ID and the worker's nonce. Increment the upkeep revision for every successful
upkeep, including multiple payments in one block, and advance the audit nonce after
successful reconciliation so proofs cannot be reused even when no further upkeep
has occurred.
Use unambiguous encoding. Publish the cycle fields in events and a query so workers
can recover them without depending on one indexer's history.

This provides fresh work after upkeep; it does not prevent workers from mining
while the station is still healthy during that same upkeep cycle. An upkeep
transaction ID has the same limitation, and its sender knows it before inclusion.
A prepared proof still must satisfy the current imbalance and required difficulty
at submission, but its solver has a head start over someone who begins then.
Recommendation: accept and model that head start if the simple anchor is sufficient
for gameplay. Do not claim the full one-hour/day race starts only at imbalance.
The decision to permit that behavior remains distinct from choosing the anchor.

Choose explicitly whether prepared proofs are part of gameplay. If they are, remove
the anti-premine claim and simulate prepared auditors. If the race must begin after
imbalance, the challenge needs an anchor unavailable beforehand, with a bounded
way to establish and retain it. Do not assume the chain can retrieve arbitrary old
block hashes on demand. Merely adding a predictable onset height is insufficient.

The onset rule also needs definition across supply withdrawal, upkeep, changing
commitments and a changing imbalance ratio. Projection can calculate a crossing
inside one unchanged interval; it cannot reconstruct arbitrary past supply changes
from today's values. Avoid letting a harmless state mutation restart the race clock
while the station remains imbalanced.

**Implementation finding:** the current
[HashBuildAndCheckDifficulty](/Users/abstract/Documents/Structs/Code/structs/x/structs/types/work.go)
counts zeroes only up to the required difficulty and returns that count. It does not
report all leading zeroes actually achieved. A reward based on achieved difficulty
needs a separate complete, bounded count. Decide whether rewards use actual work or
the difficulty required at submission; the simulator and chain must use the same rule.

### 6. Enforcement remains player-triggered; no shutdown schedule

**Clarification:** GridCascade already supplies the downstream processing once an
allocation is resized or destroyed. `SetPower` updates destination capacity and
queues a reduction; `Destroy` removes its contribution and queues the destination.
Reuse those paths and the existing bounded cascade. No replacement propagation
system is proposed here.

**Resolved by the user:** no due-height shutdown index, scheduled health checks or
automatic teardown when projected health reaches zero. The incentive to act is the
collateral reward and the strategic value of disrupting another team's infrastructure.
My previous recommendation to schedule terminal shutdown is withdrawn.

The lifecycle is:

1. Elapsed blocks change projected health, throughput and audit eligibility. Queries
   and admission checks use that projection without deleting allocations or emitting
   a fictitious completed shutdown event.
2. An auditor submits valid work against the current imbalance. Zero projected
   health must not itself block that audit while unsupported commitments remain;
   otherwise the most neglected station would escape the enforcement mechanism.
   Define zero-throughput imbalance arithmetic without division by zero.
3. Accepted reconciliation closes oldest ordinary allocations or resizes the sole
   automated output, recalculates shared capacity and handles the reserved reward.
   Existing GridCascade propagates those actual changes in bounded work.

An owner can also correct the station through authorized changes. Actual supply
withdrawal and other existing capacity mutations continue to trigger their normal
cascade; this decision removes only time-triggered upkeep enforcement. Workers may
finish an already accepted audit across blocks; that is not a periodic upkeep scan.

If nobody acts, outgoing commitments can remain installed despite projected health
being zero. That is accepted player-enforced maintenance, not a missing background
job. The report must therefore say health reaches zero at approximately 30 days,
not guarantee all downstream service has shut down by that instant. Local projected
capacity and installed outgoing commitments are distinct, and must be labeled as
such in queries and events. Exhausting collateral through an actual claim remains
an explicit disrepair transition. Do not add recursive full-graph readiness checks
or let a read-only health projection silently perform audit settlement.

Test time passage with no transactions, zero-health audits, and subsequent bounded
cascade completion. No scheduled cleanup should substitute for a player action.

### 7. Exclusive automated output resolves the ordering concern

The user confirmed that a station with an automated output has no other outgoing
allocations. There is no competing allocation to prioritize: resize that sole output;
for stations with ordinary allocations, shed them oldest-first. Preserving the
automated routing object does not create an ordering conflict under this invariant.

Keep these implementation requirements:

- Resize automated power to the **durable safe budget**, excluding temporary upkeep
  benefits. Exclusivity resolves ordering but does not make temporary power durable.
- At zero budget, retain the automated object with zero delivered power and zero
  positive load, as required by the proposed preserve-rather-than-delete behavior.
- Enforce exclusivity in both directions: no automated output alongside existing
  outgoing allocations, and no ordinary output while an automated one exists.
  Check object/index existence, not just nonzero load. The current creation code
  explicitly checks conflicts when creating an automated output, while ordinary
  allocation admission relies on headroom. The new safe budget can leave headroom,
  and zero-power automated outputs also must not permit accidental coexistence.
- Include automated power reductions in the bounty's power-shed calculation if
  auditing them is meant to pay. Test this separately from allocation deletion.

No new output-priority system or mixed automated/fixed mode is needed.

### 8. Final settlement must not inherit mutable reward inputs

Audit acceptance and reconciliation can occur in different blocks. During that time,
supply, collateral deposits and allocations can change. Define when collateral,
achieved difficulty and the denominator of the shed fraction are fixed. Record the
accepted claim's bounded current state; never reserve against one balance and pay
against a larger later balance.

For the simplest reward calculation, price the required reduction against the
acceptance snapshot rather than the eventual overshoot from deleting a large oldest
allocation. If rewards intentionally pay for actual power removed, cap the reserve
at acceptance and settle the final amount within it. Include automated reductions
and explain whether shedding all the way to the seven-day safe budget earns more
than merely correcting the present deficit.

Preserve the dust sweep, exactly-once credit and the rule that unpaid bounty delivery
does not create audit immunity. The diagram should also distinguish permissionless
fuel from authorized recovery: an offline station cannot both require authorized
repair and automatically restart whenever a stranger fuels it.

## Economic interpretation and useful tweaks

### Separate an economic model from a forecast

The simulator files referenced in the report were not found in the searched workspace.
I reviewed the supplied results and reproduced small arithmetic checks, but did not
rerun the sweeps or verify the SQL extracts. The reported passing parameter sets are
evidence under the model's assumptions, not independent validation of the chain.

Keep the simulator, exact parameter file, seeds and anonymized input extracts in a
reviewable location. Publish tables generated from that one configuration. Multiple
old and new bracket configurations currently appear together.

The claim that hashers win every simulated race needs stochastic testing and
confidence intervals, not just a representative solve time. For independent uniform
hashes, difficulty 9 takes an expected `16^9` attempts: about **2.1–9.5 hours** at
the report's 9–2 MH/s rates. Difficulty 5 takes about **0.12–0.52 seconds**. The
fourfold reward increase spans a 65,536-fold difference in expected hash attempts.
This does not invalidate early races, because required difficulty falls over time,
but it does mean participation and profitability are not established by the payout
multiplier alone. Test owners and external auditors choosing when to start, stop,
wait or precompute, including the opportunity cost of refining instead.

### Correct a few quantitative claims

Under the recommended **100 kW / 10 MW** marginal brackets:

| Check | Recomputed result |
| --- | ---: |
| One fuel-day at 1 MW | 6.26 Alpha |
| One fuel-day at 29.6 MW | 133.84 Alpha |
| Ten 3 MW stations / one 30 MW station | 1.3342× |
| Extra 30% output per Alpha for a whole 29.6 MW fuel-day, at full health | 66.35 kW-days |

The last value differs from the report's 78.9 kW-days because that comparison uses
only the cheapest marginal bracket, rather than the price of the whole fuel purchase.
Label marginal comparisons accordingly and show the actual purchase comparison too.

The stock figures also need a reconciliation: 70,744 burned + 13,042 staked + 98,908
liquid = **182,694 Alpha**, versus approximately **172,700** refined. Genesis funds
or another issuance source could explain the approximately 9,994 difference. Show
that source explicitly and avoid double-counting staked/module balances or omitting
guild/provider collateral. This is a data-accounting question, not evidence of a
chain exploit.

### The accepted supplier attack is not necessarily uneconomic

The cited 1,000 Alpha is refundable stake, not 1,000 Alpha spent per attack. Supply
can already exist, come from a generator, be shared with another strategic objective,
or be temporarily redirected. An attacker may value the outage far more than the bounty.
Acceptance makes it a consented risk; it does not make it economically moot.

Keep the mechanic, as already agreed. Change the report's claim and model reusable
supply, repeated acceptance and strategic damage separately from bounty profit.

### Preserve a reason to route through more than one station

A freshly maintained station can commit approximately **66.19%** of accepted supply
under the seven-day durable budget. Each station-to-station hop applies another
limit. Two such hops leave about **43.81%**, three about **29.00%**, before other
loads; receiving stations also incur their own upkeep costs when fuelled.

Combined with scale discounts, this strongly favors one large hub with directly
connected players. That can be intentional, but the current splitting test does not
measure the cost of a layered network. Model relay → neighborhood → players and
compare its resilience and price with one hub before choosing the final parameters.
Do not solve this immediately by adding another station class; first determine
whether the actual topology trade-off is attractive.

### Do not make all play more expensive merely to force an absorption ratio

Burns roughly matching issuance describe one stationary monetary-supply target.
They are not by themselves a definition of a healthy game economy. Demand for
holdings, uneven wealth, population and the distribution of losses matter too.

Build costs charged after repeated combat losses can make losing players unable to
rebuild while well-defended incumbents accumulate Alpha. Tripling draw and then
pricing construction per kW also compounds the changes. Measure the affordable
recovery path for new and defeated players, not only mean burn per active player.

Treat generator lifetime, mining/refining changes, construction costs, guild-bank
fees and adaptive issuance as separate proposals. Adaptive issuance in particular
needs an account-splitting/hash-participation model before calling it the inevitable
long-term solution. Keep the useful target of a modest infrastructure sink; it need
not solve the entire monetary economy on its own.

## Implementation and release corrections

- **Rebase the prerequisites against the current branch.** ICA hosting is already
  disabled in fresh genesis and the v0.22.0 migration, with an empty allow-list; see
  [app/ica.go](/Users/abstract/Documents/Structs/Code/structs/app/ica.go) and
  [upgrade policy](/Users/abstract/Documents/Structs/Code/structs/app/upgrades/v0_22_0/ica.go).
  That is implemented code, not confirmation of deployment. The blanket assertion
  that ICA currently allows everything is stale for this candidate.
- **Name the exact remaining guild-sale issue.**
  [GuildCache.SetOwner](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/guild_cache.go)
  already clears the outgoing owner's full direct guild grant. Existing rank-policy
  choices were explicitly accepted. Do not reopen those under the old generic label
  without a new reproduction. Cache coherence and worst-case gas still deserve
  targeted verification; this review does not certify those open areas as fixed.
- **Keep settlement work evidence-driven.** I agree with reducing my earlier broad
  closing-obligation proposal if real-app tests establish that valid payouts only
  fail for insolvency. However, “freeze provider” alone does not explain how its
  allocation stops contributing power, how billing stops, or how the consumer's
  amount remains protected. Specify those consequences before removing the fallback.
- **Budget consensus work deterministically.** A shared budget is a good objective,
  but 25–30M measured store gas is not inherently 10% of block time. Benchmark CPU,
  events and memory too. Use deterministic work accounting, never elapsed machine
  time to choose which consensus operations run. Existing exact-height agreement
  expirations cannot simply be skipped when a new shared budget runs out.
- **Shadow mode should not quietly become a global scan.** Replay an exported state
  and timestamped changes off-chain, or use explicitly bounded sampling. Label
  projected deadlines separately from events actually emitted by a block mutation.
  The indexer cannot be the sole source of information required to reproduce an audit.

## Recommended next revision

1. Specify the now-approved migration funding amount and accounting; correct the
   terminology and uptime claims to retain health-based output decline. Resolve
   post-purchase supply changes and who can afford the actual payment unit.
2. Write one integer formula specification covering acceptance, upkeep, fuel expiry,
   audit onset/proof, reward reservation, automated resizing and disrepair recovery.
3. Reconcile the economic input totals and generate all tables from one saved
   parameter set. Replace guaranteed race claims with probabilities.
4. Add the missing scenarios: cheap prepay then expansion, health-reset pricing,
   accepted input increases, precomputed proofs, upkeep during a race, one-time
   migration minting and supply reconciliation, partial payouts, whole-day member funding, automated-only routes,
   repeated claims and delayed multi-hop/cyclic cascades.
5. Shadow the design before choosing final prices. Measure ordinary players'
   willingness and ability to pay, audit participation and topology changes, alongside
   total burn and energy scarcity.

My strongest recommendation is to keep the fuel/accepted-supply/durable-contract
core and make its rules exact. That is enough new gameplay for one release. The
broader economy package should remain an experiment until those rules survive
adversarial and multi-station simulation.

## Review scope

Read both tabs of the supplied report and inspected the relevant current keeper,
proof-of-work, allocation, agreement, ICA and upgrade code. Recomputed the health,
tier pricing, expected hash-work and monetary-total examples using a local Python
arithmetic check. No production files or gameplay rules were changed; no simulator
sweep, chain test suite or live SQL query was run for this review.
