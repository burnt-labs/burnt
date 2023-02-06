package ibc_tests

import (
	"context"
	"encoding/json"
	"fmt"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	paramsproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/icza/dyno"
	"os"
	"strings"
	"testing"
	"time"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	votingPeriod = "10s"
	maxDepositPeriod = "10x"
)
func TestDungeonTransferBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	ctx := context.Background()

	imageTag := os.Getenv("BURNT_IMAGE")
	imageTagComponents := strings.Split(imageTag, ":")

	// Chain factory
	cf := ibctest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*ibctest.ChainSpec{
		{Name: "osmosis", Version: "v11.0.0"},
		{
			Name:    imageTagComponents[0],
			Version: imageTagComponents[1],
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: imageTagComponents[0],
						Version:    imageTagComponents[1],
						UidGid:     "1025:1025",
					},
				},
				GasPrices:      "0.0uburnt",
				GasAdjustment:  1.3,
				Type:           "cosmos",
				ChainID:        "burnt-1",
				Bin:            "burntd",
				Bech32Prefix:   "burnt",
				Denom:          "uburnt",
				TrustingPeriod: "336h",
				ModifyGenesis: modifyGenesisShortProposals(votingPeriod, maxDepositPeriod),
			},
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	osmosis, burnt := chains[0], chains[1]

	// Relayer Factory
	client, network := ibctest.DockerSetup(t)
	relayer := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(
		t, client, network)

	// Prep Interchain
	const ibcPath = "burnt-osmo-dungeon-test"
	ic := ibctest.NewInterchain().
		AddChain(burnt).
		AddChain(osmosis).
		AddRelayer(relayer, "relayer").
		AddLink(ibctest.InterchainLink{
			Chain1:  burnt,
			Chain2:  osmosis,
			Relayer: relayer,
			Path:    ibcPath,
		})

	// Log location
	f, err := ibctest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	// Reporter/logs
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build Interchain
	require.NoError(t, ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: ibctest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false},
	),
	)

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, burnt, osmosis)
	burntUser := users[0]
	osmosisUser := users[1]
	t.Logf("created burnt user %s", burntUser.FormattedAddress())
	t.Logf("created osmosis user %s", osmosisUser.FormattedAddress())

	burntUserBalInitial, err := burnt.GetBalance(ctx, burntUser.FormattedAddress(), burnt.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, burntUserBalInitial)

	// Get Channel ID
	t.Log("getting IBC channel IDs")
	burntChannelInfo, err := relayer.GetChannels(ctx, eRep, burnt.Config().ChainID)
	require.NoError(t, err)
	burntChannelID := burntChannelInfo[0].ChannelID

	osmoChannelInfo, err := relayer.GetChannels(ctx, eRep, osmosis.Config().ChainID)
	require.NoError(t, err)
	osmoChannelID := osmoChannelInfo[0].ChannelID

	// Query staking denom
	t.Log("verifying staking denom")
	grpcAddress := burnt.GetHostGRPCAddress()
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	require.NoError(t, err)

	stakingQueryClient := stakingtypes.NewQueryClient(conn)
	paramsResponse, err := stakingQueryClient.Params(ctx, &stakingtypes.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, "uburnt", paramsResponse.Params.BondDenom)

	// Disable sends of Burnt staking token
	t.Log("disabling sendability of burnt staking token")
	sendEnableds := []*banktypes.SendEnabled{
		{
			Denom: "uburnt",
			Enabled: false,
		},
	}
	data, err := json.Marshal(sendEnableds)
	require.NoError(t, err)

	proposal := paramsproposal.NewParameterChangeProposal(
		"disable burnt send",
		"disable burnt send but longer",
		[]paramsproposal.ParamChange{
		{
			Subspace: banktypes.ModuleName,
			Key: "SendEnabled",
			Value: string(data),
			},
		},
		)
	govClient := govtypes.NewMsgClient(conn)
	govClient.SubmitProposal(ctx, govtypes.NewMsgSubmitProposal(proposal, sdktypes.Coins{}, burnt.)

	// Send Transaction
	t.Log("sending tokens from burnt to osmosis")
	amountToSend := int64(1_000_000)
	dstAddress := osmosisUser.FormattedAddress()
	transfer := ibc.WalletAmount{
		Address: dstAddress,
		Denom:   burnt.Config().Denom,
		Amount:  amountToSend,
	}
	tx, err := burnt.SendIBCTransfer(ctx, burntChannelID, burntUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)
	require.NoError(t, tx.Validate())

	// relay packets and acknowledgments
	require.NoError(t, relayer.FlushPackets(ctx, eRep, ibcPath, osmoChannelID))
	require.NoError(t, relayer.FlushAcknowledgements(ctx, eRep, ibcPath, burntChannelID))

	// test source wallet has decreased funds
	expectedBal := burntUserBalInitial - amountToSend
	gaiaUserBalNew, err := burnt.GetBalance(ctx, burntUser.FormattedAddress(), burnt.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, expectedBal, gaiaUserBalNew)

	// Trace IBC Denom
	srcDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", burntChannelID, burnt.Config().Denom))
	dstIbcDenom := srcDenomTrace.IBCDenom()

	// Test destination wallet has increased funds
	t.Log("verifying receipt of tokens on osmosis")
	osmosUserBalNew, err := osmosis.GetBalance(ctx, osmosisUser.FormattedAddress(), dstIbcDenom)
	require.NoError(t, err)
	require.Equal(t, amountToSend, osmosUserBalNew)
}


func modifyGenesisShortProposals(votingPeriod string, maxDepositPeriod string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "voting_params", "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "deposit_params", "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "deposit_params", "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
