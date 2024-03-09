package main

import (
	"os"

    sdkmath "cosmossdk.io/math"
    sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"structs/app"
	"structs/cmd/structsd/cmd"

)

func main() {
    sdk.DefaultPowerReduction = sdkmath.NewIntFromUint64(10)

	rootCmd, _ := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
