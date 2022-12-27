package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/burnt-labs/burnt/testutil/keeper"
	"github.com/burnt-labs/burnt/x/schedule/keeper"
	"github.com/burnt-labs/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.ScheduleKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
