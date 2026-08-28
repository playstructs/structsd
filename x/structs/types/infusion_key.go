package types

import (
	"strings"
)

/* An infusion is keyed by its destination and the address that funded it,
 * joined with a hyphen: "<objectType>-<index>-<address>".
 *
 * That only round-trips because of two properties neither half declares. A
 * destination id carries exactly one hyphen - GetObjectID writes "%d-%d" - and a
 * bech32 address carries none, its charset having no hyphen in it. Split the
 * joined string on "-" and you get exactly three parts back.
 *
 * The rule lives here rather than in the keeper because GenesisState.Validate
 * has to apply the same one and cannot import the keeper. A record whose key
 * does not round-trip is not merely odd: the keeper resolves an infusion by
 * splitting that key, and a split that does not yield three parts produces a
 * cache with no CurrentContext behind it, which panics on the first method call.
 */

// BuildInfusionKey joins a destination and an address into the stored key.
// (InfusionKey is already the store prefix constant.)
func BuildInfusionKey(destinationId string, address string) string {
	return destinationId + "-" + address
}

/* ParseInfusionKey splits a stored infusion key back into its parts.
 *
 * Returns an error rather than partial results for anything that does not split
 * into exactly three components, which is the shape every key written by a
 * runtime path has.
 */
func ParseInfusionKey(infusionKey string) (destinationId string, address string, err error) {
	parts := strings.Split(infusionKey, "-")
	if len(parts) != 3 {
		return "", "", NewObjectNotFoundError("infusion", infusionKey)
	}

	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", NewObjectNotFoundError("infusion", infusionKey)
	}

	return parts[0] + "-" + parts[1], parts[2], nil
}

// ValidInfusionParts reports whether a destination and address join into a key
// that parses back to the same pair. Genesis import assigns these fields
// directly, so this is where a record that could never be resolved is caught.
func ValidInfusionParts(destinationId string, address string) bool {
	parsedDestination, parsedAddress, err := ParseInfusionKey(BuildInfusionKey(destinationId, address))
	return err == nil && parsedDestination == destinationId && parsedAddress == address
}
