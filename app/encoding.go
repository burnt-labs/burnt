package app

import (
	"github.com/BurntFinance/burnt/app/params"
	"github.com/cosmos/cosmos-sdk/std"
)

func MakeEncodingConfig() params.EncodingConfig {
	encodingConfig := MakeEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
