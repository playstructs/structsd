package types

// PlanetSectors lists the chain's assigned sectors. Assignment draws an index
// into this list, so its order is a consensus rule.
var PlanetSectors = [...]uint64{127, 999, 44}

// IsPlanetSector reports whether a sector is one of the chain's assigned sectors.
func IsPlanetSector(sector uint64) bool {
	for _, assigned := range PlanetSectors {
		if sector == assigned {
			return true
		}
	}
	return false
}
