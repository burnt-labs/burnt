package keeper_test

import (
	"testing"

	testkeeper "github.com/BurntFinance/burnt/testutil/keeper"
	"github.com/BurntFinance/burnt/x/burnt/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.BurntKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
