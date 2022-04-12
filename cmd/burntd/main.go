package main

import (
	"os"

	"github.com/BurntFinance/burnt/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/spf13/cobra"
	"github.com/tendermint/spm/cosmoscmd"
	//"github.com/tendermint/starport/starport/pkg/cosmoscmd"
	tmcmds "github.com/tendermint/tendermint/cmd/tendermint/commands"
)

func main() {
	rootCmd, _ := cosmoscmd.NewRootCmd(
		app.Name,
		app.AccountAddressPrefix,
		app.DefaultNodeHome,
		app.Name,
		app.ModuleBasics,
		app.New,
		// this line is used by starport scaffolding # root/arguments
		cosmoscmd.AddSubCmd(tmcmds.RollbackStateCmd),
		cosmoscmd.CustomizeStartCmd(func(startCmd *cobra.Command) {
			wasm.AddModuleInitFlags(startCmd)
		}),
	)
	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
