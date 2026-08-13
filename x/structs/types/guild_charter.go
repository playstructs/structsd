package types

import (
	"strconv"
)

/* Guild charter preimages.
 *
 * Both strings live here rather than in the handler so the CLI grinder and the
 * consensus check cannot drift: a mismatch of one byte makes every proof anyone
 * mines invalid, and there is no error that would say so.
 */

/* GuildCharterWorkInput builds the proof-of-work preimage for founding a guild.
 *
 * Shaped like the four existing proofs (subject, activity tag, start block,
 * NONCE, nonce) but binding two subjects rather than one. The existing proofs
 * name no player because their reward is object-bound: a build proof stolen out
 * of the mempool merely finishes somebody else's struct. A charter's reward
 * accrues to a named owner, so both ends have to be bound. Without the solver a
 * pool member could be relabelled and robbed of credit; without the founder a
 * mempool observer could redirect the guild to themselves.
 *
 * The separator is load-bearing, not decoration. Both sides are player ids with
 * the same type prefix, so bare concatenation would let ("1-4", "21-7") and
 * ("1-42", "1-7") produce the same string and share a solution. It follows the
 * raid preimage's fleet@planet precedent.
 *
 * The anchor is what makes a solution single-use: founding a guild moves it, so
 * every nonce mined against the old value dies at once, including the one just
 * spent. Block heights never repeat, so a proof can never come back.
 */
func GuildCharterWorkInput(solverPlayerId string, founderPlayerId string, anchor uint64, nonce string) string {
	return solverPlayerId + "@" + founderPlayerId + GuildCharterActivity +
		strconv.FormatUint(anchor, 10) + "NONCE" + nonce
}

/* GuildCharterConsentInput builds the message a founder signs offline to let
 * somebody else found their guild.
 *
 * Collected before the grind starts, because a charter is a race: if the winning
 * nonce had to travel to the founder for a signature, the pool with the most
 * attentive leader would beat the pool with the most hashpower.
 *
 * It binds the whole shape of the guild, not just the founder. The signature is
 * a bearer token shared with an entire pool, and an unbound one would let any
 * holder name their own substation as the new guild's entry point, which is
 * every future member's power supply. Binding reactor, substation and endpoint
 * means a holder can only produce exactly the guild the founder described.
 *
 * Replay protection is the anchor again, with no counter, and it holds only
 * because spending a consent founds a guild *by proof*, which moves the anchor.
 * That is a condition rather than a given: the reactor entitlement path founds a
 * guild without moving the anchor, so GuildCreate refuses a third-party founding
 * there instead of leaving the signature live. Anything else added later that
 * accepts a consent has to move the anchor or be refused the same way.
 *
 * Deliberately not the proxyNonce grid attribute used by
 * GuildMembershipJoinProxy: that is bumped when somebody proxy-joins the
 * founder, so a consent that has to survive weeks of mining would die for an
 * unrelated reason.
 */
func GuildCharterConsentInput(founderPlayerId string, reactorId string, entrySubstationId string, endpoint string, anchor uint64) string {
	return GuildCharterActivity + founderPlayerId +
		"REACTOR" + reactorId +
		"SUBSTATION" + entrySubstationId +
		"ENDPOINT" + endpoint +
		"ANCHOR" + strconv.FormatUint(anchor, 10)
}
