package types

import (
	//sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
)

// TODO This isn't going to work because this doesn't have access to k?


func CreateBaseStructCache(structId string) (StructCache, error) {
    return StructCache{
        StructID: structId,
        StatusAttributeId: GetStructAttributeIDByObjectId(StructAttributeType_status, structId),
    }, nil
}

func (cache *StructCache) GetStruct(ctx context.Context) (Struct, error) {
    if (!cache.StructureLoaded) {
        return true
    }

    return false
}


func (cache *StructCache) IsBuilt(ctx context.Context) bool {
    return false
}

func (cache *StructCache) IsOnline(ctx context.Context) bool {

	return false
}
