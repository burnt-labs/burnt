package types

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	ParamsStoreKeyMinimumBalance = []byte("MinimumBalance")
	ParamsStoreKeyUpperBound     = []byte("UpperBound")

	// Ensure that params implements the proper interface
	_ paramtypes.ParamSet = (*Params)(nil)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(gasMin sdk.Coin, upperBound uint64) Params {
	return Params{
		MinimumBalance: gasMin,
		UpperBound:     upperBound,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(sdk.NewCoin("default-token", sdk.NewInt(100)), 1000)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamsStoreKeyMinimumBalance, &p.MinimumBalance, validateMinimumBalance),
		paramtypes.NewParamSetPair(ParamsStoreKeyUpperBound, &p.UpperBound, validateUpperBound),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMinimumBalance(p.MinimumBalance); err != nil {
		return sdkerrors.Wrap(err, "minimum balance")
	}
	if err := validateUpperBound(p.UpperBound); err != nil {
		return sdkerrors.Wrap(err, "upper bound")
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validation functions

func validateMinimumBalance(i interface{}) error {
	v, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if err := sdk.ValidateDenom(v.Denom); err != nil {
		return err
	}
	if v.Amount.Uint64() == 0 {
		return fmt.Errorf("cannot provide empty minimum gas amount")
	}

	return nil
}

func validateUpperBound(i interface{}) error {
	val, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if val == 0 {
		return fmt.Errorf("invalid value for upper bound, can't be zero")
	}

	return nil
}
