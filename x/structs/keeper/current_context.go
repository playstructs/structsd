package keeper

import (
	"cmp"
	"context"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
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

	permissions             map[string]*PermissionsCache
	guildRankRegisters      map[string]*GuildRankRegisterCache

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


	// Transient combat state (nil outside attack handler, not committed)
	Attack *AttackContext

	// signerAddress is the authenticated address that signed the message being
	// handled (msg.Creator). It is the acting identity for every Layer 1
	// permission check and belongs to the operation, not to any entity: a
	// player may hold many addresses with different permission bits, and
	// PlayerCache instances are shared across every address of that player.
	// Write-once via setSigner so a later lookup of a caller-supplied address
	// can never redefine who is acting.
	signerAddress string

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

		permissions:            make(map[string]*PermissionsCache),
		guildRankRegisters:     make(map[string]*GuildRankRegisterCache),


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

// SignerAddress returns the authenticated address acting in this operation, or
// the empty string in block hooks and genesis where nothing signed.
func (cc *CurrentContext) SignerAddress() string {
	return cc.signerAddress
}

// setSigner records the acting address. Calling it twice with the same address
// is a no-op; a second, different address is a programming error, because it
// would mean the operation has two acting identities and permission checks
// could resolve against either.
func (cc *CurrentContext) setSigner(address string) error {
	if cc.signerAddress == address {
		return nil
	}
	if cc.signerAddress != "" {
		return types.NewAddressValidationError(address, "signer_already_set")
	}
	cc.signerAddress = address
	return nil
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

	commitCaches(cc.players)
	commitCaches(cc.addresses)
	commitCaches(cc.infusions)
	commitCaches(cc.allocations)
	commitCaches(cc.guilds)
	commitCaches(cc.guildMembershipApps)
	commitCaches(cc.fleets)
	commitCaches(cc.agreements)
	commitCaches(cc.planets)
	commitCaches(cc.providers)
	commitCaches(cc.structs)
	commitCaches(cc.substations)
	commitCaches(cc.reactors)
	commitCaches(cc.gridAttributes)
	commitCaches(cc.planetAttributes)
	commitCaches(cc.structAttributes)
	commitCaches(cc.permissions)
	commitCaches(cc.guildRankRegisters)

	cc.committed = true
}

// commitCaches commits every entry of a cache map in ascending key order.
//
// Go randomizes map iteration order per process, so ranging a cache map
// directly is only safe while every Commit writes to keys derived from its own
// map key. AddressCache.Commit does not: it allocates an auth account number
// from the account keeper's global sequence for any address that has none, so
// two addresses committed in map order get their numbers swapped from one node
// to the next. That is divergent auth state and a different app hash, reachable
// both from a genesis AddressList and from one transaction carrying two
// AddressRegister messages.
//
// Sorting is what makes the order a property of the data rather than of the
// runtime. Every caller must go through here; x/structs/keeper/arch_commit_test.go
// fails on a bare range over a cache map inside CommitAll, and on a cache map
// that CommitAll never commits at all.
func commitCaches[K cmp.Ordered, V interface{ Commit() }](caches map[K]V) {
	keys := make([]K, 0, len(caches))
	for key := range caches {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {
		caches[key].Commit()
	}
}
