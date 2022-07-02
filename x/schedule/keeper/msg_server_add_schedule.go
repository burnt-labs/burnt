package keeper

import (
	"context"

	"github.com/BurntFinance/burnt/x/schedule/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

	payer, err := sdk.AccAddressFromBech32(msg.Payer)
	if err != nil {
		return nil, err
	}

	contract, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, err
	}

	// todo: anti spam protection. how do we keep this from getting blown up for free?

	gasDenom := k.GetParams(ctx).GasDenom
	payingAccount := signer
	if !payer.Equals(signer) {
		payingAccount = payer
	}
	limits, err := k.determineGasLimit(ctx, payingAccount, signer)
	if err != nil {
		panic(err)
	}
	balance := k.bankKeeper.GetBalance(ctx, payingAccount, gasDenom)
	limit := limits.AmountOfNoDenomValidation(gasDenom)
	if limit.LT(balance.Amount) {
		// the payer doesn't have the funds
		return nil, types.ErrInvalidAllowance
	}

	k.AddScheduledCall(ctx, signer, contract, msg.FunctionName, msg.BlockHeight, &payer)

	return &types.MsgAddScheduleResponse{}, nil
}
