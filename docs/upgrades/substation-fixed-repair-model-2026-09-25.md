# Fixed-price substation repair: design and model results

September 25, 2026. **This is much simpler, and the modeled payment mechanics work.
Its economic tradeoff is strong: large consolidated stations become very cheap to
maintain.** I recommend it as a maintenance and sabotage mechanic if that is an
acceptable outcome, rather than expecting it to regulate total energy abundance.

This report explores option 5 at the user's request. It does not adopt a production
fee or implement chain changes. Previous rating and prepaid-capacity models remain
available for comparison.

## The proposed player experience

> A substation loses condition over time. Pay a fixed amount of Alpha to repair it.
> Neglected stations lose output and can become vulnerable to imbalance sabotage.

One repair transaction burns the fee and restores health to 100%. There are no
purchased days, temporary output boosts, capacity ratings, expansion charges or
remaining-service balances. Adding supply needs no repair-pricing calculation.

The modeled output rule is:

```text
health = max(0, 1 - (blocks since repair / wear duration)^2)
output = floor(accepted supply × 70% × health)
```

The 70% baseline is retained from the existing unfuelled design. Removing the bonus
does not silently raise the baseline to 100%. A 100% baseline appears only in a
sensitivity sweep, not the recommendation.

At fresh health, 100 MW of accepted input produces 70 MW. Seven model days later it
produces about 66.19 MW; after 30 it projects zero. Repair restores 70 MW, provided
the input is still present. It does not recreate destroyed allocations or restore
missing supply.

Keep the agreed seven-day forecast for new outgoing commitments, exclusive
automated output, oldest-first ordinary shedding, and bounded GridCascade. Keep
the agreed collateral reserve and zero-collateral disrepair rule. A repair is not
a collateral deposit; exhausted collateral must be funded through the recovery
rules before operation resumes.

**No automatic upkeep scan or age-triggered shutdown is introduced.** Time changes
projected health. Players still enforce unsupported outgoing commitments through
audits, and actual supply reductions trigger normal propagation.

## Results at a glance

The illustrative fee is **10 Alpha per repair**, with sensitivity runs at 1, 50,
100 and 500 Alpha. These are test inputs, not a selected live price.

| Question | Result |
| --- | --- |
| Can disconnecting supply reduce the repair bill? | No. Zero input, 1 kW, 100 MW and much larger input all pay the same fee. |
| Can cheap repair followed by expansion obtain unpriced capacity? | Expansion benefits from the station's current health at no extra charge. This is deliberately included in flat pricing, not a billing bypass. |
| Can repeated payments accumulate prepaid service? | No. Each resets the repair timestamp; none buys future time. Duplicate repair in the same block is rejected in the model. |
| Can replacing the station avoid repair costs? | Yes if creation supplies fresh health for less than a repair. Including the first repair in the creation charge closes this price bypass. |
| Can splitting a station reduce the bill? | No. Each station owes the same fee. Parallel splitting preserves approximately the same aggregate output but multiplies repair costs. |
| Can large stations remain cheap indefinitely? | Yes. Repair cost does not scale with supply, and extra headroom permits longer repair intervals. |
| Can insufficient funding cause a blackout? | Yes. Lower fees make recovery easier, but a treasury can still fail to accumulate a whole repair payment before production stops. |
| Does this prevent a glut? | No. Permanent generation still accumulates; flat pricing makes maintenance an even weaker counterweight. |

Ran **1,126 scenario rows** plus a specific audit-enforcement scenario. All **43
model tests passed**, including 12 new tests, 2,000 exact outage-boundary checks,
1,000 parallel-partition checks, and a lossy-loop withdrawal regression. The full
suite also retains the previous rating and original-design tests. Outputs were
replayed and checked for byte-identical JSON and CSV files.

![Fixed repair model charts](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-fixed-repair-results-2026-09-25/fixed-repair-scenarios.png)

## 1. The simple price is internally consistent

Under supply-based pricing, disconnecting input let the owner buy a cheap purchase
that repaired a much larger station. Under this rule, repair has an explicit fixed
price for the whole station, including whatever supply it accepts later.

The model deliberately allows repairing an empty station and connecting substantial
supply afterward. It costs exactly what repairing after connecting would cost.
There is no need to remember a previous peak or charge a top-up. Health still ages
from the actual payment; connecting supply later does not refresh it.

This simplicity comes from giving up proportional pricing. It should not be
described as making large stations pay their proportional share by another method.

## 2. Repair timing remains a meaningful decision

For a station directly supporting fixed demand, the approximate maximum interval
before losing full service is:

```text
interval = wear duration × sqrt(1 - demand / (70% × input))
```

This expression requires fresh output to meet demand. The model finds the exact
first failing block with integer arithmetic; the square root is an explanatory
formula, not proposed consensus code. Repair at that block assumes the payment is
included before the operation that needs the restored capacity. Real players need
a margin for inclusion delays.

With **16.3 MW demand** and a **10 Alpha fee**:

| Raw supply | Latest repair interval, model days | Long-run repair burn, Alpha/day |
| --- | ---: | ---: |
| 24.45 MW | 6.55 | 1.53 |
| 29.6 MW | 13.86 | 0.72 |
| 32.6 MW | 16.04 | 0.62 |
| 48.9 MW | 21.71 | 0.46 |
| 153.9 MW | 27.64 | 0.36 |

Being barely large enough is expensive: 23.286 MW barely supports that demand at
fresh health and needs repair about every **0.105 model days**, costing roughly
95.15 Alpha/day at the same fixed fee. A station exactly at fresh capacity loses
headroom on the next block. The useful strategy is spare capacity, not frantic
repairing at the physical limit.

Stations carrying outgoing contracts remain subject to the seven-day admission
buffer. At a fresh station's maximum admitted outgoing load, seven days of wear
uses that buffer. The longer intervals above concern appropriately smaller load,
including players served directly; they do not bypass contract admission.

Repairing late saves money but reduces safety margin against supplier withdrawal
and rival audits. Repairing early costs more but preserves headroom. The model
does not quantify how often players will attack, or claim that either strategy is
universally optimal once attacks are included.

## 3. Consolidation is the strongest incentive

Compare 100 MW total raw input and 50 MW total demand split evenly across independent
parallel stations. Each has the same health curve and approximately 16.04-day repair
interval. With a 10 Alpha repair fee:

| Parallel stations | Total repair burn, Alpha/day | Capacity exposed if one station completely fails |
| --- | ---: | ---: |
| 1 | 0.62 | 100% |
| 2 | 1.25 | 50% |
| 10 | 6.24 | 10% |
| 100 | 62.36 | 1% |

The failure column is a topology comparison, not a prediction that every sabotage
destroys a whole station. Shared upstream dependencies can also destroy the apparent
resilience of a split design.

There is a potentially interesting choice here: cheap central infrastructure versus
more expensive isolation and redundancy. But without sufficiently valuable isolation,
the financial incentive overwhelmingly favors large shared hubs. The model does not
establish that combat risk will balance it automatically.

Serial routing is also expensive. With 100 MW at the root, fresh stations and each
link committed at its safe admission budget, useful leaf output is about **70 MW**
with one station, **46.33 MW** with two, and **30.67 MW** with three. Each extra
station adds a repair fee as well as another conversion loss. There is no free
output bonus from creating empty stations.

## 4. Creation must include the first repair

If a freshly created station costs only the earlier candidate 0.5 Alpha founding
burn plus 0.5 Alpha collateral, a neglected station with a 10–500 Alpha repair fee
may be cheaper to replace. Moving players and contracts may impose friction, but
that should not be the sole economic protection.

**Recommended simple rule:** creation automatically charges the normal fixed repair
fee as part of its initial cost and starts the station fully repaired. This needs
no separate commissioning transaction or operating cap.

At the illustrative 10 Alpha fee, the candidate creation total is **11 Alpha**:
10 burned for the initial repair, 0.5 founding burn, and 0.5 deposited as collateral.
Even assuming all old collateral can be recovered, replacement still burns more
than repairing. Re-routing and other setup costs are omitted, favoring the replacement
strategy in this comparison. Actual deletion/refund rules are not implemented here.

Existing stations can receive the already planned one-time migration initialization
and collateral mint. The ordinary creation path must not expose that free migration
initialization. The model distinguishes seeded migration fixtures from paid creation;
normal creation funds its collateral from the creator and mints nothing.

## 5. A low fixed fee helps affordability; it cannot guarantee recovery

The treasury experiment uses a synthetic 29.6 MW station supporting 16.3 MW demand.
While the full load operates, the modeled cohort earns 3,000 Alpha/day, of which a
chosen fraction reaches the repair wallet. It begins healthy with funded collateral.
Results below start with **zero liquid repair funds** and no fallback load:

| Repair fee | Income reaching treasury | Result over 90 model days |
| --- | --- | --- |
| 10 Alpha | 0.01%, or 0.3 Alpha/day | Fails around day 13.86; only 4.16 Alpha accumulated. |
| 10 Alpha | 0.1%, or 3 Alpha/day | Full observed uptime; six repairs. |
| 100 Alpha | 0.1%, or 3 Alpha/day | Fails around day 13.86; only 41.59 Alpha accumulated. |
| 100 Alpha | 0.25%, or 7.5 Alpha/day | Full observed uptime; six repairs. |
| 500 Alpha | 1%, or 30 Alpha/day | Still fails before it can afford the first repair. |

Other rows test a prefunded repair and shedding to an essential load that can keep
earning. Outside sponsorship, existing player savings and market transfers are
excluded, so a failed closed scenario does not mean real players have no recovery
path. Guild rank revenue is still the guild's token; it is not automatically liquid
Alpha available for repair.

This version makes whole-payment sponsorship easier at a low fee without adding
micropayment buckets. Anyone permitted to spend their own funds can sponsor the
fixed repair; sponsorship need not confer administrative rights.

## 6. The sink depends on station count, not economic scale

With `N` active stations, repair fee `F`, and typical repair interval `I`:

```text
maintenance burn per day ≈ N × F / I
```

For two stations repairing every 14 days, achieving the old target of 2–10% of an
assumed 3,000 Alpha/day issuance requires **420–2,100 Alpha per repair**. For 100
stations at that cadence, it requires **8.4–42 Alpha**. A price calibrated to a
small number of huge hubs can be punishing for small independent stations.

At 10 Alpha and the previous 14-day comparison cadence, a seeded station burns
**260 Alpha in subsequent repairs over 365 days**, regardless of whether it handles
153.9 MW or grows to 1,248.9 MW. The initial migration health is free in this
scenario; an ordinary new station would also pay its first repair at creation.

With 10% of issuance reinvested in permanent generation, available capacity reaches
approximately **16× fixed demand** by day 365. At 50%, approximately **53.57×**.
These assume no combat, fixed demand and 2 kW per Alpha added to existing generators.
The supply grows immediately; no expansion surcharge or deferred acceptance is used.

**The flat fee cannot independently regulate an economy that grows while station
count stays small.** Raising it scales everyone's bill; it does not remove the
advantage of consolidation. The persistent 30% loss and health decay consume usable
capacity, but are not an ever-growing demand for energy.

## 7. Player enforcement is preserved, including its consequences

The neglected-relay test still behaves as agreed. At day 31 the relay projects zero,
while its stored 60 MW allocation lets a freshly repaired downstream station show
42 MW. No age-triggered work is queued. A valid modeled audit reduces the relay's
automated output to zero, removes downstream power, and pays a 0.25 Alpha bounty
from the candidate 0.5 Alpha reserve at the maximum reward multiplier.

Thus cheaper repair does not make stale upstream claims self-enforcing. Auditors
remain necessary. Repairing after ordinary allocations were destroyed does not
recreate those contracts.

Retain accurate proof-anchor events if repair updates the last-upkeep anchor. That
can invalidate prepared audit work as in the earlier design. A repair must not be
treated as proof that every imbalance is cured: missing input can leave commitments
unsupported even at full health. This experiment assumes valid audit proofs; it
does not rerun the previous mining-race simulation or establish a new proof policy.

## Recommendation and implementation outline

I prefer this to the operating-rating design for the simplicity the user wants.
Keep the single repair action, retained 70% baseline, health decay, and existing
player-driven audit consequences. Include the first repair in creation. Use **10
Alpha only as an initial playtest point**, alongside cheaper and more expensive
variants; the model does not establish a production-optimal fee.

Accept explicitly that large hubs benefit and that this is a modest maintenance
sink. If strong proportional energy consumption is required, that needs another
gameplay mechanic. Adding capacity tracking back into this repair rule would undo
the simplification.

Implementation would proceed as follows:

1. Specify a fixed `repair_cost_ualpha` parameter, health duration, and retained
   baseline. Remove prepaid-duration and capacity-rating concepts from the selected
   design; do not remove unrelated guild, vault or reactor changes.
2. Add one repair message using the established CurrentContext and signer pattern.
   Validate spend authority, destination existence, non-closing state, required
   collateral and available funds before burning and updating the health timestamp.
   Reject an already repaired same-block state without charging again.
3. Include the first repair payment in normal creation. Keep migration funding and
   initial health a distinct one-time initialization. Preserve aggregate bank supply
   and emit accurate funding, burn and address-association events.
4. Calculate projected output and admission budget lazily with integer arithmetic.
   Preserve bounded cascade, settlement ordering, audit reward solvency and recovery
   from disrepair. Repair grants neither administrative authority nor free collateral.
5. Expose repair price, last repair, projected health/output and collateral through
   queries/events. The interface can estimate time to insufficient capacity and
   display affordable repair/recovery choices without promising guaranteed uptime.
6. Port the regression cases to keeper and real-app tests, including bank rollback,
   unauthorized sponsorship, creation/replacement, same-block messages, audit races,
   migration and bounded propagation under real gas. Update `test_chain.sh` when
   implementation is actually chosen.

There is no new growing financial history, graph scan for pricing, capacity cap
transaction or upkeep schedule. Model inheritance retains unused old Python fields
for compatibility; they are not required fields in this alternative's chain design.

## Assumptions, reproduction and artifacts

The model uses the previous report's **15,500 blocks per model day** for comparability.
At an exact five-second block, a calendar day is 17,280 blocks; production duration
parameters should use the intended convention. The normalized daily figures here
are model-day figures, not measurements of current chain wall time.

Treasury observations are 96 ticks per model day. Repair thresholds are exact in
the separate cadence calculation; treasury outage times are sampled. Revenue,
wealth, demand and generation are scenario assumptions. No live graph export,
combat execution, market price, permission implementation or Cosmos gas benchmark
is included. Allocation sorting and graph scans inside Python are modeling tools,
not recommendations for consensus code.

```sh
python3 -m unittest discover -s tools/substation_model -p 'test_*.py' -v
python3 tools/substation_model/fixed_repair_experiments.py
python3 tools/substation_model/fixed_repair_experiments.py --out build/substation-fixed-repair-replay
build/substation-model-venv/bin/python tools/substation_model/plot_fixed_repair.py build/substation-model/substation-fixed-repair-results-2026-09-25/results.json
```

- [Model](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/fixed_repair_model.py)
- [Tests](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/test_fixed_repair_model.py)
- [Experiment runner](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/fixed_repair_experiments.py)
- [Results, configuration and source fingerprints](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-fixed-repair-results-2026-09-25/results.json)
- [Chart](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-fixed-repair-results-2026-09-25/fixed-repair-scenarios.png)

CSV files alongside the JSON contain individual cases. No production chain code
was changed by this exploration.
