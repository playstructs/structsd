"""Reconstruct migrated ordinary-stake counters from the old staking export.

Vesting sources retain their own counters. Ordinary sources instead contribute
merged, redeemable shares and pending refunds, excluding retained self-stake.
This is a boundary comparison, not a queue-liveness or spendability verifier.
"""
from collections import defaultdict

from upgrade_inventory import payload
from verify_upgrade_bank import amount
from verify_upgrade_staking import shares


PRECISION = 10**18
DECIMAL_LIMIT = 2**256 * PRECISION
BASE = "/cosmos.auth.v1beta1.BaseAccount"


def redemption_value(delegated, total, tokens):
    """Exact positive SDK 0.53 TokensFromShares(...).TruncateInt().

    LegacyDec.Quo first divides at 36 places, then rounds to 18 with ties to
    even. A rational floor differs immediately below an integer boundary.
    """
    if not 0 < delegated <= total < DECIMAL_LIMIT or not 0 <= tokens < 2**256:
        raise ValueError("invalid ordinary-stake share backing")
    numerator = delegated * tokens
    if numerator >= DECIMAL_LIMIT:
        raise ValueError("ordinary-stake SDK multiplication overflows")
    wide = numerator * PRECISION**2 // total
    rounded, remainder = divmod(wide, PRECISION)
    if remainder * 2 > PRECISION or (remainder * 2 == PRECISION and rounded % 2):
        rounded += 1
    if rounded >= DECIMAL_LIMIT:
        raise ValueError("ordinary-stake SDK division overflows")
    return rounded // PRECISION


def original_liquid_delegated(before, owner, vault, accounts, owners):
    state = before["app_state"]["staking"]
    for field in ("validators", "delegations", "unbonding_delegations"):
        if not isinstance(state.get(field), list):
            raise ValueError(f"missing ordinary-stake export inventory: {field}")
    denom = state["params"]["bond_denom"]
    if not isinstance(denom, str) or not denom:
        raise ValueError("missing staking denomination")
    ordinary = {address for address, subject in owners.items()
        if subject == owner and accounts[address]["@type"] == BASE}
    if vault in accounts:
        if accounts[vault]["@type"] != BASE:
            raise ValueError("migration destination already has a non-base account")
        ordinary.add(vault)
    validators = {}
    for row in state["validators"]:
        key = row["operator_address"]
        if key in validators:
            raise ValueError("duplicate ordinary-stake validator")
        validators[key] = row

    def contributes(delegator, validator):
        # An existing vault has no signer association and is not a retained
        # operator source. Its already-held ordinary stake still contributes.
        return delegator in ordinary and not (
            delegator in owners and payload(delegator) == payload(validator))

    merged, total_refunds = defaultdict(int), 0
    seen = set()
    for row in state["delegations"]:
        key = (row["delegator_address"], row["validator_address"])
        if key in seen:
            raise ValueError("duplicate ordinary-stake delegation")
        seen.add(key)
        if contributes(*key):
            held = shares(row["shares"])
            if held <= 0:
                raise ValueError("ordinary delegation has no shares")
            merged[key[1]] += held
    seen = set()
    for row in state["unbonding_delegations"]:
        key = (row["delegator_address"], row["validator_address"])
        if key in seen:
            raise ValueError("duplicate ordinary-stake unbonding")
        seen.add(key)
        if contributes(*key):
            if not isinstance(row.get("entries"), list) or not row["entries"]:
                raise ValueError("ordinary unbonding has no entries")
            ids = set()
            for entry in row["entries"]:
                identity = entry["unbonding_id"]
                if identity in ids:
                    raise ValueError("duplicate ordinary unbonding entry")
                ids.add(identity)
                total_refunds += amount(entry["balance"])
    # Redelegation entries already have a destination delegation above. Adding
    # their initial balances or shares again would double-count the same stake.
    value = total_refunds
    for validator, held in merged.items():
        if validator not in validators:
            raise ValueError("ordinary stake references a missing validator")
        row = validators[validator]
        value += redemption_value(held, shares(row["delegator_shares"]), amount(row["tokens"]))
    if value >= 2**256:
        raise ValueError("ordinary-stake counter exceeds SDK coin range")
    return [{"denom": denom, "amount": str(value)}] if value else []
