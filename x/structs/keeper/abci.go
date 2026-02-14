package keeper

import (
	"context"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func (k *Keeper) BeginBlocker(ctx context.Context) {

    k.logger.Debug("Begin Block Processes")

    k.EmitEventTime(ctx)

    k.EventAllGenesis(ctx)

    k.StructSweepDestroyed(ctx)

    k.logger.Debug("Begin Block Complete")
}

// Called every block, update validator set
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	k.logger.Debug("End Block Processes")
    cc := k.NewCurrentContext(ctx)
    defer cc.CommitAll()

	cc.AgreementExpirations()

	/* Cascade all the possible failures across the grid
	 *
	 * This will mean that there will be some cases in which
	 * devices have one last block of power before shutting down
	 * but I think that's ok. We'll see how it goes in practice.
	 */
	cc.GridCascade()

    cc.ProcessInfusionDestructionQueue()

    k.logger.Debug("End Block Complete")

	return []abci.ValidatorUpdate{}, nil
}

func (k Keeper) EmitEventTime(ctx context.Context) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventTime{&types.EventTimeDetail{BlockHeight: ctxSDK.BlockHeight(), BlockTime: ctxSDK.HeaderInfo().Time.UTC() }})
}

func (k *Keeper) EventAllGenesis(ctx context.Context) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)

    if ctxSDK.BlockHeight() > 1 { return }

    k.logger.Info("Spewing Genesis Events into the Indexer")

	// Player
    players := k.GetAllPlayer(ctx)
    for _, player := range players {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlayer{Player: &player})
    }

	// Address
    addresses := k.GetAllAddressExport(ctx)
    for _, address := range addresses {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAddressAssociation{&types.AddressAssociation{Address: address.Address, PlayerIndex: address.PlayerIndex, RegistrationStatus: types.RegistrationStatus_approved}})
	}

	// Permissions
    permissions := k.GetAllPermissionExport(ctx)
    for _, permission := range permissions {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPermission{&types.PermissionRecord{PermissionId: permission.PermissionId, Value: permission.Value}})
    }

	// Grid Attributes
	grids := k.GetAllGridExport(ctx)
    for _, grid := range grids {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGrid{&types.GridRecord{AttributeId: grid.AttributeId, Value: grid.Value}})
    }

	// Reactor
    reactors := k.GetAllReactor(ctx)
    for _, reactor := range reactors {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventReactor{Reactor: &reactor})
    }

	// Infusion
    infusions := k.GetAllInfusion(ctx)
    for _, infusion := range infusions {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventInfusion{Infusion: &infusion})
    }

	// Struct Type
    structTypes := k.GetAllStructType(ctx)
    for _, structType := range structTypes {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStructType{StructType: &structType})
    }

    // Allocation
    allocations := k.GetAllAllocation(ctx)
    for _, allocation := range allocations {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAllocation{Allocation: &allocation})
    }

    // Agreement
    agreements := k.GetAllAgreement(ctx)
    for _, agreement := range agreements {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventAgreement{Agreement: &agreement})
    }

    // Fleet
    fleets := k.GetAllFleet(ctx)
    for _, fleet := range fleets {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventFleet{Fleet: &fleet})
    }

    // Guild (+ bank addresses)
    guilds := k.GetAllGuild(ctx)
    for _, guild := range guilds {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuild{Guild: &guild})
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildBankAddress{
            &types.EventGuildBankAddressDetail{
                GuildId:            guild.Id,
                BankCollateralPool: authtypes.NewModuleAddress(types.GuildBankCollateralPool + guild.Id).String(),
                BankTokenPool:      types.ModuleName,
            },
        })
    }


    // Planet
    planets := k.GetAllPlanet(ctx)
    for _, planet := range planets {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlanet{Planet: &planet})
    }

    // Planet attributes
    planetAttrs := k.GetAllPlanetAttributeExport(ctx)
    for _, attr := range planetAttrs {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventPlanetAttribute{attr})
    }

    // Provider (+ addresses)
    providers := k.GetAllProvider(ctx)
    for _, provider := range providers {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProvider{Provider: &provider})
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProviderAddress{
            &types.EventProviderAddressDetail{
                ProviderId:     provider.Id,
                CollateralPool: authtypes.NewModuleAddress(types.ProviderCollateralPool + provider.Id).String(),
                EarningPool:    authtypes.NewModuleAddress(types.ProviderEarningsPool + provider.Id).String(),
            },
        })
    }

    // Provider guild access
    providerAccess := k.GetAllProviderGuildAccessExport(ctx)
    for _, entry := range providerAccess {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProviderGrantGuild{
            &types.EventProviderGrantGuildDetail{ProviderId: entry.ProviderId, GuildId: entry.GuildId},
        })
    }

    // Struct
    structs := k.GetAllStruct(ctx)
    for _, s := range structs {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStruct{Structure: &s})
    }

    // Struct attributes
    structAttrs := k.GetAllStructAttributeExport(ctx)
    for _, attr := range structAttrs {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStructAttribute{attr})
    }

    // Struct defenders
    defenders := k.GetAllStructDefenderExport(ctx)
    for _, d := range defenders {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventStructDefender{StructDefender: d})
    }

    // Substation
    substations := k.GetAllSubstation(ctx)
    for _, s := range substations {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventSubstation{Substation: &s})
    }

    // Guild membership applications
    apps := k.GetAllGuildMembershipApplicationExport(ctx)
    for _, app := range apps {
        _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildMembershipApplication{GuildMembershipApplication: &app})
    }

}