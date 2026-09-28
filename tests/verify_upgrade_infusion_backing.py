"""Check post-upgrade reactor positions independently of matching event payloads."""
from verify_upgrade_bank import amount
from verify_upgrade_staking import shares


def unique(rows, key, label):
    result = {}
    for row in rows:
        identity = key(row)
        if identity in result:
            raise ValueError(f"duplicate {label}: {identity}")
        result[identity] = row
    return result


def verify_infusion_backing(after, vaults):
    state = after["app_state"]
    structs, staking = state["structs"], state["staking"]
    owners = set(unique(structs["playerList"], lambda row: row["id"], "player"))
    owners |= set(unique(structs["guildList"], lambda row: row["id"], "guild"))
    by_address = {}
    for row in structs["addressList"]:
        index = amount(row["playerIndex"])
        if index == 0:
            continue
        owner = f"1-{index}"
        if owner not in owners or row["address"] in by_address:
            raise ValueError("invalid or duplicate player address association")
        by_address[row["address"]] = owner
    if len(set(structs["vaultOwnerList"])) != len(structs["vaultOwnerList"]):
        raise ValueError("duplicate provisioned vault owner")
    if set(vaults) != set(structs["vaultOwnerList"]):
        raise ValueError("vault event mappings must cover all provisioned owners")
    for owner, address in vaults.items():
        if owner not in owners or not isinstance(address, str) or not address or address in by_address:
            raise ValueError("invalid or conflicting vault association")
        by_address[address] = owner
    validators = unique(staking["validators"], lambda row: row["operator_address"], "validator")
    reactors = unique(structs["reactorList"], lambda row: row["id"], "reactor")
    by_validator = unique(structs["reactorList"], lambda row: row["validator"], "reactor validator")
    pair = lambda row: (row["delegator_address"], row["validator_address"])
    delegations = unique(staking["delegations"], pair, "delegation")
    unbondings = unique(staking["unbonding_delegations"], pair, "unbonding")

    def backing(key):
        validator = validators.get(key[1])
        if validator is None:
            raise ValueError(f"missing validator for staking position: {key}")
        fuel = 0
        if key in delegations:
            delegated, total = shares(delegations[key]["shares"]), shares(validator["delegator_shares"])
            if not 0 < delegated <= total:
                raise ValueError("invalid delegation share backing")
            fuel = delegated * amount(validator["tokens"]) // total
        defusing = sum(amount(entry["balance"]) for entry in unbondings.get(key, {}).get("entries", []))
        if max(fuel, defusing) >= 2**64:
            raise ValueError("staking position exceeds infusion uint64 range")
        return fuel, defusing

    differences, seen = [], set()
    for row in structs["infusionList"]:
        if not row["destinationId"].startswith("3-"):
            continue
        reactor = reactors.get(row["destinationId"])
        if reactor is None:
            raise ValueError("infusion references a missing reactor")
        key = (row["address"], reactor["validator"])
        if key in seen:
            raise ValueError("duplicate reactor infusion")
        seen.add(key)
        label = f"{row['destinationId']}/{row['address']}"
        owner = by_address.get(row["address"])
        recorded_owner = row.get("ownerId") or row.get("playerId")
        if owner is None or recorded_owner != owner or row.get("playerId", "") not in ("", owner):
            differences.append(f"infusion ownership disagrees with address association: {label}")
        if owner and owner.startswith("0-") and row.get("playerId", ""):
            differences.append(f"guild infusion carries a player id: {label}")
        if row.get("destinationType") != "reactor":
            differences.append(f"infusion destination type disagrees: {label}")
        fuel, defusing = backing(key)
        if (amount(row["fuel"]), amount(row.get("defusing", "0"))) != (fuel, defusing):
            differences.append(f"infusion fuel/defusing differs from staking backing: {label}")
        validator = validators[key[1]]
        if type(validator["jailed"]) is not bool or validator["status"] not in (
                "BOND_STATUS_BONDED", "BOND_STATUS_UNBONDING", "BOND_STATUS_UNBONDED"):
            raise ValueError("invalid validator eligibility state")
        enabled = validator["status"] == "BOND_STATUS_BONDED" and not validator["jailed"]
        if amount(row["ratio"]) != int(enabled) or amount(row["power"]) != (fuel if enabled else 0):
            differences.append(f"infusion power/eligibility disagrees: {label}")
        if shares(row["commission"]) > 10**18:
            differences.append(f"invalid infusion commission: {label}")
    for key in delegations.keys() | unbondings.keys():
        if key[0] not in by_address:
            continue  # Unassociated staking assets are outside the player migration.
        fuel, defusing = backing(key)
        if (fuel or defusing) and (key not in seen or key[1] not in by_validator):
            differences.append(f"associated staking position has no reactor infusion: {key}")
    return {"differences": differences, "reactor_infusions": len(seen),
        "scope": "post-upgrade live ownership, mathematical floor of staking share backing, unbonding balances, flat power and validator eligibility; not aggregate grid topology or pending queue liveness"}
