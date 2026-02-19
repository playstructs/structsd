package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// =============================================================================
// Committable Interface
// =============================================================================

// Committable represents any cache that can track and persist changes.
// All complex *Cache types must implement this interface.
type Committable interface {
	// ID returns the unique identifier for this cache
	ID() string

	// IsChanged returns true if any mutations occurred
	IsChanged() bool

	// Commit persists all changes to the KV store
	Commit()
}

// =============================================================================
// CurrentContext
// =============================================================================

// CurrentContext holds all state for a single operation (transaction or block hook).
// It provides entity cache deduplication - the same entity is only loaded once
// per operation, and all accessors return the same cache instance.
//
// Usage in message handlers:
//
//	func (k msgServer) SomeHandler(goCtx context.Context, msg *types.MsgSome) (...) {
//	    ctx := sdk.UnwrapSDKContext(goCtx)
//	    cc := k.NewCurrentContext(ctx)
//	    defer cc.CommitAll()
//	    // ... use cc.GetStruct(), cc.GetPlayer(), etc.
//	}
//
// Usage in ABCI hooks:
//
//	func (k Keeper) EndBlocker(ctx context.Context) error {
//	    cc := k.NewCurrentContext(ctx)
//	    cc.ProcessSomething()
//	    cc.CommitAll()
//	    return nil
//	}
type CurrentContext struct {
	ctx context.Context
	k   *Keeper



    // Actually Implemented Shit (AIS)
    addresses       map[string]*AddressCache
  	gridAttributes   map[string]*GridAttributeCache
  	structAttributes map[string]*StructAttributeCache
  	planetAttributes map[string]*PlanetAttributeCache

	permissions     map[string]*PermissionsCache

	players             map[string]*PlayerCache
	fleets              map[uint64]*FleetCache

	agreements          map[string]*AgreementCache
	guilds              map[string]*GuildCache
	guildMembershipApps map[string]*GuildMembershipApplicationCache
	infusions           map[string]*InfusionCache
	planets             map[string]*PlanetCache

	// Complex entity caches (Committable, tracked in pendingCommits)




	providers           map[string]*ProviderCache
	structs             map[string]*StructCache
	substations         map[string]*SubstationCache

	// Write-through attribute caches (read cache + immediate write to store)


	// Lightweight caches (committed directly by CommitAll)

	allocations     map[string]*AllocationCache

	reactors        map[string]*ReactorCache
	structTypes     map[uint64]*StructTypeCache // read-only, never committed


	// State flags
	committed bool
}

// NewCurrentContext creates a fresh context for an operation.
func (k *Keeper) NewCurrentContext(ctx context.Context) *CurrentContext {
	return &CurrentContext{
		ctx: ctx,
		k:   k,

        // Actually Implemented Shit (AIS)
		addresses:       make(map[string]*AddressCache),

		gridAttributes:   make(map[string]*GridAttributeCache),
		structAttributes: make(map[string]*StructAttributeCache),
		planetAttributes: make(map[string]*PlanetAttributeCache),

		permissions:     make(map[string]*PermissionsCache),

		players:             make(map[string]*PlayerCache),
		fleets:              make(map[uint64]*FleetCache),

		agreements:          make(map[string]*AgreementCache),

		guilds:              make(map[string]*GuildCache),
		guildMembershipApps: make(map[string]*GuildMembershipApplicationCache),
		infusions:           make(map[string]*InfusionCache),
		planets:             make(map[string]*PlanetCache),


		// Complex entity caches


		providers:           make(map[string]*ProviderCache),
		structs:             make(map[string]*StructCache),
		substations:         make(map[string]*SubstationCache),



		// Lightweight caches

		allocations:     make(map[string]*AllocationCache),

		reactors:        make(map[string]*ReactorCache),
		structTypes:     make(map[uint64]*StructTypeCache),


	}
}

// =============================================================================
// Context Accessors
// =============================================================================

// Context returns the underlying sdk.Context
func (cc *CurrentContext) Context() sdk.Context {
	return sdk.UnwrapSDKContext(cc.ctx)
}

// Keeper returns the keeper reference
func (cc *CurrentContext) Keeper() *Keeper {
	return cc.k
}

// =============================================================================
// Commit and Lifecycle
// =============================================================================

// CommitAll persists all changes from all accessed caches.
// This should be called at the end of the operation (typically via defer).
func (cc *CurrentContext) CommitAll() {
	if cc.committed {
		cc.k.logger.Warn("CurrentContext.CommitAll called multiple times")
		return
	}


	// Actually Implemented Shit (AIS)

	for _, addressCache := range cc.addresses {
        addressCache.Commit()
	}

	for _, allocationCache := range cc.allocations {
        allocationCache.Commit()
	}

    for _, playerCache := range cc.players {
        playerCache.Commit()
    }


    for _, guildCache := range cc.guilds {
        guildCache.Commit()
    }

    for _, guildMembershipApp := range cc.guildMembershipApps {
        guildMembershipApp.Commit()
    }

    for _, fleetCache := range cc.fleets {
        fleetCache.Commit()
    }

    for _, infusionCache := range cc.infusions {
        infusionCache.Commit()
    }

	for _, permissionsCache := range cc.permissions {
	    permissionsCache.Commit()
	}

    for _, gridAttributeCache := range cc.gridAttributes {
        gridAttributeCache.Commit()
    }

    for _, structAttributeCache := range cc.structAttributes {
        structAttributeCache.Commit()
    }

    for _, planetAttributeCache := range cc.planetAttributes {
        planetAttributeCache.Commit()
    }

    for _, agreementCache := range cc.agreements {
        agreementCache.Commit()
    }

    for _, planetCache := range cc.planets {
        planetCache.Commit()
    }

    for _, providerCache := range cc.providers {
        providerCache.Commit()
    }

    for _, structCache := range cc.structs {
        structCache.Commit()
    }

    for _, substationCache := range cc.substations {
        substationCache.Commit()
    }

    for _, reactorCache := range cc.reactors {
        reactorCache.Commit()
    }

	cc.committed = true
    /*
	TODO
	cc.k.logger.Debug("CurrentContext committed",
		"entity_cache_count", 0,
	)
	*/
}
