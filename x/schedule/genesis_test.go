package schedule_test

import (
	"testing"

	keepertest "github.com/burnt-labs/burnt/testutil/keeper"
	"github.com/burnt-labs/burnt/testutil/nullify"
	"github.com/burnt-labs/burnt/x/schedule"
	"github.com/burnt-labs/burnt/x/schedule/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ScheduleKeeper(t)
	schedule.InitGenesis(ctx, *k, genesisState)
	got := schedule.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
