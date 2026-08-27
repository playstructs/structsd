package structs

import (
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/keeper"
	"structs/x/structs/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {

	// =========================================================================
	// Phase 0: Pre-processing lookup maps
	// =========================================================================

	playerPlanetMap := make(map[string]string)
	for _, player := range genState.PlayerList {
		playerPlanetMap[player.Id] = player.PlanetId
	}

	structStatusMap := make(map[string]uint64)
	for _, attr := range genState.StructAttributeList {
		attrTypeId, ok := parseAttributeTypeId(attr.AttributeId)
		if !ok {
			continue
		}
		if types.StructAttributeType(attrTypeId) == types.StructAttributeType_status {
			structId := objectIdFromAttributeId(attr.AttributeId)
			structStatusMap[structId] = attr.Value
		}
	}

	allocationPowerMap := make(map[string]uint64)
	for _, attr := range genState.GridList {
		attrTypeId, ok := parseAttributeTypeId(attr.AttributeId)
		if !ok {
			continue
		}
		if types.GridAttributeType(attrTypeId) == types.GridAttributeType_power {
			objectId := objectIdFromAttributeId(attr.AttributeId)
			if strings.HasPrefix(objectId, fmt.Sprintf("%d-", types.ObjectType_allocation)) {
				allocationPowerMap[objectId] = attr.Value
			}
		}
	}

	// =========================================================================
	// Foundation: Direct keeper writes (no CC)
	// =========================================================================

	k.SetPort(ctx, genState.PortId)
	if k.ShouldBound(ctx, genState.PortId) {
		err := k.BindPort(ctx, genState.PortId)
		if err != nil {
			panic("could not claim port capability: " + err.Error())
		}
	}
	k.SetParams(ctx, genState.Params)

	/* Staking and genutil initialise before this module, so a genesis validator's
	 * reactor was already created by AfterValidatorCreated — and stamped with an
	 * eligibility height computed from params that did not exist yet, which means
	 * the production default rather than whatever the genesis file says. Restamp
	 * now that the params are real. Imported reactors are overwritten by
	 * GenesisImportReactor below, so a restored chain keeps its own clock.
	 */
	k.RestampReactorCharterEligibility(ctx)

	var structTypeTop uint64
	for _, elem := range types.CreateStructTypeGenesis() {
		if elem.Id > structTypeTop {
			structTypeTop = elem.Id
		}
		k.SetStructType(ctx, elem)
	}
	k.SetStructTypeCount(ctx, structTypeTop+1)

	// Entity counts derived from max index in each list.
	// Use advanceCount to avoid resetting counts that staking hooks
	// (ReactorInitialize) may have already advanced before this module's
	// InitGenesis runs.
	advanceCount := func(current uint64, genesis uint64) uint64 {
		if genesis > current {
			return genesis
		}
		return current
	}

	k.SetGuildCount(ctx, advanceCount(k.GetGuildCount(ctx), maxIndex(genState.GuildList, func(g types.Guild) uint64 { return g.Index })+1))

	/* The charter anchor cannot be rebuilt from anything else, so an unset one
	 * falls back to the current height rather than to zero. Zero is not a
	 * neutral default here: it would read as an age of the whole chain, which is
	 * the easiest point on the difficulty curve, and hand out a guild for one
	 * leading zero on the first block after import.
	 */
	if genState.GuildCharterAnchor == 0 {
		k.SetGuildCharterAnchor(ctx, uint64(ctx.BlockHeight()))
	} else {
		k.SetGuildCharterAnchor(ctx, genState.GuildCharterAnchor)
	}

	k.SetReactorCount(ctx, advanceCount(k.GetReactorCount(ctx), maxIndexFromId(genState.ReactorList, func(r types.Reactor) string { return r.Id })+1))
	k.SetSubstationCount(ctx, advanceCount(k.GetSubstationCount(ctx), maxIndexFromId(genState.SubstationList, func(s types.Substation) string { return s.Id })+1))
	k.SetPlayerCount(ctx, advanceCount(k.GetPlayerCount(ctx), maxIndex(genState.PlayerList, func(p types.Player) uint64 { return p.Index })+1))
	k.SetPlanetCount(ctx, advanceCount(k.GetPlanetCount(ctx), maxIndexFromId(genState.PlanetList, func(p types.Planet) string { return p.Id })+1))
	k.SetStructCount(ctx, advanceCount(k.GetStructCount(ctx), maxIndex(genState.StructList, func(s types.Struct) uint64 { return s.Index })+1))
	k.SetProviderCount(ctx, advanceCount(k.GetProviderCount(ctx), maxIndex(genState.ProviderList, func(p types.Provider) uint64 { return p.Index })+1))
	k.SetAllocationCount(ctx, advanceCount(k.GetAllocationCount(ctx), maxIndex(genState.AllocationList, func(a types.Allocation) uint64 { return a.Index })+1))

	// =========================================================================
	// Phase 1: CC population
	// =========================================================================

	cc := k.NewCurrentContext(ctx)

	// Guilds
	for _, guild := range genState.GuildList {
		cc.GenesisImportGuild(guild)
	}
	for _, app := range genState.GuildMembershipApplicationList {
		cc.GenesisImportGuildMembershipApplication(app)
	}

	// Reactors
	for _, reactor := range genState.ReactorList {
		cc.GenesisImportReactor(reactor)
	}

	// Substations
	for _, substation := range genState.SubstationList {
		cc.GenesisImportSubstation(substation)
	}

	// Players (after substations)
	for _, player := range genState.PlayerList {
		cc.GenesisImportPlayer(player)
	}

	// Providers
	for _, provider := range genState.ProviderList {
		cc.GenesisImportProvider(provider)
		k.IndexProviderPoolAddresses(ctx, provider.Id)
	}

	// Permissions
	for _, elem := range genState.PermissionList {
		cc.GenesisImportPermission([]byte(elem.PermissionId), types.Permission(elem.Value))
	}

	// Guild rank permissions (store only, no events; EventAllGenesis will emit)
	for _, rec := range genState.GuildRankPermissionList {
		if rec == nil {
			continue
		}
		k.SetGuildRankPermissionStoreOnly(ctx, rec.ObjectId, rec.GuildId, types.Permission(rec.Permissions), rec.Rank)
	}

	// Addresses
	for _, elem := range genState.AddressList {
		cc.GenesisImportAddress(elem.Address, elem.PlayerIndex)
	}

	// Address registration-proof nonces. A separate list from AddressList
	// because a revoked address keeps its nonce and loses its association, and
	// that is the row a restored chain must not forget: dropping it resets the
	// address to nonce 0 and makes its original proof replayable.
	for _, elem := range genState.AddressNonceList {
		cc.GenesisImportAddressNonce(elem.Address, elem.Nonce)
	}

	// Planets
	for _, planet := range genState.PlanetList {
		cc.GenesisImportPlanet(planet)
	}

	// Fleets (send all home)
	for _, fleet := range genState.FleetList {
		homePlanetId := playerPlanetMap[fleet.Owner]
		cc.GenesisImportFleet(fleet, homePlanetId)
	}

	// Structs
	for _, s := range genState.StructList {
		importedStatus := structStatusMap[s.Id]
		cc.GenesisImportStruct(s, importedStatus)
	}

	// Selective grid attribute import: player ore, proxyNonce, nonce only
	for _, attr := range genState.GridList {
		if isGenesisGridImportable(attr.AttributeId) {
			cc.SetGridAttribute(attr.AttributeId, attr.Value)
		}
	}

	// Non-reactor infusions
	for _, infusion := range genState.InfusionList {
		cc.GenesisImportInfusion(infusion)
	}

	// Allocations
	for _, allocation := range genState.AllocationList {
		importedPower := allocationPowerMap[allocation.Id]
		cc.GenesisImportAllocation(allocation, importedPower)
	}

	// Agreements
	for _, agreement := range genState.AgreementList {
		cc.GenesisImportAgreement(agreement)
	}

	/* Every provider leaves this function with a checkpoint clock standing at or
	 * after the genesis height.
	 *
	 * The exported value is restored by the grid import above and is the right
	 * answer for a height-preserving restart. A file that carries no checkpoint
	 * row - a hand-written genesis, or one exported before that attribute was
	 * imported at all - would otherwise start the provider at zero, and the
	 * first checkpoint would bill from block zero across the load these
	 * agreements just rebuilt. Stamping the genesis height instead says the
	 * provider has been paid up to the moment the chain starts, which is the
	 * only reading consistent with the agreements it is starting with.
	 */
	genesisHeight := uint64(ctx.BlockHeight())
	suppliedCheckpoints := make(map[string]bool, len(genState.ProviderList))
	for _, attr := range genState.GridList {
		attrTypeId, ok := parseAttributeTypeId(attr.AttributeId)
		if !ok {
			continue
		}
		if types.GridAttributeType(attrTypeId) == types.GridAttributeType_checkpointBlock {
			suppliedCheckpoints[objectIdFromAttributeId(attr.AttributeId)] = true
		}
	}

	for _, provider := range genState.ProviderList {
		if suppliedCheckpoints[provider.Id] {
			// The exported clock is the right answer and may legitimately sit
			// below the genesis height: a restart at H+1 carrying a checkpoint of
			// H owes exactly one block. Flooring it here would be the same bug in
			// the other direction, silently forgiving service already rendered.
			continue
		}
		cc.GetProvider(provider.Id).SetCheckpointBlock(genesisHeight)
	}

	// Reactor infusions (rebuild from staking delegations)
	for _, reactor := range genState.ReactorList {
		cc.GenesisImportReactorInfusions(reactor)
	}

	/* Pending grid cascades.
	 *
	 * The queue was exported but never imported, which was survivable only
	 * while GridCascade drained it to exhaustion every block - anything pending
	 * at export time had been enqueued and processed inside the same block, so
	 * an export could not catch one. Now that the cascade is bounded and carries
	 * work forward, a restore that dropped the queue would leave those objects
	 * over-subscribed with nothing left to schedule them.
	 *
	 * Re-appended in export order, which is sequence order, so the restored
	 * queue is the exported one.
	 */
	for _, objectId := range genState.GridCascadeQueue {
		if err := k.AppendGridCascadeQueue(ctx, objectId); err != nil {
			panic(err)
		}
	}

	// =========================================================================
	// Commit
	// =========================================================================

	cc.CommitAll()

	// Guild names are stored on guild records, while the normalized lookup is
	// derived state. Rebuild it on every import in list order; first wins is
	// deterministic and matches the upgrade migration's collision policy.
	k.ClearGuildNameIndex(ctx)
	claimedGuildNames := make(map[string]bool)
	for _, guild := range genState.GuildList {
		if guild.Name == "" {
			continue
		}
		normalized := types.NormalizeName(guild.Name)
		if claimedGuildNames[normalized] {
			if duplicate, found := k.GetGuild(ctx, guild.Id); found {
				duplicate.Name = ""
				k.SetGuild(ctx, duplicate)
			}
			continue
		}
		claimedGuildNames[normalized] = true
		k.SetGuildNameIndex(ctx, guild.Name, guild.Id)
	}

	// IBC core, transfer and bank initialize before structs in app_config.go.
	// Rebuild this derived protect-only index from their exported state so an
	// export/import restart cannot make existing voucher backing confiscatable.
	k.ProtectLegacyGuildEscrowBalances(ctx)

	// Struct defenders (after CC commit, so structs are in KV store)
	for _, elem := range genState.StructDefenderList {
		protected, found := k.GetStruct(ctx, elem.ProtectedStructId)
		if !found {
			continue
		}
		k.SetStructDefender(ctx, elem.ProtectedStructId, protected.Index, elem.DefendingStructId)
	}

	// Infusion maturity sweep queue (raw composite-key rows). Restored verbatim
	// so the EndBlocker continues processing UBD entries that were already in
	// flight when the export was taken. Absent in pre-v0.17.0 exports; the
	// keeper helper is a no-op on empty rows.
	for _, row := range genState.InfusionMaturitySweepQueue {
		k.ImportInfusionMaturitySweepRow(ctx, row)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.PortId = k.GetPort(ctx)

	genesis.AddressList = k.GetAllAddressExport(ctx)
	genesis.AddressNonceList = k.GetAllAddressProofNonceExport(ctx)

	genesis.AgreementList = k.GetAllAgreement(ctx)

	genesis.AllocationList = k.GetAllAllocation(ctx)
	genesis.AllocationCount = k.GetAllocationCount(ctx)

	genesis.InfusionList = k.GetAllInfusion(ctx)
	genesis.InfusionDestructionQueue = k.GetInfusionDestructionQueueExport(ctx)
	genesis.InfusionMaturitySweepQueue = k.GetInfusionMaturitySweepQueueExport(ctx)

	genesis.FleetList = k.GetAllFleet(ctx)

	genesis.GuildList = k.GetAllGuild(ctx)
	genesis.GuildCount = k.GetGuildCount(ctx)
	genesis.GuildMembershipApplicationList = k.GetAllGuildMembershipApplicationExport(ctx)
	genesis.GuildCharterAnchor, _ = k.GetGuildCharterAnchor(ctx)

	genesis.PlanetList = k.GetAllPlanet(ctx)
	genesis.PlanetCount = k.GetPlanetCount(ctx)
	genesis.PlanetAttributeList = k.GetAllPlanetAttributeExport(ctx)

	genesis.PlayerList = k.GetAllPlayer(ctx)
	genesis.PlayerCount = k.GetPlayerCount(ctx)

	genesis.ProviderList = k.GetAllProvider(ctx)
	genesis.ProviderCount = k.GetProviderCount(ctx)
	genesis.ProviderGuildAccessList = k.GetAllProviderGuildAccessExport(ctx)

	genesis.ReactorList = k.GetAllReactor(ctx)
	genesis.ReactorCount = k.GetReactorCount(ctx)

	genesis.StructList = k.GetAllStruct(ctx)
	genesis.StructCount = k.GetStructCount(ctx)
	genesis.StructAttributeList = k.GetAllStructAttributeExport(ctx)
	genesis.StructDefenderList = k.GetAllStructDefenderExport(ctx)
	genesis.StructDestructionQueue = k.GetStructDestructionQueueExport(ctx)

	genesis.SubstationList = k.GetAllSubstation(ctx)
	genesis.SubstationCount = k.GetSubstationCount(ctx)

	genesis.GridList = k.GetAllGridExport(ctx)
	genesis.GridCascadeQueue = k.GetGridCascadeQueueExport(ctx)

	genesis.PermissionList = k.GetAllPermissionExport(ctx)
	genesis.GuildRankPermissionList = k.GetAllGuildRankPermissionExport(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}

func parseAttributeTypeId(attributeId string) (uint64, bool) {
	parts := strings.SplitN(attributeId, "-", 2)
	if len(parts) == 0 {
		return 0, false
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func objectIdFromAttributeId(attributeId string) string {
	parts := strings.SplitN(attributeId, "-", 3)
	if len(parts) < 3 {
		return ""
	}
	return parts[1] + "-" + parts[2]
}

func isGenesisGridImportable(attributeId string) bool {
	attrTypeId, ok := parseAttributeTypeId(attributeId)
	if !ok {
		return false
	}
	switch types.GridAttributeType(attrTypeId) {
	case types.GridAttributeType_proxyNonce, types.GridAttributeType_nonce:
		return true
	case types.GridAttributeType_ore:
		objectId := objectIdFromAttributeId(attributeId)
		return strings.HasPrefix(objectId, fmt.Sprintf("%d-", types.ObjectType_player))
	case types.GridAttributeType_checkpointBlock:
		/* A provider's checkpoint block is the one grid attribute that is not
		 * derivable from the objects around it. Load and capacity are rebuilt by
		 * the import itself, so admitting them would double-count; the checkpoint
		 * is a clock, and nothing else on the export says where it stood.
		 *
		 * Losing it is not a cosmetic drift. Checkpoint() bills
		 * (currentBlock - checkpointBlock) * rate * aggregate load, and an unset
		 * attribute reads as zero, so the first checkpoint after a restore bills
		 * the whole height of the chain against the full reconstructed load.
		 * SweepRevenue clamps that to what the pool holds rather than failing, so
		 * the visible outcome is every consumer's collateral swept into the
		 * provider's earnings pool in one transaction.
		 */
		objectId := objectIdFromAttributeId(attributeId)
		return strings.HasPrefix(objectId, fmt.Sprintf("%d-", types.ObjectType_provider))
	default:
		return false
	}
}

func indexFromId(id string) uint64 {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) < 2 {
		return 0
	}
	idx, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	return idx
}

func maxIndex[T any](list []T, indexFn func(T) uint64) uint64 {
	var max uint64
	for _, item := range list {
		if idx := indexFn(item); idx > max {
			max = idx
		}
	}
	return max
}

func maxIndexFromId[T any](list []T, idFn func(T) string) uint64 {
	var max uint64
	for _, item := range list {
		if idx := indexFromId(idFn(item)); idx > max {
			max = idx
		}
	}
	return max
}
