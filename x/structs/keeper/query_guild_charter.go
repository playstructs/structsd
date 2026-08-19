package keeper

import (
	"context"

	"structs/x/structs/types"
)

/* GuildCharter reports the state of the global guild charter puzzle.
 *
 * The anchor is included because a proof is bound to it: a client that grinds
 * against a stale anchor produces a proof the handler will refuse, so it has to
 * be told which one it is working against rather than assuming the anchor is the
 * height it last saw a guild founded at.
 */
func (k Keeper) GuildCharter(goCtx context.Context, req *types.QueryGuildCharter) (*types.QueryGuildCharterResponse, error) {
	return &types.QueryGuildCharterResponse{
		Anchor:          k.CharterAnchor(goCtx),
		Age:             k.CharterAge(goCtx),
		Difficulty:      uint32(k.CharterDifficulty(goCtx)),
		DifficultyRange: k.GetParams(goCtx).CharterDifficultyRange(),
	}, nil
}
