# Connection-count and outgoing-capacity upkeep: model results

September 25, 2026. **This pricing basis fixes the original incoming-supply trick.**
Connected players and outgoing commitments do not disappear just because incoming
power is temporarily removed. The model confirms that the upkeep bill stays put.

There are still separate ways to manipulate those billable quantities at purchase
time: disconnect players, or temporarily shrink a dynamic outgoing allocation.
The formula is a better description of the services being provided, but a single
purchase-time snapshot does not ensure those services pay for their restored health.

## Formula and scope

For `d` whole purchased days:

```text
price = d × (base fee + connected-player count × per-player fee)
      + tiered outgoing-capacity charge for d days
```

The default example uses **1 Alpha base**, **0.05 Alpha per connected player**, and
the existing candidate marginal tariff applied to **total outgoing allocation
power**. The sweep also tests base fees of 0 and 10 Alpha, per-player fees of 0.01
and 1 Alpha, and one- and seven-day purchases. None is an approved production price.

Count all connected player objects, not addresses or only online players. Sum all
outgoing commitments before applying the tier tariff. Do not re-tier each allocation;
splitting an allocation into smaller rows should not change the station's bill.

This exploration changes the billing basis of the earlier whole-day upkeep design.
It retains global health restoration, health decay, prepaid expiry and the seven-day
prepayment limit. The temporary bonus is conservatively limited to the supply present
at purchase. This is not the separate fixed-price/no-bonus alternative, and it does
not include the previous discount-delay rule.

No incoming supply tariff is added on top. Outgoing capacity is the station's
exported service; the connection count prices its directly served players. This
separates those two uses of power. Real serial routes can still pay at each station
that provides another service; the formula does not promise a chain-wide single charge.

## Main results

Illustrative station: 100 MW incoming power and 100 connected players. Where shown,
the station also supplies a **dynamic 60 MW outgoing allocation**.

| Action before one-day upkeep | No outgoing allocations | With 60 MW dynamic output |
| --- | ---: | ---: |
| Normal purchase | 6 Alpha | 255.36 Alpha |
| Temporarily cut input to 1 kW | 6 Alpha | 255.36 Alpha |
| Disconnect players, buy upkeep, reconnect | 1 Alpha | 250.36 Alpha |
| Shrink dynamic output to 1 kW, buy upkeep, restore | 6 Alpha; no output to shrink | 6.0095 Alpha |
| Disconnect players and shrink dynamic output | 1 Alpha | 1.0095 Alpha |

The same player count and outgoing capacity are restored afterward in every case.
The model checks allocation identity as well as power. It does not assume that
recreating a destroyed agreement is equivalent to retaining it.

Ran **990 scenario rows** and **69 passing model tests**, including 14 new tests
and 1,000 exact rational-arithmetic price checks. Previous alternative-model tests
remain in the suite. Replay outputs match byte for byte. These are research-model
results, not live-chain exploit reproductions or approval of the parameters.

## 1. Your original reasoning is confirmed

Cutting accepted input from 100 MW to 1 kW leaves the 100 players connected and
the 60 MW commitment outstanding. The quote remains **255.36 Alpha**. Restoring
the input provides no billing discount.

Likewise, expanding input without changing the customer count or outgoing load
does not raise the bill. This is intentional under the proposed basis, not a lost
charge: direct service is priced per player rather than per watt.

The pricing sweep covers 0–1,000 connected players, outgoing capacity from zero
through 60 MW, and incoming supply from 1 kW through 1,000 MW. For a fixed customer
count and outgoing load, changing input never changes the quote.

Operational capacity still falls when input falls. The conservative temporary-bonus
control also means buying while input is small may leave less boosted output after
reconnection, even though the price was unchanged. Neither effect makes the price
cheaper. This distinguishes the pricing test from a promise of identical service
under every rearrangement.

## 2. Connection count can be deliberately changed

With no outgoing allocations, disconnect all 100 players, purchase upkeep for the
empty station at the **1 Alpha base fee**, then reconnect them. The modeled station
has its original count and restored health, but paid 1 Alpha instead of 6.

No second station is necessary for this variant. Migration to a standby station
also works, but temporary disconnection is enough if the authorized sequence can
complete before a material period of lost service.

At 1,000 players and the same price parameters, the ordinary cost is 51 Alpha and
the empty-station cost is 1 Alpha. At a zero base fee, a station with neither
players nor outgoing capacity can restore health for free in this formula.

### Permissions and workload matter

This is not permissionless control of someone else's guild or players. Current
connection changes are permission-gated. Reconnection checks station authority and
the target player's connection authority. A station owner cannot be assumed to
hold the latter for every player.

The model counts **two connection mutations per temporarily removed player**.
For 100 players, that is 200 mutations; for 1,000, 2,000. These counts are not gas
estimates and do not establish that an entire guild can fit in one transaction or
block. Smaller controlled groups or cooperating players require less coordination.
One player's case needs only its disconnect, upkeep and reconnect operations.

The relevant current paths are
[SubstationPlayerDisconnect](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/msg_server_substation_player_disconnect.go:9),
[SubstationPlayerConnect](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/msg_server_substation_player_connect.go:9)
and [MigrateSubstation](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/player_cache.go:464).
The model maintains unique player-to-station associations and updates both station
counts on migration. It does not simply overwrite a count to obtain a discount.

An authorization boolean in the research model represents already-held authority;
it is not a reimplementation or verification of the chain's permission resolver.
Real message execution, ante charges, gas and inclusion timing remain unmeasured.

## 3. Dynamic outgoing allocations can also change temporarily

The existing [AllocationUpdate handler](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/msg_server_allocation_update.go:9)
calls [SetDynamicPower](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/allocation_cache.go:218).
That method permits a positive smaller power for dynamic allocations and checks
available capacity on restoration. It does not destroy and recreate the allocation.

The model performs:

```text
60 MW dynamic allocation → 1 kW
purchase upkeep using that smaller outgoing load
restore the same allocation to 60 MW
run the queued cascade
```

Restoration is checked against the proposed durable admission budget, not merely
raw input. A fresh 100 MW station has sufficient safe capacity for 60 MW, so the
restoration passes. With 100 connected players, the bill falls from **255.36 Alpha
to 6.0095 Alpha**, saving about **97.65%**.

A downstream station with its own 30 MW static outgoing allocation retains that
allocation when upstream power is restored before cascade. If cascade runs while
the input is low, the downstream allocation is destroyed and is not recreated by
the restoration. Both orders are tested.

This follows the current deferred-shedding behavior: the queue checks current load
and capacity when processed. It does not remember every transient shortage as an
obligation to destroy something later. See
[GridCascade](/Users/abstract/Documents/Structs/Code/structs/x/structs/keeper/grid_context.go:243).

**This result is specifically about dynamic allocations.** The model rejects direct
resizing of static, automated and provider-agreement allocations. Static deletion
and recreation, automated resizing triggered by other mutations, and provider
capacity-change settlement need their own tests. Their associated costs should not
be silently assumed away. In particular, this experiment does not demonstrate free
provider-agreement repricing or lost consumer collateral.

## 4. The economic shape improves in some useful ways

The connected-player component follows the total number of directly served players,
so dividing an honest network among more stations does not multiply that component.
For 100 players, the default per-player component stays 5 Alpha:

| Parallel stations, same total players and supply | Total one-day quote | Player component |
| --- | ---: | ---: |
| 1 | 6 Alpha | 5 Alpha |
| 2 | 7 Alpha | 5 Alpha |
| 10 | 15 Alpha | 5 Alpha |
| 100 | 105 Alpha | 5 Alpha |

The base fee still rewards consolidation, but now only that component grows with
station count. This is a more nuanced structure than charging every station one
identical repair fee regardless of its customers.

On the other hand, direct energy abundance remains possible without higher upkeep:
with one player and no outgoing allocations, the quote remains **1.05 Alpha** as
input grows from 1 MW to 1,000 MW. Fresh unboosted power per player grows from
700 kW to 700 MW. That is an intentional per-player subsidy to higher power usage,
not a model error. It should be accepted or rejected as an economic choice.

For outgoing power, tiering the aggregate rather than each allocation makes simple
fragmentation price-neutral within a station. Cross-station routing retains the
ordinary conversion losses and may repeat base/tier charges.

The model does not rerun a calibrated global market or claim that this fee schedule
stabilizes issuance. Player counts, actual demand, safe service intervals and the
prevalence of deliberate disconnection would determine realized upkeep revenue.

## 5. A base fee limits the bypass, but does not restore proportional pricing

If all customers can be temporarily disconnected, the player-based price reduces
to the base fee at purchase time. Making that fee larger reduces the percentage
saving, but also makes empty and small stations more expensive. If dynamic output
can also be minimized, its capacity component can approach zero at the same time.

With both actions in the illustrative 100-player/60-MW case, the payment is
**1.0095 Alpha**, instead of 255.36. This is not a claim that all owners can arrange
both actions cheaply; it identifies what the formula allows under that authority.

There is also a startup version: purchase upkeep while a station is empty, then
connect players. For a seven-day purchase with 100 eventual players, paying while
empty costs **7 Alpha** instead of **42 Alpha**. No temporary disconnection of
already present players is required.

As a diagnostic, the experiment prices a flat one-day per-player reconnection fee.
It closes the one-day example arithmetically but not the seven-day one: 7 Alpha for
empty upkeep plus 5 Alpha of connection fees is 12 Alpha, still below 42. That fee
was not added to the model's actual connection rules and is not a recommended final
policy. It demonstrates why any new guard must cover the full benefit being sold,
including carried prepaid time, rather than just one chosen example.

## Recommendation

**Keep this as a promising definition of what upkeep charges for. Do not treat the
purchase-time count and load alone as sufficient protection.** It succeeds against
incoming-supply manipulation and gives a clearer separation between direct-player
service and exported power.

The remaining design choice is whether cheaply changing those actual services is
acceptable gameplay. If it is, expect coordinated groups and mutable allocations
to pay less than the headline tariff. If it is not, the design must account for
service added after repair as well as service present at repair, or select a pricing
rule that does not promise that proportionality.

That does not automatically require historical billing logs or a manual operating
cap. But a connection/increase charge, commitment rule or health-entitlement rule
would be an additional mechanic that must be specified and tested. Reintroducing
the previous station-local discount delay alone would retain the standby-rotation
problem already demonstrated. I have not quietly added any such guard here.

## Implementation boundaries and reproduction

Production pricing can use the existing stored connection count and outgoing load.
It should not scan all players or allocations. The Python grid sums local edges
for convenience; that is not a proposed consensus implementation.

Before adopting this design, real-app tests should cover multi-message ordering,
signing-key permissions, bank rollback, customer migration gas, dynamic restoration,
deferred downstream cascade, static deletion and provider settlement separately.
Whole-station health, prepaid growth, same-block proof anchors and creation pricing
also remain policy surfaces. Existing guild, vault, reactor and migration work was
not changed by this exploration.

Time uses the previous 15,500-block model-day convention. At exactly five seconds
per block, a calendar day would be 17,280 blocks. Main price comparisons occur in
one logical transaction window and do not depend on that wall-clock convention.

```sh
python3 -m unittest discover -s tools/substation_model -p 'test_*.py' -v
python3 tools/substation_model/service_pricing_experiments.py
python3 tools/substation_model/service_pricing_experiments.py --out build/substation-service-pricing-replay
```

- [Model](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/service_pricing_model.py)
- [Tests](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/test_service_pricing_model.py)
- [Experiment runner](/Users/abstract/Documents/Structs/Code/structs/tools/substation_model/service_pricing_experiments.py)
- [Results, parameters and source fingerprints](/Users/abstract/Documents/Structs/Code/structs/build/substation-model/substation-service-pricing-results-2026-09-25/results.json)

CSV files alongside the JSON contain all 990 scenario rows. No production chain
code, permissions, transactions or migrations were modified.
