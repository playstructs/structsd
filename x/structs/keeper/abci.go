package keeper

import (
	"context"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func (k *Keeper) BeginBlocker(ctx context.Context) {
	ctx = sdk.UnwrapSDKContext(ctx).WithValue(deferPermissionCleanupKey{}, true)

    k.logger.Debug("Begin Block Processes")

    k.EmitEventTime(ctx)
    k.EventAllGenesis(ctx)
    k.processReactorCommissionRevisionReplay(ctx)

    k.StructSweepDestroyed(ctx)

    k.logger.Debug("Begin Block Complete")
}

// Called every block, update validator set
func (k *Keeper) EndBlocker(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	ctx = sdk.UnwrapSDKContext(ctx).WithValue(deferPermissionCleanupKey{}, true)
	k.logger.Debug("End Block Processes")
	if err := k.EvaluateReactorStakeCap(ctx); err != nil { return nil, err }

	// Commit each phase before the next one opens a CurrentContext. Grid
	// cascade and infusion reconciliation can update the same attributes, and
	// overlapping contexts would calculate and commit from stale snapshots.
	preSweepCC := k.NewCurrentContext(ctx)
	preSweepCC.AgreementExpirations()

    /* Process the bounded grid cascade. A capacity loss reaches downstream
     * allocations only as their source is serviced. Deep graphs or queue
     * backlogs can therefore retain downstream capacity for multiple blocks;
     * this phase does not guarantee shutdown throughout the graph this block.
	 */
	preSweepCC.GridCascade()
	preSweepCC.CommitAll()

	/* Reconcile any infusions whose unbonding-delegation entries matured this
	 * block. Cosmos SDK v0.53 silently completes UBDs in x/staking's EndBlocker
	 * (which runs before ours) and fires no hook, so we walk the structs-side
	 * maturity queue to clear stale Defusing values. Empty infusions get
	 * enqueued for destruction by the reconciliation helper, then processed in
	 * the immediately-following ProcessInfusionDestructionQueue.
	 */
	k.ProcessInfusionMaturitySweep(ctx)

	destructionCC := k.NewCurrentContext(ctx)
	destructionCC.ProcessInfusionDestructionQueue()
	destructionCC.CommitAll()

	/* Reconcile infusions on reactors that were slashed or changed bonded status. Both kinds share one rotating queue and one per-block budget.
	 *
	 * BeforeValidatorSlashed only queues the reactor: it runs in BeginBlock with
	 * no gas meter over a delegator set the delegators size, so doing the work
	 * there made one slash cost whatever they had built up. It also fires before
	 * staking applies the slash, so running here has the second advantage that
	 * live staking state is already the answer and nothing has to be carried.
	 * Restoration runs after this block's cascade: a pending capacity loss is
	 * not undone by promising that a later batch will restore the source.
	 */
	k.ProcessReactorSlashReconcileQueue(ctx)

	// Generator transitions have a separate bounded budget and commit before
	// later phases. They may enqueue owner/downstream cascades for next block.
	k.ProcessGeneratorPowerQueue(ctx)

	/* Finish removing permissions granted on objects that have been destroyed.
	 *
	 * The row count is chosen by whoever owned the object, and an agreement's
	 * expiry height is chosen by the consumer who opened it, so doing all of it
	 * at destruction time put an attacker-sized amount of work in one block. The
	 * remainder is inert - a permission is only consulted after its object has
	 * been loaded, and a destroyed object cannot be - so this is collection, not
	 * settlement, and it can take as many blocks as it needs.
	 */
	k.ProcessPermissionCleanupQueue(ctx)



	k.logger.Debug("End Block Complete")

	return []abci.ValidatorUpdate{}, nil
}

// ProcessInfusionMaturitySweep drains a bounded batch of InfusionMaturitySweepQueue
// rows whose CompletionTime <= the current block time and reconciles each affected
// infusion against live Cosmos staking state. Native refunds are not delayed by
// this bookkeeping budget. This is the structs-module
// substitute for the missing AfterUnbondingComplete hook in Cosmos SDK v0.53.
func (k *Keeper) ProcessInfusionMaturitySweep(ctx context.Context) {
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	k.processInfusionMaturitySweep(ctx, cc)
}

func (k *Keeper) processInfusionMaturitySweep(ctx context.Context, cc *CurrentContext) {
    ctxSDK := sdk.UnwrapSDKContext(ctx)
    blockTime := ctxSDK.HeaderInfo().Time

    matured := k.DequeueMatureInfusionSweeps(ctx, blockTime)
    if len(matured) == 0 {
        return
    }

    k.logger.Info("Infusion maturity sweep", "count", len(matured), "blockTime", blockTime)

    for _, infusionId := range matured {
        infusion, found := k.GetInfusionByID(ctx, infusionId)
        if !found {
            // Infusion was already destroyed (e.g. by a reactor-cancel-defusion
            // followed by destruction queue processing). Nothing to reconcile.
            continue
        }
        if infusion.DestinationType != types.ObjectType_reactor {
            continue
        }

        reactor, reactorFound := k.GetReactor(ctx, infusion.DestinationId)
        if !reactorFound {
            k.logger.Warn("Infusion maturity sweep: reactor missing", "infusionId", infusionId, "reactorId", infusion.DestinationId)
            continue
        }

        playerAddress, err := sdk.AccAddressFromBech32(infusion.Address)
        if err != nil {
            k.logger.Warn("Infusion maturity sweep: invalid delegator address", "infusionId", infusionId, "address", infusion.Address, "error", err)
            continue
        }
        validatorAddress, err := sdk.ValAddressFromBech32(reactor.Validator)
        if err != nil {
            k.logger.Warn("Infusion maturity sweep: invalid validator address", "infusionId", infusionId, "validator", reactor.Validator, "error", err)
            continue
        }

        // A queued source can have been revoked or moved since enqueueing.
        // This bookkeeping pass must not auto-register it as a new player.
        if err := k.reconcileExistingInfusionForDelegation(ctx, cc, playerAddress, validatorAddress); err != nil {
            // The row was consumed by the dequeue. Put it back after this
            // block's time so Fuel and Defusing are retried rather than left
            // stale with nothing scheduled to touch them again.
            k.logger.Error("Infusion maturity sweep: reconcile failed", "infusionId", infusionId, "error", err)
            k.EnqueueInfusionMaturitySweep(ctx, blockTime.Add(time.Nanosecond), infusionId)
        }
    }
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

	// Guild rank permissions
	guildRankPerms := k.GetAllGuildRankPermissionExport(ctx)
	for _, rec := range guildRankPerms {
		if rec != nil {
			_ = ctxSDK.EventManager().EmitTypedEvent(&types.EventGuildRankPermission{GuildRankPermissionRecord: rec})
		}
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




    if err := k.EmitReactorStakeCap(ctx); err != nil { panic(err) }
    if err := k.EmitReactorCommissionRevisionSnapshot(ctx); err != nil {
        panic(err)
    }
    if err := k.EmitOwnerReactorDiversitySnapshot(ctx); err != nil { panic(err) }

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


    // Vault associations follow their player/guild owner events. Replay does
    // not create accounts or advertise these addresses as signing keys.
    if err := k.EmitVaultAddressEvents(ctx); err != nil {
        k.logger.Error("Cannot replay vault address associations", "error", err)
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
