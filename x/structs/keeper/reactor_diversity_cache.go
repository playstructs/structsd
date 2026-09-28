package keeper

import (
	"math"
	"math/big"
	"sort"

	"structs/x/structs/types"
)

type ReactorDiversityPositionCache struct {
	CC                 *CurrentContext
	OwnerID, ReactorID string
	Power              *big.Int
	original           string
	Changed            bool
}

func (c *ReactorDiversityPositionCache) ID() string      { return c.OwnerID + "/" + c.ReactorID }
func (c *ReactorDiversityPositionCache) IsChanged() bool { return c.Changed }
func (c *ReactorDiversityPositionCache) Commit() {
	if c.Changed && c.Power.String() != c.original {
		c.CC.k.setReactorDiversityPosition(c.CC.ctx, types.ReactorDiversityPosition{OwnerId: c.OwnerID, ReactorId: c.ReactorID, Power: c.Power.String()})
		c.original = c.Power.String()
	}
	c.Changed = false
}

type OwnerReactorDiversityCache struct {
	CC                  *CurrentContext
	OwnerID             string
	Power, SquaredPower *big.Int
	AppliedBonus        uint64
	original            types.OwnerReactorDiversity
	Changed, Dirty      bool
}

func (c *OwnerReactorDiversityCache) ID() string      { return c.OwnerID }
func (c *OwnerReactorDiversityCache) IsChanged() bool { return c.Changed }
func (c *OwnerReactorDiversityCache) GetState() types.OwnerReactorDiversity {
	return types.OwnerReactorDiversity{OwnerId: c.OwnerID, Power: c.Power.String(), SquaredPower: c.SquaredPower.String(), AppliedBonus: c.AppliedBonus}
}
func (c *OwnerReactorDiversityCache) Commit() {
	if c.Dirty {
		panic("reactor diversity must be refreshed before commit")
	}
	row := c.GetState()
	if c.Changed && (row.Power != c.original.Power || row.SquaredPower != c.original.SquaredPower || row.AppliedBonus != c.original.AppliedBonus) {
		c.CC.k.setOwnerReactorDiversity(c.CC.ctx, row)
		c.original = row
	}
	c.Changed = false
}

func (cc *CurrentContext) GetOwnerReactorDiversity(owner string) *OwnerReactorDiversityCache {
	if c, found := cc.ownerReactorDiversity[owner]; found {
		return c
	}
	row, found := cc.k.GetOwnerReactorDiversity(cc.ctx, owner)
	if !found {
		row = types.OwnerReactorDiversity{OwnerId: owner, Power: "0", SquaredPower: "0"}
	}
	if row.OwnerId != owner {
		panic("reactor diversity owner key mismatch")
	}
	c := &OwnerReactorDiversityCache{CC: cc, OwnerID: owner, Power: mustDiversityInteger(row.Power), SquaredPower: mustDiversityInteger(row.SquaredPower), AppliedBonus: row.AppliedBonus, original: row}
	cc.ownerReactorDiversity[owner] = c
	return c
}

func (cc *CurrentContext) GetReactorDiversityPosition(owner, reactor string) *ReactorDiversityPositionCache {
	key := owner + "/" + reactor
	if c, found := cc.reactorDiversityPositions[key]; found {
		return c
	}
	row, found := cc.k.GetReactorDiversityPosition(cc.ctx, owner, reactor)
	if !found {
		row = types.ReactorDiversityPosition{OwnerId: owner, ReactorId: reactor, Power: "0"}
	}
	if row.OwnerId != owner || row.ReactorId != reactor {
		panic("reactor diversity position key mismatch")
	}
	c := &ReactorDiversityPositionCache{CC: cc, OwnerID: owner, ReactorID: reactor, Power: mustDiversityInteger(row.Power), original: row.Power}
	cc.reactorDiversityPositions[key] = c
	return c
}

// ChangeReactorDiversityPosition replaces one infusion's post-commission share
// inside its reactor aggregate. Square the aggregate, never individual rows.
// Call only after the base-capacity mutation, so a refresh sees the final base.
func (cc *CurrentContext) ChangeReactorDiversityPosition(owner, reactor string, old, next uint64) {
	if old == next {
		return
	}
	p := cc.GetReactorDiversityPosition(owner, reactor)
	s := cc.GetOwnerReactorDiversity(owner)
	before := new(big.Int).Set(p.Power)
	if before.Cmp(new(big.Int).SetUint64(old)) < 0 {
		panic("reactor diversity position underflow")
	}
	p.Power.Sub(p.Power, new(big.Int).SetUint64(old))
	p.Power.Add(p.Power, new(big.Int).SetUint64(next))
	s.Power.Sub(s.Power, before)
	s.Power.Add(s.Power, p.Power)
	s.SquaredPower.Sub(s.SquaredPower, new(big.Int).Mul(before, before))
	s.SquaredPower.Add(s.SquaredPower, new(big.Int).Mul(p.Power, p.Power))
	if s.Power.Sign() < 0 || s.SquaredPower.Sign() < 0 {
		panic("reactor diversity total underflow")
	}
	p.Changed, s.Changed, s.Dirty = true, true, true
}

// prepareOwnerDiversityBaseChange temporarily removes the applied bonus before
// a base delta. Otherwise adding base power could saturate the grid while the
// old bonus still occupies space, irreversibly losing genuine base capacity.
// The returned capacity is the total before the change, for final propagation.
func (cc *CurrentContext) prepareOwnerDiversityBaseChange(owner string) uint64 {
	s := cc.GetOwnerReactorDiversity(owner)
	cc.RefreshOwnerReactorDiversity(owner)
	key := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner)
	before := cc.getGridAttribute(key)
	if before < s.AppliedBonus {
		panic("applied reactor diversity exceeds owner capacity")
	}
	if s.AppliedBonus != 0 {
		cc.SetGridAttribute(key, before-s.AppliedBonus)
	}
	s.AppliedBonus = 0
	s.Changed, s.Dirty = true, false
	return before
}

// ApplyOwnerInfusionCapacityDelta installs a base contribution and its portfolio
// delta together. An empty reactor denotes a generator, whose power is excluded
// from diversity but can change how much bonus fits beside the base capacity.
func (cc *CurrentContext) ApplyOwnerInfusionCapacityDelta(owner, reactor string, old, next uint64) {
	if old == next {
		return
	}
	before := cc.prepareOwnerDiversityBaseChange(owner)
	key := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner)
	cc.SetGridAttributeDelta(key, old, next)
	if reactor != "" {
		cc.ChangeReactorDiversityPosition(owner, reactor, old, next)
	}
	cc.GetOwnerReactorDiversity(owner).Dirty = true
	cc.refreshOwnerReactorDiversity(owner, false)
	cc.propagateOwnerCapacityChange(owner, before, cc.getGridAttribute(key))
}

// RefreshOwnerReactorDiversity consumes dirty totals before a dependent read.
// Theoretical power stays arbitrary precision. Only the portion fitting beside
// base capacity is installed, preserving the grid's existing uint64 ceiling.
func (cc *CurrentContext) RefreshOwnerReactorDiversity(owner string) {
	cc.refreshOwnerReactorDiversity(owner, true)
}

func (cc *CurrentContext) refreshOwnerReactorDiversity(owner string, propagate bool) {
	s := cc.ownerReactorDiversity[owner]
	if s == nil || !s.Dirty {
		return
	}
	if cc.reactorDiversityParams == nil {
		p := cc.k.GetParams(cc.ctx).WithReactorDefaults()
		cc.reactorDiversityParams = &p
	}
	p := cc.reactorDiversityParams
	bonus, err := types.CalculateReactorDiversityBonus(s.Power, s.SquaredPower, p.DiversityMaxBonusBps, p.DiversityTargetExtraReactors)
	if err != nil {
		panic(err)
	}
	key := GetGridAttributeIDByObjectId(types.GridAttributeType_capacity, owner)
	before := cc.getGridAttribute(key)
	if before < s.AppliedBonus {
		panic("applied reactor diversity exceeds owner capacity")
	}
	base := before - s.AppliedBonus
	room := math.MaxUint64 - base
	if bonus.Cmp(new(big.Int).SetUint64(room)) > 0 {
		bonus.SetUint64(room)
	}
	s.AppliedBonus = bonus.Uint64() // bounded above before narrowing
	s.Dirty, s.Changed = false, true
	after := base + s.AppliedBonus
	if before != after {
		cc.SetGridAttribute(key, after)
		if propagate {
			cc.propagateOwnerCapacityChange(owner, before, after)
		}
	}
}

func (cc *CurrentContext) refreshDirtyReactorDiversity() {
	owners := make([]string, 0, len(cc.ownerReactorDiversity))
	for owner := range cc.ownerReactorDiversity {
		owners = append(owners, owner)
	}
	sort.Strings(owners)
	for _, owner := range owners {
		cc.RefreshOwnerReactorDiversity(owner)
	}
}

func (cc *CurrentContext) propagateOwnerCapacityChange(owner string, before, after uint64) {
	if before == after {
		return
	}
	id, found := cc.k.GetAutoResizeAllocationBySource(cc.ctx, owner)
	if found {
		found = cc.AutoResizeAllocation(id, after)
		if !found {
			cc.k.ClearAutoResizeAllocationBySource(cc.ctx, owner)
		}
	}
	if !found && after < before {
		cc.k.AppendGridCascadeQueue(cc.ctx, owner)
	}
}
