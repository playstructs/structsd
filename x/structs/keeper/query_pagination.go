package keeper

import (
	"github.com/cosmos/cosmos-sdk/types/query"

	"structs/x/structs/types"
)

/* boundedPagination caps what a single query may ask a node to do.
 *
 * query.Paginate takes the request at its word. A Limit is a uint64 and is not
 * capped, so one request can ask for every row in a collection and have all of
 * them decoded and retained; Offset+Limit is added without an overflow check;
 * and a request with no Limit turns CountTotal on, which walks the entire prefix
 * after the page has been collected. None of these queries require
 * authorization.
 *
 * Two clamps:
 *
 * Limit is capped at QueryPageLimitMaximum. A client that wants more follows
 * next_key, which is the pagination the SDK recommends anyway.
 *
 * CountTotal is forced off, because it is the expensive half: it scans the whole
 * prefix however small the page, so capping the page alone would not bound the
 * work. Responses therefore report total as 0. That is not a new contract - the
 * SDK already ignores count_total whenever a key is set, so every client already
 * paginating by key sees 0 today - but it is a visible change for a client that
 * was reading it, and it is called out in the upgrade notes.
 *
 * Callers pass req.Pagination straight through; a nil request is left nil so the
 * SDK's own defaults apply, and those are already bounded (limit 100). The
 * default's CountTotal is handled by the explicit page request built below.
 */
func boundedPagination(pagination *query.PageRequest) *query.PageRequest {
	if pagination == nil {
		return &query.PageRequest{
			Limit:      types.QueryPageLimitDefault,
			CountTotal: false,
		}
	}

	limit := pagination.Limit
	if limit == 0 {
		limit = types.QueryPageLimitDefault
	}
	if limit > types.QueryPageLimitMaximum {
		limit = types.QueryPageLimitMaximum
	}

	return &query.PageRequest{
		Key:        pagination.Key,
		Offset:     pagination.Offset,
		Limit:      limit,
		CountTotal: false,
		Reverse:    pagination.Reverse,
	}
}
