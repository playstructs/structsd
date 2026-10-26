package types

import "strconv"

// AddressRegisterActivity is the domain separator for the address registration
// proof. It keeps these sign bytes from colliding with any other signed payload
// in the module.
const AddressRegisterActivity = "REGISTERADDRESS"

/* AddressRegisterProofInput builds the message an address key signs to prove it
 * consents to being absorbed into a player account.
 *
 * The proof is what authorizes AddressRegister to sweep the address's entire
 * spendable balance and move its delegations onto the player's primary address,
 * so it has to be good for exactly one registration. It used to be
 * "PLAYER<id>ADDRESS<addr>" — a pure function of two values that never change —
 * which made it a permanent bearer credential rather than a one-time consent:
 *
 *   - The only thing standing between an old proof and a second sweep was the
 *     address index, and AddressRevoke deletes exactly that. Revoke an address,
 *     let it receive funds again, resubmit the original proof, sweep. The player
 *     side can do this unaided, since PermissionCheck passes an owner on their
 *     own player object.
 *
 *   - Nothing bound the chain, so a proof signed here verified on any other
 *     Structs chain where the same player id existed.
 *
 * The chain id closes the second. The nonce closes the first, and it is stored
 * per address rather than per player because this is an offline credential: a
 * counter that unrelated activity could bump would expire a proof its signer had
 * already handed over, which is the failure mode proxyNonce carries and the
 * reason it is not reused here.
 */
func AddressRegisterProofInput(chainId string, playerId string, address string, nonce uint64) string {
	return "CHAIN" + chainId + "|" + AddressRegisterActivity +
		"PLAYER" + playerId +
		"ADDRESS" + address +
		"NONCE" + strconv.FormatUint(nonce, 10)
}
