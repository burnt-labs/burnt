package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/BurntFinance/burnt/testutil/keeper"
	"github.com/BurntFinance/burnt/x/burnt/keeper"
	"github.com/BurntFinance/burnt/x/burnt/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.BurntKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
