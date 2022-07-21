package keeper

import (
	"context"
	"encoding/json"
	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"strconv"
)

func (k msgServer) AddSchedule(goCtx context.Context, msg *types.MsgAddSchedule) (*types.MsgAddScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if msg.BlockHeight <= uint64(ctx.BlockHeight()) {
		return nil, types.ErrInvalidScheduledBlockHeight
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, err
	}

	contract, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, err
	}

	ownerQueryMsg, err := json.Marshal(map[string]interface{}{
		"is_owner": signer.Bytes(),
	})
	ownerQueryRes, err := k.wasmViewKeeper.QuerySmart(ctx, contract, ownerQueryMsg)
	if err != nil {
		return nil, err
	}

	isOwner, err := strconv.ParseBool(string(ownerQueryRes))
	if err != nil {
		return nil, err
	}
	if !isOwner {
		return nil, types.ErrUnauthorized
	}

	// todo: anti spam protection. how do we keep this from getting blown up for free?
	// probably we just charge gas for this

	gasDenom := k.GetParams(ctx).GasDenom
	balanceMinimum := k.GetParams(ctx).ScheduledBalanceMinimum
	balance := k.bankKeeper.GetBalance(ctx, contract, gasDenom)

	if balance.Amount.LT(sdk.NewIntFromUint64(balanceMinimum)) {
		// the contract doesn't have the funds
		return nil, types.ErrUnmetMinimumBalance
	}

	k.AddScheduledCall(ctx, signer, contract, msg.CallBody, msg.BlockHeight)

	return &types.MsgAddScheduleResponse{}, nil
}
