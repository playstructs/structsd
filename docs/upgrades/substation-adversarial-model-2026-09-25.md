# Adversarial model: can the proposed substation economy collapse?

September 25, 2026. **Yes: the model produces both blackouts and persistent energy
gluts.** Some are intentional strategic consequences, some depend on missing pricing
rules, and some follow from permanent generation rather than the substation system.
These are simulated design outcomes, not reproduced exploits in deployed chain code.

The most important result is that the current proposal does not by itself create
lasting energy scarcity. A guild can accumulate permanent supply and maintain useful
service with occasional upkeep instead of daily purchases. The strongest unresolved
pricing exploit is even simpler: briefly reduce accepted supply, buy cheap upkeep,
and restore supply after the purchase has refreshed the whole station's health.

## Artifacts and scope

- [Model and reproduction instructions](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/README.md)
- [Candidate parameters](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/parameters.json)
- [Full results, configuration, seed and source fingerprints](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-model-results-2026-09-25/results.json)
- [Scenario charts](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-model-results-2026-09-25/collapse-scenarios.png)

Ran **1,308 scenario rows**, including **36,000 seeded stochastic audit races**, and
**14 model tests** with additional randomized arithmetic checks. CSVs sit beside
the JSON. The original external simulator and live indexer extracts were unavailable;
this is a newly built model using the supplied report's aggregate inputs. No live
network transactions, chain reset or production code changes were made.

The model respects the latest decisions: declining health during paid upkeep,
no scheduled enforcement of age, exclusive automated outputs, oldest-first ordinary
shedding, collateral bootstrap minting and whole upkeep purchases from liquid Alpha.
It does not convert guild rank tokens into Alpha or add generator expiration.

## Results at a glance

| Attempt | Result | Interpretation |
| --- | --- | --- |
| Buy cheaply, then restore accepted supply | 0.0095 Alpha refreshes a 100 MW station instead of 401.36 Alpha | Unresolved pricing rule; the first proposed safeguard is insufficient |
| Maintain enough permanent supply, buy upkeep infrequently | Full modeled demand served for 9.08 Alpha/day, about 0.30% of 3,000 Alpha/day issuance | A legal strategy that defeats the assumed 2–10% recurring sink in this glut case |
| Invest 10% of issuance in existing permanent generation | Available capacity reaches 15.97× fixed demand after one year | Upkeep does not prevent accumulating a glut |
| Withdraw 6% of an accepted hub input | 99% of outgoing commitments disappear in the concentrated-oldest case | Intentional policy with a very large amplification of damage |
| Start with an unfunded treasury and insufficient income contributions | Production ceases before the treasury can afford upkeep; projected output eventually reaches zero | Conditional closed-funding recovery trap |
| Neglect a relay; maintain its downstream station; no auditor arrives | Downstream still shows 65.01 MW at day 60 although relay projection is zero | Player-enforced commitments can outlive their projected backing |
| Audit that neglected relay | Downstream capacity falls to zero; reward is 0.0625 Alpha at difficulty 5 | Enforcement works once a valid player action occurs |
| Use a healthy lossy cycle to print unlimited power | Failed; useful output stayed below external supply and drained to zero after withdrawal | Negative control: ordinary settled routing is not an unlimited energy source |

## 1. Cheap health restoration survives the obvious prepayment fix

Start with a 100 MW leaf station, no outgoing contracts and 29 days since its last
upkeep. Its projected unfuelled output is about **4.59 MW**.

The owner temporarily accepts only **1 kW**, purchases one whole upkeep day, then
restores the original supply. At the proposed prices, the payment is **0.0095 Alpha**.
The ordinary 100 MW payment would have been **401.36 Alpha**.

| Pricing rule tested | Payment | Output after restoring supply |
| --- | ---: | ---: |
| Expiry alone covers whatever supply is present | 0.0095 Alpha | Approximately 100 MW |
| Temporary benefit limited to paid capacity, but payment restores health globally | 0.0095 Alpha | Approximately 70 MW |
| Control: bill the station's full operating ceiling despite the temporary input cut | 401.36 Alpha | Approximately 100 MW |

The second row matters most. **My earlier paid-capacity ceiling only fixes the
temporary benefit; it does not fix buying a whole-station health reset cheaply.**
The resulting saving exceeds 99.99% in this example.

This attack assumes the owner can lower and restore its own accepted supply, and
that the final formula prices the purchase from that temporarily lowered supply.
It is not a current-chain exploit: the proposed messages do not exist yet. No
unauthorized transfer or disappearing consumer refund is assumed. The station has
no outgoing allocations, so a transient input reduction need not destroy contracts.

**Required design work:** tie both restored condition and prepaid benefit to the
paid operating capacity. A sticky billable operating ceiling closes this example,
but legitimate downsizing and expansion still need an explicit rule. Do not certify
the price algorithm merely because it caps the temporary 30% benefit.

## 2. A legal low-upkeep strategy weakens the recurring sink

I swept 572 combinations of supply, purchase interval and one- or seven-day payments.
The objective was full service for **16.3 MW demand** at the lowest average burn,
not maximum output or the highest possible commitments.

One tested policy maintains:

- **32.6 MW** accepted supply, twice demand.
- **One paid upkeep day every 16 days.**
- **145.24 Alpha** per purchase, or **9.0775 Alpha/day**.
- Worst capacity immediately before renewal: just above demand, **1.00178×**.

The station serves the full fixed demand throughout the modeled cycle. Average
upkeep is **0.303% of 3,000 Alpha/day issuance**, not the desired 2–10% sink. This
does not use the cheap-reset loophole: every purchase pays for the full supply.

It exploits the product being sold: one purchase buys temporary extra output **and
restores health for the following weeks**. If durable supply is sufficient, the
guild mainly wants that infrequent health restoration.

The original report's different glut strategy—trim supply almost exactly to demand
divided by 0.7 and buy one day every 29 days—does **not** preserve uptime in this
model. Once the paid day ends, even slight health decay puts supply below the full
load. In the tested 1.43×-supply case, full-load uptime is only **3.45%** of the cycle,
although partial capacity remains available. These two strategies must not be
confused: adequate excess supply makes cheap upkeep viable; tight supply does not.

**Interpretation:** cheap infrequent upkeep may be an acceptable reward for investing
in excess infrastructure. If so, stop treating daily upkeep spending as the steady
state for every guild. If a larger guaranteed sink is essential, the economics need
another reason to spend—not a more aggressive assumption about player behavior.

## 3. Permanent supply can overwhelm the upkeep sink

For an intentionally hostile-to-scarcity scenario, I held demand at **16.3 MW**,
started with **153.9 MW** raw supply, allowed **3,000 Alpha/day** issuance and no
successful attacks on generators. Upkeep is fully priced for all accepted supply
once every 14 days. Newly built supply waits until the next purchase before entering
the station, avoiding the cheap-prepayment assumption above.

Using the lowest generator yield in the report, **2 kW per Alpha**:

| Issuance invested in generation | Raw supply after 365 days | Available capacity / demand at day 365 | Total upkeep burned |
| --- | ---: | ---: | ---: |
| 0% | 153.9 MW | 6.60× | 16,366.86 Alpha |
| 10% | 372.9 MW | 15.97× | 27,570.78 Alpha |
| 50% | 1,248.9 MW | 53.45× | 72,386.46 Alpha |

The experiment spends real modeled Alpha on generator infusion and upkeep. The
aggregate cash ledger balances: starting liquid Alpha plus issuance equals ending
cash plus those burns. Permanent capacity accumulates even though construction
burns Alpha once. The 10% case would still have over three times the stated demand
if that demand increased fivefold by year end.

This does not predict that players will keep overbuilding. It shows that a wealthy
operator *can* sustain and deepen a glut within these assumptions. Other players
could destroy the infrastructure, demand could grow, or the game could adopt another
supply sink. None of those outcomes is produced by station upkeep alone.

I would keep this distinct from a monetary collapse. Oversupply can make energy
cheap and upkeep unattractive without proving that Alpha itself has no value.

## 4. A small withdrawal can erase nearly all contracted service

The test hub has **100 MW** accepted input and admits **66.1889 MW** of outgoing
commitments under the seven-day safe budget. The oldest allocation carries 99% of
those commitments; the newer one carries 1%.

Withdraw **6 MW**, leaving 94 MW input. Fresh unfuelled throughput is now 65.8 MW,
only about **0.389 MW** below installed commitments. Oldest-first deletion removes
the entire **65.527 MW** oldest allocation. The older group's incoming service is
zero, and only about **0.662 MW** of outgoing commitments survives.

This uses ordinary supply-loss cascade, not a prepared audit proof. The attacker
must already control accepted input. The underlying source can still have ample
capacity; the outage is caused by contract shedding and network organization.

When the older group represents its same entitlement as **100 smaller allocations**,
the identical withdrawal destroys only **0.99%** of total commitments instead of
99%. Thus the policy creates a strong incentive to fragment allocations. That
increases the number of rows and work needed when a larger outage arrives.

You have explicitly accepted destructive supply withdrawals and oldest-first
shedding. I would not silently change either. This result puts numbers on the
trade-off: fragmentation and replacement planning become defensive strategies.
Show the next allocation to be shed in owner tooling and benchmark fragmented
networks before selecting block-work budgets.

## 5. A guild can run out of production before it can afford upkeep

This fixture is synthetic: **29.6 MW** supply serves **16.3 MW** of fixed player
load. It starts healthy with funded station collateral but zero spendable treasury
Alpha. Maximum income while operational is 3,000 Alpha/day, of which a selected
fraction actually reaches the upkeep treasury. Rank-token revenue is not counted
as spendable Alpha.

The guild tries to purchase upkeep when needed and retries when it cannot afford
the **133.84 Alpha** whole-day bill.

| Share of online income reaching treasury | Player response | Outcome |
| --- | --- | --- |
| 0.10%: 3 Alpha/day while earning | Keep full load | Full-load readiness fails around day 13.86; treasury stalls at 41.59 Alpha; output reaches zero |
| 0.10% | Shed to 3.5 MW essential load | Production lasts longer, but treasury still stalls below one purchase, at 82.06 Alpha |
| 0.25%: 7.5 Alpha/day | Keep full load | Treasury stalls at 103.98 Alpha, still below the purchase threshold |
| 0.25% | Shed to essential load | Recovery succeeds; full-load uptime over 90 days is about 77% |
| 0.50%: 15 Alpha/day | Keep full load | No modeled outage; treasury accumulates enough before readiness fails |

Once projected capacity is zero and the closed model has no outside funds, it
cannot earn its way out. This is a **conditional funding trap**, not a claim that
real guilds cannot recover: member savings, treasury token redemption, trade,
sponsorship, load reorganization and new supply are excluded from this fixture.
Essentials-only operation assumes players can execute the required load changes.

**Simple mitigation:** the interface should show spendable **Alpha**, the full upkeep
price, and how much time remains before the current load stops fitting. Encourage
a liquid operating reserve. The migration's collateral mint does not provide that
reserve: those coins secure bounties and are not upkeep spending money.

## 6. Player-enforced commitments can persist without projected backing

The graph experiment has a 100 MW relay feeding a maintained downstream station
through its sole automated allocation of **66.1889 MW**. The relay receives no
upkeep. The downstream station buys seven days at a time. Nobody audits for 60 days.

At day 30 the relay's projected throughput is zero, but its outgoing allocation is
still installed. At day 60 the downstream station still reports **65.01 MW** available
from that claim. There are no automatically queued upkeep jobs, exactly as requested.

A valid difficulty-5 audit at day 60 resizes the relay output to zero, pays
**0.0625 Alpha** from its reserve, and downstream available capacity becomes zero.
The modeled queue drains in one block for this small graph.

This is not proof of unlimited power or an accounting exploit. It is the consequence
of intentionally allowing commitments to outlive their backing until someone acts.
Both bounty hunters and hostile teams can end it. The practical open question is
how consistently they will do so across many stations for the available rewards.
Keep the player-driven design; test participation, reward size and coverage rather
than introducing the scheduled shutdown you rejected.

## 7. Prepared proofs accelerate audits without removing the difficulty gate

The race simulation uses the repository's integer difficulty equation, the two
supplied severity endpoint ranges, independent random hashes and an aggregate
2 MH/s hasher against a difficulty-5 sniper. Each cell below uses 2,000 trials.

| Case | Work begins at imbalance: median | Seven days of work prepared within same known upkeep cycle: median |
| --- | ---: | ---: |
| Severe endpoint | 1.09 hours | 0.76 hours |
| Small-imbalance endpoint | 29.53 hours | 20.68 hours |

Prepared work reaches the maximum reward multiplier in these trials. It cannot
claim immediately at the instant imbalance begins: the current required difficulty
still applies. Preparation assumes the anchor/nonce does not change meanwhile.

These runs support treating advance work as a strategic advantage rather than
assuming it destroys the mechanic. They do not establish profitability: electricity,
hardware, competing refining work and transaction inclusion are not priced. Severity
is held fixed because the proposed interpolation table was not supplied. Rewards
are based on actual achieved difficulty, unlike the current helper's truncated count.

## Controls, sensitivity and limits

- A healthy two-station automated cycle settled without useful output exceeding
  its 100 MW external source. Removing that source drained both edge powers to zero
  in 31 toy-budget blocks. I did not find unlimited energy from that cycle.
- With a 0.5 Alpha reserve and 0.01 Alpha dust threshold, repeated **distinct** full
  reductions exhaust collateral after six difficulty-9 audits or 30 difficulty-5
  audits. Every iteration needs a new real imbalance; completed audit replay is
  rejected by the model. Total payouts never exceed the reserve.
- Fragmenting a full shutdown into 8,192 outgoing allocations took 34 model blocks
  with a 256 visit/edge-operation budget, versus one block for a single allocation.
  This is illustrative work accounting, not measured Cosmos gas or a production
  block-time guarantee.
- Destroying the supplied report's 141.5 MW giant generator leaves 12.4 MW raw
  supply. Fresh durable output is 8.68 MW against 16.3 MW demand. Equal-share,
  identical full loadouts all fail readiness, while a 3.5 MW essentials configuration
  still fits. The model assumes destruction happened; it does not prove the combat
  attack is feasible.
- No model here tests provider refunds, live ante/permissions, vesting, actual
  multi-block audit escrow, development upgrades or the production gas ceiling.
  Numeric bounds in this report are evidence for these scenarios, not exhaustive
  security or economic guarantees.

## What I would change or decide next

1. **Close cheap health restoration before further price tuning.** The billable
   capacity must govern both health restoration and temporary output benefits.
2. **Decide whether a glut is an allowed strategic outcome.** Under permanent supply
   and bounded demand, station upkeep alone cannot guarantee scarcity. If abundance
   is allowed until combat destroys it, model that explicitly rather than claiming
   recurring upkeep automatically balances the economy.
3. **Model operating reserves in Alpha separately from guild wealth.** Rank tokens,
   bounty collateral and liquid upkeep money are different resources.
4. **Preserve the agreed destructive policies, but expose their consequences.**
   Oldest-first is deterministic; it also rewards fragmentation and can make a
   tiny deficit cause a huge contractual outage.
5. **Test auditors as economic actors.** Prepared work, neglected routes, low-value
   claims and competitors' incentives matter more than assuming an auditor always
   appears. No scheduled enforcement is proposed.

The highest-priority implementation issue is the pricing loophole. The highest-priority
economic decision is whether permanent overcapacity is something players are allowed
to achieve and defend. Neither is solved merely by tuning the current daily price.
