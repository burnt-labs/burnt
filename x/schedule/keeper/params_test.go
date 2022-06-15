package keeper_test

import (
	"testing"

	testkeeper "github.com/BurntFinance/burnt/testutil/keeper"
	"github.com/BurntFinance/burnt/x/schedule/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.ScheduleKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
