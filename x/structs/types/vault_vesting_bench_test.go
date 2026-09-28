package types_test

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"
	"structs/x/structs/types"
)

// Isolates schedule CPU/allocation costs. This does not measure store gas or
// establish a production account-size bound; real-app gas coverage is separate.
func BenchmarkVaultVestingSchedules(b *testing.B) {
	for _, count := range []int{1, 100, 1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			sources := make([]vestingexported.VestingAccount, count)
			for i := range sources {
				address := make([]byte, 20)
				binary.BigEndian.PutUint64(address[12:], uint64(i+1))
				source, err := vestingtypes.NewDelayedVestingAccount(authtypes.NewBaseAccountWithAddress(address), vestingCoins(100), 200)
				require.NoError(b, err)
				sources[i] = source
			}
			address, err := types.VaultAddress("1-1")
			require.NoError(b, err)
			vault, err := types.NewVaultVestingAccount("1-1", authtypes.NewBaseAccountWithAddress(address), nil, sources...)
			require.NoError(b, err)
			registry := cdctypes.NewInterfaceRegistry()
			authtypes.RegisterInterfaces(registry)
			vestingtypes.RegisterInterfaces(registry)
			types.RegisterInterfaces(registry)
			cdc := codec.NewProtoCodec(registry)
			wire, err := cdc.MarshalInterface(vault)
			require.NoError(b, err)
			at := time.Unix(100, 0)
			b.Run("decode", func(b *testing.B) {
				b.ReportAllocs()
				b.ReportMetric(float64(len(wire)), "account-bytes")
				for i := 0; i < b.N; i++ {
					var account sdk.AccountI
					if err := cdc.UnmarshalInterface(wire, &account); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("locked", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					vault.LockedCoins(at)
				}
			})
			b.Run("delegate-refund", func(b *testing.B) {
				one, balance := vestingCoins(1), vestingCoins(int64(count)*100)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					vault.TrackDelegation(at, balance, one)
					vault.TrackUndelegation(one)
				}
			})
		})
	}
}
