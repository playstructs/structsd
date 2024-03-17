package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"structs/app"
	"structs/cmd/structsd/cmd"

    sdk "github.com/cosmos/cosmos-sdk/types"
    "cosmossdk.io/math"

)

func main() {
    sdk.DefaultPowerReduction = math.NewIntFromUint64(10)

	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
