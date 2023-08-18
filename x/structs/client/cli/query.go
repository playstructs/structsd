package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"structs/x/structs/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group structs queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())
	cmd.AddCommand(CmdListReactor())
	cmd.AddCommand(CmdShowReactor())
	cmd.AddCommand(CmdListSubstation())
	cmd.AddCommand(CmdShowSubstation())
	cmd.AddCommand(CmdListAllocation())
	cmd.AddCommand(CmdShowAllocation())
    cmd.AddCommand(CmdListInfusion())
    cmd.AddCommand(CmdShowInfusion())
	cmd.AddCommand(CmdListGuild())
	cmd.AddCommand(CmdShowGuild())
	cmd.AddCommand(CmdListPlayer())
	cmd.AddCommand(CmdShowPlayer())
	// this line is used by starport scaffolding # 1

	return cmd
}
