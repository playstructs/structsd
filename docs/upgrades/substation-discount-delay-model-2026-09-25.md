# Delayed upkeep discounts: model results

September 25, 2026. **The delay stops an immediate disconnect–repair–reconnect
discount at one station. It does not reliably impose downtime on connected players.**
Players can rotate between active and standby substations, allowing the standby
station to qualify for cheap upkeep while another serves the load.

I would not adopt this as the sole fix for the connected-player case. It introduces
pricing memory and delays legitimate discounts without removing the larger bypass.
It can be a speed bump if station rotation is intentionally acceptable gameplay.

## Rule modeled

Retain supply-based upkeep pricing, whole-day purchases and whole-station health
restoration. Remember a protected supply amount after a reduction. A lower price
becomes available only after accepted supply stays unchanged for the chosen delay.

- Before qualification, upkeep remains available at the protected price.
- Every actual accepted-supply change below the protected amount restarts the delay.
- Restoring supply to the protected amount cancels qualification.
- A later reduction starts a fresh waiting period, even if an earlier discount
  had already matured.
- Unaccepted inputs do not affect the price or timer.
- Time is checked lazily when quoting or mutating supply. There is no scheduled
  upkeep scan or automatic shutdown.

This implementation uses a protected supply value and an optional reduction
timestamp per station. It is a conservative, constant-size interpretation of
"the reduced level must persist." It does not implement a moving-window maximum
or keep a history of every supply change.

The main experiments also limit the temporary bonus to paid capacity, as a
conservative control. Extra restored input still gets the baseline 70% output
multiplied by the station's globally restored health. The bypass therefore does
not depend on giving expanded supply a free temporary 100% bonus.

## 1. The immediate trick is stopped

For a 100 MW station temporarily reduced to 1 kW:

| Pricing and timing | One-day payment | Existing 60 MW outgoing allocation |
| --- | ---: | --- |
| No guard; restore before cascade | 0.0095 Alpha | Survives |
| Seven-day guard; restore before cascade | 401.36 Alpha | Survives |
| No guard; stay reduced through cascade for seven days | 0.0095 Alpha | Destroyed |
| Seven-day guard; stay reduced through cascade for seven days | 0.0095 Alpha | Destroyed |

This supports the user's distinction: **a sustained reduction can destroy valuable
outgoing allocations, whereas connected-player service can be restored by adding
power back.** Outgoing contracts do not themselves prevent the immediate variant.

The current code queues reductions and later checks whether load still exceeds
capacity. Restoring capacity before that check can avoid shedding. See
[GridCascade](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/grid_context.go:243)
and [allocation destination changes](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/allocation_cache.go:325).
The Python experiment reproduces that ordering; it does not execute a proposed
upkeep transaction against a live chain, since that feature is not implemented.

## 2. Two stations remove the intended downtime penalty

Use two owner-controlled stations:

1. Station A serves the players from 100 MW of input. B holds only 1 kW.
2. B remains small for at least seven days, qualifying for the small upkeep price.
3. At the switch, pay **0.0095 Alpha** to restore B's health.
4. Move the large input and authorized player connections to B. Leave A with 1 kW.
5. A waits while B serves the players. Repeat in the other direction.

The active station serves **50 MW demand** with switches every **14 model days**.
Its minimum output over every served interval remains about **54.76 MW**, even
with the temporary benefit capped to the tiny paid supply. Total input is conserved:
100 MW active plus 1 kW for the standby. No extra generation is assumed.

Over **26 purchases**:

| Strategy | Upkeep burn | Modeled full service |
| --- | ---: | --- |
| One station, ordinary full-size upkeep every 14 days | 10,435.36 Alpha | Maintained |
| Two stations rotating with a seven-day discount delay | 0.247 Alpha | Maintained |

The comparison uses the same purchase cadence, not a claim that full-size upkeep
every 14 days is the globally optimal honest strategy. The discount exceeds 99.99%
on each repeated purchase at that cadence.

### What "maintained" assumes

Supply reassignment and authorized player migration complete in one logical switch.
Intermediate message states may briefly lack service, but no served block boundary
needs to expose the low-power station. Supply is checked at the last block before
each switch and after each switch; monotonic health decay covers the interval.

The current handlers support moving allocation destinations and migrating authorized
players between substations. The player handler explicitly disconnects the previous
connection as part of migrating. See
[SubstationPlayerMigrate](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/msg_server_substation_player_migrate.go:9)
and [SubstationAllocationConnect](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/msg_server_substation_allocation_connect.go:9).

This is evidence that movement is part of the design, not a real-app reproduction
of the complete future transaction. Permissions must be held, and moving a large
cohort may require multiple transactions under the gas cap. The economic example
does not assume permission to move arbitrary players. It excludes transaction
costs, inclusion delays, setup costs, provider settlement and migration contention.
Those can add friction; they are not an established protection against rotation.

The stations are seeded existing fixtures. No repeated creation or free collateral
mint is used in the loop. An extra station has an initial cost and tied-up collateral,
which must be added to a real strategy's budget. A fixed one-time cost does not
remove the recurring pricing advantage. A rotation station has no outgoing contracts
in this scenario, so it avoids the very destruction cost that distinguishes them
from connected-player service.

## 3. Increasing the delay requires more standby stations

With a 14-day active interval:

| Discount delay | Stations in rotation | Time each waits before reuse | Steady repair price |
| --- | ---: | ---: | ---: |
| 7 days | 2 | 14 days | 0.0095 Alpha |
| 14 days | 2 | 14 days | 0.0095 Alpha |
| 21 days | 3 | 28 days | 0.0095 Alpha |
| 30 days | 2 | 14 days | 401.36 Alpha |
| 30 days | 4 | 42 days | 0.0095 Alpha |

A sufficient station count is `1 + ceil(delay / active interval)`, assuming spare
input for idle stations and permission to switch service. An idle station can reach
zero projected health and still be repaired before use if its collateral remains
funded; it has no outgoing imbalance to audit in this fixture.

Longer delays raise setup requirements. They do not force the same players to go
without power for the waiting period.

## 4. Even one station does not necessarily lose useful service

A station with substantial spare supply can lower its input while keeping enough
for existing demand. One modeled case begins at 100 MW with seven days of wear,
reduces to **32.6 MW for seven days**, and still supplies at least **17.85 MW**
throughout, enough for **16.3 MW demand**.

It then qualifies for a **145.24 Alpha** one-day upkeep purchase instead of
401.36 Alpha, and can restore larger input after the purchase. That is a 63.81%
discount without losing the modeled useful service.

Reducing unused excess is a legitimate optimization, not necessarily an exploit.
It does show why "seven days of reduced capacity" cannot be equated with "seven
days of lost production." The value of unused headroom, military vulnerability and
future demand are separate questions.

For a single station that really does lose all useful output while reduced to 1 kW,
the saving is **401.3505 Alpha per one-day purchase**. Across a seven-day wait,
foregone earnings would have to exceed about **57.34 Alpha/day** to outweigh that
one-purchase saving on earnings alone. The model sweeps hypothetical lost income;
it does not assert a real station's earning rate or price the strategic risk.

## 5. A naive timer has another bypass; the strict timer has a cost

If qualification counts only from the first reduction:

1. Reduce 100 MW to 99 MW and start the seven-day timer.
2. One block before expiry, reduce to 1 kW.
3. Buy at the 1 kW price when the old timer matures.

The deep reduction lasted only one block. This was reproduced in the deliberately
weaker counterfactual model. Requiring the *new* reduced supply to persist blocks it.

However, restarting qualification after every change means small fluctuations can
preserve an old price indefinitely. A station that drops from 100 MW to around
1 kW, then oscillates by **one mW per day**, still quotes 100 MW after 29 days in
the strict model. Upkeep remains possible at 401.36 Alpha; it is the discount that
is denied.

This can inconvenience legitimate variable supply or be induced by a controller
of an already accepted input. Unaccepted unsolicited inputs are excluded. A more
forgiving rule would need another precise policy and fresh adversarial tests;
ignoring "small" changes naively invites a sequence of individually small cuts.

## Recommendation

**Do not add this delay on the assumption that it solves the connected-player
loophole.** It does solve the single-station instantaneous price drop, but the
rotating-station strategy defeats its intended economic penalty.

If rotation is considered fun, acceptable infrastructure management, the delay
can intentionally permit it. In that case, model upkeep revenue under widespread
rotation and acknowledge the much weaker sink. Do not count on seven days of
player downtime as compensation.

If the requirement is that large amounts of operating power must actually pay for
their restored health, a station-local waiting period is insufficient when both
power and consumers can move. That requires a different pricing or health rule,
or an explicitly accepted limit on the economic ambition. I would avoid adding
player-migration lockouts solely to rescue this pricing mechanism without reviewing
their wider gameplay consequences.

This finding does not require changing the accepted player-driven audit system,
oldest-first shedding, collateral bounty, or bounded cascade behavior.

## Reproduction, coverage and limits

Ran **326 scenario rows**: 144 single-station price cases, 150 rotation cases,
four outgoing-allocation cases, four timer cases and 24 hypothetical downtime-cost
cases. All **55 model tests passed**, including 12 new tests and 5,000 seeded random
pricing transitions. The original and subsequent model tests remain in the suite.
JSON and CSV results were replayed and checked for identical contents.

Durations use the previous comparison convention of 15,500 blocks per model day.
At an exact five-second block, a calendar day is 17,280 blocks. This changes wall
time calibration, not the rotation mechanism or the relation between waiting and
service intervals. Candidate tariffs, collateral and duration values are not
approved production parameters.

Mixed-size stacked prepayments remain rejected by the inherited model; every
rotation purchases one day after the prior purchase expired. The demonstrated
bypass therefore does not depend on resolving that separate pricing policy.
No live chain transactions, real gas measurements, bank-failure simulation or
production code changes were made.

```sh
python3 -m unittest discover -s tools/substation_model -p 'test_*.py' -v
python3 tools/substation_model/discount_delay_experiments.py
python3 tools/substation_model/discount_delay_experiments.py --out build/substation-discount-delay-replay
```

- [Delay model](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/discount_delay_model.py)
- [Tests](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/test_discount_delay_model.py)
- [Experiment runner](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/discount_delay_experiments.py)
- [Results and source fingerprints](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-discount-delay-results-2026-09-25/results.json)

CSV files alongside the JSON contain the individual cases.
