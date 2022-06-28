package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

type WasmKeeper interface {
	Execute(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error)
}

type FeeGrantKeeper interface {
	GetAllowance(ctx sdk.Context, granter, grantee sdk.AccAddress) (feegrant.FeeAllowanceI, error)
	IterateAllFeeAllowances(ctx sdk.Context, cb func(grant feegrant.Grant) bool) error
	UseGrantedFees(ctx sdk.Context, granter, grantee sdk.AccAddress, fee sdk.Coins, msgs []sdk.Msg) error
}

type BankKeeper interface {
	GetDenomMetaData(ctx sdk.Context, denom string) (banktypes.Metadata, bool)
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}
