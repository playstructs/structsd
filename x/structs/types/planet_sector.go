package types

// IsPlanetSector reports whether a sector is one of the chain's assigned sectors.
func IsPlanetSector(sector uint64) bool {
	return sector == 127 || sector == 999 || sector == 44
}
