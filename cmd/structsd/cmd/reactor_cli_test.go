package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stretchr/testify/require"
	"structs/app"
)

// Each real root command seals the SDK address config. Isolated processes also
// keep Cobra flags and client config from leaking between generated transactions.
func TestReactorCLIProcess(t *testing.T) {
	raw := os.Getenv("STRUCTS_REACTOR_CLI_TEST_ARGS")
	if raw == "" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		panic(err)
	}
	app.DefaultNodeHome = os.Getenv("STRUCTS_REACTOR_CLI_TEST_HOME")
	root := NewRootCmd()
	root.SetArgs(args)
	if err := svrcmd.Execute(root, "", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestReactorCLIDefaultVaultAndExplicitDelegator(t *testing.T) {
	address, err := bech32.ConvertAndEncode("structs", bytes.Repeat([]byte{1}, 20))
	require.NoError(t, err)
	validator, err := bech32.ConvertAndEncode("structsvaloper", bytes.Repeat([]byte{2}, 20))
	require.NoError(t, err)
	destination, err := bech32.ConvertAndEncode("structsvaloper", bytes.Repeat([]byte{3}, 20))
	require.NoError(t, err)
	for _, operation := range []struct {
		command, message string
		args             []string
	}{
		{"reactor-infuse", "MsgReactorInfuse", []string{validator, "1000000ualpha"}},
		{"reactor-defuse", "MsgReactorDefuse", []string{validator, "1000000ualpha"}},
		{"reactor-cancel-defusion", "MsgReactorCancelDefusion", []string{validator, "1000000ualpha", "123"}},
		{"reactor-begin-migration", "MsgReactorBeginMigration", []string{validator, destination, "1000000ualpha"}},
	} {
		for _, selector := range []string{"default", "guild", "explicit", "invalid"} {
			t.Run(operation.command+"/"+selector, func(t *testing.T) {
				args := append([]string{"tx", "structs", operation.command}, operation.args...)
				switch selector {
				case "guild":
					args = append(args, "--owner-id", "0-1")
				case "explicit":
					args = append(args, "--delegator-address", address)
				case "invalid":
					args = append(args, "--delegator-address", "invalid-address")
				}
				home := t.TempDir()
				args = append(args, "--generate-only", "--offline", "--from", address,
					"--home", home, "--chain-id", "", "--account-number", "1", "--sequence", "1")
				encoded, err := json.Marshal(args)
				require.NoError(t, err)
				child := exec.Command(os.Args[0], "-test.run=^TestReactorCLIProcess$")
				child.Env = append(os.Environ(), "STRUCTS_REACTOR_CLI_TEST_ARGS="+string(encoded), "STRUCTS_REACTOR_CLI_TEST_HOME="+home)
				var stdout, stderr bytes.Buffer
				child.Stdout, child.Stderr = &stdout, &stderr
				err = child.Run()
				if selector == "invalid" {
					require.Error(t, err)
					require.Contains(t, stderr.String(), "invalid account address")
					return
				}
				require.NoError(t, err, stderr.String())
				var tx struct {
					Body struct {
						Messages []map[string]any `json:"messages"`
					} `json:"body"`
				}
				require.NoError(t, json.Unmarshal(stdout.Bytes(), &tx), stdout.String())
				require.Len(t, tx.Body.Messages, 1)
				message := tx.Body.Messages[0]
				require.Equal(t, "/structs.structs."+operation.message, message["@type"])
				require.Equal(t, address, message["creator"])
				require.Equal(t, map[string]any{"denom": "ualpha", "amount": "1000000"}, message["amount"])
				if operation.command == "reactor-begin-migration" {
					require.Equal(t, validator, message["validator_src_address"])
					require.Equal(t, destination, message["validator_dst_address"])
				} else {
					require.Equal(t, validator, message["validator_address"])
				}
				if operation.command == "reactor-cancel-defusion" {
					require.Equal(t, "123", message["creation_height"])
				}
				if selector == "explicit" {
					require.Equal(t, address, message["delegator_address"])
				} else {
					require.Empty(t, message["delegator_address"])
				}
				if selector == "guild" {
					require.Equal(t, "0-1", message["ownerId"])
				}
			})
		}
	}
}
