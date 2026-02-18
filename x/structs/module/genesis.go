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
	}
	for _, elem := range genState.ProviderGuildAccessList {
		k.ProviderGrantGuild(ctx, elem.ProviderId, elem.GuildId)
	}

	// Permissions
	for _, elem := range genState.PermissionList {
		cc.GenesisImportPermission([]byte(elem.PermissionId), types.Permission(elem.Value))
	}

	// Addresses
	for _, elem := range genState.AddressList {
		cc.GenesisImportAddress(elem.Address, elem.PlayerIndex)
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

	// Reactor infusions (rebuild from staking delegations)
	for _, reactor := range genState.ReactorList {
		cc.GenesisImportReactorInfusions(reactor)
	}


	// =========================================================================
	// Commit
	// =========================================================================

	cc.CommitAll()

	// Struct defenders (after CC commit, so structs are in KV store)
	for _, elem := range genState.StructDefenderList {
		protected, found := k.GetStruct(ctx, elem.ProtectedStructId)
		if !found {
			continue
		}
		k.SetStructDefender(ctx, elem.ProtectedStructId, protected.Index, elem.DefendingStructId)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.PortId = k.GetPort(ctx)

	genesis.AddressList = k.GetAllAddressExport(ctx)

	genesis.AgreementList = k.GetAllAgreement(ctx)

	genesis.AllocationList = k.GetAllAllocation(ctx)
	genesis.AllocationCount = k.GetAllocationCount(ctx)

	genesis.InfusionList = k.GetAllInfusion(ctx)
	genesis.InfusionDestructionQueue = k.GetInfusionDestructionQueueExport(ctx)

	genesis.FleetList = k.GetAllFleet(ctx)

	genesis.GuildList = k.GetAllGuild(ctx)
	genesis.GuildCount = k.GetGuildCount(ctx)
	genesis.GuildMembershipApplicationList = k.GetAllGuildMembershipApplicationExport(ctx)

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
