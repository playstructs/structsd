package keeper

import (
	"context"
	"encoding/binary"
	"hash/fnv"
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// planetSector follows the existing block-AppHash seeded math/rand system.
// The stable planet-ID offset gives worlds created in one block separate draws
// without consuming the owner's combat nonce. FNV-1a and the sector order are
// consensus rules. An absent/short AppHash uses seed zero, as existing draws do.
func planetSector(ctx context.Context, planetID string) uint64 {
	hash := sdk.UnwrapSDKContext(ctx).BlockHeader().AppHash
	var seed int64
	if len(hash) >= 8 {
		seed = int64(binary.BigEndian.Uint64(hash[:8]))
	}
	offset := fnv.New64a()
	_, _ = offset.Write([]byte(planetID))
	orb := rand.New(rand.NewSource(seed + int64(offset.Sum64())))
	sectors := [...]uint64{127, 999, 44}
	return sectors[orb.Intn(len(sectors))]
}

func (cache *PlanetCache) GetSector() uint64 {
	return cache.GetPlanet().Sector
}

// BackfillPlanetSectors is upgrade-only: scan all worlds, including completed
// and ownerless ones, and publish normal planet events for changed records.
// Each cache is committed and discarded separately. Repeating the migration
// preserves assigned sectors and emits no redundant planet updates.
func (k Keeper) BackfillPlanetSectors(ctx context.Context) {
	for _, row := range k.GetAllPlanet(ctx) {
		if row.Sector != 0 {
			continue
		}
		cc := k.NewCurrentContext(ctx)
		planet := cc.GetPlanet(row.Id)
		// The migration already loaded this record; don't read it again on commit.
		planet.Planet, planet.PlanetLoaded, planet.storedOwner = row, true, row.Owner
		planet.Planet.Sector = planetSector(ctx, row.Id)
		planet.Changed = true
		cc.CommitAll()
	}
}
