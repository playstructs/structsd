# Structs integration tests

Bash scripts run against a live `structsd` chain. Prerequisites: chain running locally, `alice` and `bob` keys in keyring.

## Scripts

- **test_chain.sh** — Full lifecycle (players, guilds, allocations, exploration, structs, combat). Use `--skip-mining` to avoid slow PoW. Use `--resume-from N` to resume from a phase.
- **test_fleet_movement.sh** — Fleet linked-list and range-based combat.
- **test_permissions.sh** — Permissions: owner defaults, grant/revoke/set, positive/negative checks, guild rank, (optional) player rank change, ordering, and object-deletion cleanup.

## Running test_permissions.sh

```bash
# Full run (creates guild, substation, extra players, then runs all permission tests)
./tests/test_permissions.sh

# Use existing state (e.g. after test_chain.sh); export PLAYER_1_ID, GUILD_ID, SUBSTATION_ID, PLAYER_2_ID, PLAYER_3_ID
./tests/test_permissions.sh --skip-setup
```

Not safe to run concurrently; one test runner per chain. Script exits with non-zero if any assertion fails (for CI).
