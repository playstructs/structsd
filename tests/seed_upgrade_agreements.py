"""Create live legacy contracts after the released gameplay fixture tears its own down."""
import json


def created_object(transaction, kind, field):
    rows = []
    for event in transaction.get("events", []):
        if event["type"] == "structs.structs.Event" + kind:
            values = [item["value"] for item in event["attributes"] if item["key"] == field]
            if len(values) != 1:
                raise ValueError("ambiguous " + kind + " creation event")
            rows.append(json.loads(values[0]))
    if not rows or len({row["id"] for row in rows}) != 1:
        raise ValueError("expected one created " + kind)
    return rows[-1]


def seed_agreements(runner):
    if (runner.directory / "agreement-seed.json").exists():
        raise ValueError("agreement fixture already attempted; preserve its evidence")
    report = {"status": "seeding started; not verified", "transactions": [], "agreements": []}
    runner.save("agreement-seed.json", report)

    def send(label, *arguments, signer="alice"):
        delivered = runner.tx(*arguments, signer=signer)
        report["transactions"].append({"step": label, "transaction": delivered})
        runner.save("agreement-seed.json", report)
        return delivered

    alice = runner.cli(runner.old, "keys", "show", "alice", "--keyring-backend", "test")["address"]
    player = runner.cli(runner.old, "query", "structs", "address", alice)["playerId"]
    # The released fixture assigns all of Alice's player capacity to her main
    # station. Draw from that station's spare capacity, not from the now-full
    # player. Resolve and verify it rather than assuming an object ID.
    station_id = runner.cli(runner.old, "query", "structs", "player", player)["Player"].get("substationId")
    if not station_id:
        raise ValueError("legacy owner has no source substation")
    upstream = runner.cli(runner.old, "query", "structs", "substation", station_id)["Substation"]
    if upstream.get("id") != station_id or upstream.get("owner") != player:
        raise ValueError("legacy source substation is not owned by the fixture player")
    report["upstream_substation"] = upstream
    runner.save("agreement-seed.json", report)
    allocation = created_object(send("source_allocation", "structs", "allocation-create", station_id, "1000",
                                     "--controller", player, "--allocation-type", "static"), "Allocation", "allocation")
    station = created_object(send("source_substation", "structs", "substation-create", player, allocation["id"]),
                             "Substation", "substation")
    provider = created_object(send("provider", "structs", "provider-create", station["id"], "3ualpha", "open-market",
                                  "0.5", "0.25", "1", "1000", "1", "10000"), "Provider", "provider")
    report.update(source_allocation=allocation, substation=station, provider=provider)
    bob = runner.cli(runner.old, "keys", "show", "bob", "--keyring-backend", "test")["address"]
    for signer in ("player_2", "player_3"):
        address = runner.cli(runner.old, "keys", "show", signer, "--keyring-backend", "test")["address"]
        send("fund_" + signer, "bank", "send", bob, address, "200000ualpha", signer="bob")
    for index, (signer, duration, capacity) in enumerate((("player_2", 800, 2), ("player_3", 810, 1), ("player_2", 820, 3))):
        agreement = created_object(send("agreement_" + str(index), "structs", "agreement-open", provider["id"],
                                       str(duration), str(capacity), signer=signer), "Agreement", "agreement")
        if agreement["providerId"] != provider["id"] or int(agreement["capacity"]) != capacity or int(agreement["endBlock"]) - int(agreement["startBlock"]) != duration:
            raise ValueError("legacy contract does not match requested terms")
        report["agreements"].append(agreement)
        runner.save("agreement-seed.json", report)
    if len({row["id"] for row in report["agreements"]}) != 3:
        raise ValueError("legacy agreement identities are not distinct")
    report["status"] = "three live legacy agreements confirmed; upgrade verification pending"
    runner.save("agreement-seed.json", report)
