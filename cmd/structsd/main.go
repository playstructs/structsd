package main

import (
	"fmt"
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"structs/app"
	"structs/cmd/structsd/cmd"

    //sdk "github.com/cosmos/cosmos-sdk/types"
    //"cosmossdk.io/math"

)

func main() {
    // Was used to change the Staking voting power
    // But now asset definition of Alpha is more in-line with Cosmos network norms
    //sdk.DefaultPowerReduction = math.NewIntFromUint64(10)

	rootCmd := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}
