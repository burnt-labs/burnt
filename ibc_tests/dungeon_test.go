package ibc_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramsutils "github.com/cosmos/cosmos-sdk/x/params/client/utils"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transfertypes "github.com/cosmos/ibc-go/v6/modules/apps/transfer/types"
	"github.com/icza/dyno"
	ibctest "github.com/strangelove-ventures/interchaintest/v6"
	"github.com/strangelove-ventures/interchaintest/v6/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v6/ibc"
	"github.com/strangelove-ventures/interchaintest/v6/testreporter"
	"github.com/strangelove-ventures/interchaintest/v6/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	votingPeriod     = "10s"
	maxDepositPeriod = "10s"
)

func TestDungeonTransferBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	ctx := context.Background()

	// pulling image from env to foster local dev
	imageTag := os.Getenv("BURNT_IMAGE")
	imageTagComponents := strings.Split(imageTag, ":")

	// disabling seeds in osmosis because it causes intermittent test failures
	osmoConfigFileOverrides := make(map[string]any)
	osmoConfigTomlOverrides := make(testutil.Toml)

	osmoP2POverrides := make(testutil.Toml)
	osmoP2POverrides["seeds"] = ""
	osmoConfigTomlOverrides["p2p"] = osmoP2POverrides

	osmoConfigFileOverrides["config/config.toml"] = osmoConfigTomlOverrides

	// Chain factory
	cf := ibctest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*ibctest.ChainSpec{
		{
			Name:    "osmosis",
			Version: "v14.0.0",
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/strangelove-ventures/heighliner/osmosis",
						Version:    "v14.0.0",
						UidGid:     "1025:1025",
					},
				},
				Type:                "cosmos",
				Bin:                 "osmosisd",
				Bech32Prefix:        "osmo",
				Denom:               "uosmo",
				GasPrices:           "0.0uosmo",
				GasAdjustment:       1.3,
				TrustingPeriod:      "336h",
				NoHostMount:         false,
				ConfigFileOverrides: osmoConfigFileOverrides,
			},
		},
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
				ModifyGenesis:  modifyGenesisShortProposals(votingPeriod, maxDepositPeriod),
			},
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	osmosis, burnt := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

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
			Denom:   "uburnt",
			Enabled: false,
		},
	}
	data, err := json.Marshal(sendEnableds)
	require.NoError(t, err)

	prop := paramsutils.ParamChangeProposalJSON{
		Title:       "Disable sendability of uburnt",
		Description: "This proposal prevents uburnt from being sent in the bank module",
		Changes: []paramsutils.ParamChangeJSON{
			{
				Subspace: banktypes.ModuleName,
				Key:      "SendEnabled",
				Value:    data,
			},
		},
		Deposit: "100uburnt",
	}

	paramChangeTx, err := burnt.ParamChangeProposal(ctx, burntUser.KeyName(), &prop)
	require.NoError(t, err)
	t.Logf("Param change proposal submitted with ID %s in transaction %s", paramChangeTx.ProposalID, paramChangeTx.TxHash)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := burnt.QueryProposal(ctx, paramChangeTx.ProposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == cosmos.ProposalStatusVotingPeriod {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status VOTING, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status VOTING after 11s")

	err = burnt.VoteOnProposalAllValidators(ctx, paramChangeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := burnt.QueryProposal(ctx, paramChangeTx.ProposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == cosmos.ProposalStatusPassed {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status PASSED, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status PASSED after 11s")

	// Send Transaction
	t.Log("sending tokens from burnt to osmosis")
	amountToSend := int64(1_000_000)
	dstAddress := osmosisUser.FormattedAddress()
	transfer := ibc.WalletAmount{
		Address: dstAddress,
		Denom:   burnt.Config().Denom,
		Amount:  amountToSend,
	}
	_, err = burnt.SendIBCTransfer(ctx, burntChannelID, burntUser.KeyName(), transfer, ibc.TransferOptions{})
	require.Error(t, err)

	// relay packets and acknowledgments
	require.NoError(t, relayer.FlushPackets(ctx, eRep, ibcPath, osmoChannelID))
	require.NoError(t, relayer.FlushAcknowledgements(ctx, eRep, ibcPath, burntChannelID))

	// test source wallet has decreased funds
	burntUserBalNew, err := burnt.GetBalance(ctx, burntUser.FormattedAddress(), burnt.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, burntUserBalInitial, burntUserBalNew)

	// Trace IBC Denom
	srcDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", burntChannelID, burnt.Config().Denom))
	burntOnOsmoIbcDenom := srcDenomTrace.IBCDenom()

	// Test destination wallet has increased funds
	t.Log("verifying receipt of tokens on osmosis")
	osmosUserBalNew, err := osmosis.GetBalance(ctx, osmosisUser.FormattedAddress(), burntOnOsmoIbcDenom)
	require.NoError(t, err)
	require.Equal(t, int64(0), osmosUserBalNew)

	// Create a user without any funds
	emptyKeyName := "burnt-empty-key"
	err = burnt.CreateKey(ctx, emptyKeyName)
	require.NoError(t, err)
	emptyKeyAddressBytes, err := burnt.GetAddress(ctx, emptyKeyName)
	require.NoError(t, err)
	emptyKeyAddress, err := types.Bech32ifyAddressBytes(burnt.Config().Bech32Prefix, emptyKeyAddressBytes)
	require.NoError(t, err)

	transfer = ibc.WalletAmount{
		Address: emptyKeyAddress,
		Denom:   osmosis.Config().Denom,
		Amount:  int64(1_000_000),
	}
	_, err = osmosis.SendIBCTransfer(ctx, osmoChannelID, osmosisUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	// relay packets and acknowledgments
	require.NoError(t, relayer.FlushPackets(ctx, eRep, ibcPath, osmoChannelID))
	require.NoError(t, relayer.FlushAcknowledgements(ctx, eRep, ibcPath, burntChannelID))

	osmoUserBalAfterIbcTransfer, err := osmosis.GetBalance(ctx, osmosisUser.FormattedAddress(), osmosis.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(9_000_000), osmoUserBalAfterIbcTransfer)

	emptyUserBals, err := burnt.AllBalances(ctx, emptyKeyAddress)
	require.NoError(t, err)
	require.Equal(t, 1, len(emptyUserBals))

	osmoDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", osmoChannelID, osmosis.Config().Denom))
	osmoOnBurntIbcDenom := osmoDenomTrace.IBCDenom()

	coin := emptyUserBals[0]
	require.Equal(t, osmoOnBurntIbcDenom, coin.Denom)
	require.Equal(t, int64(1_000_000), coin.Amount.Int64())

	require.NoError(t, burnt.SendFunds(ctx, emptyKeyName, ibc.WalletAmount{
		Address: burntUser.FormattedAddress(),
		Denom:   osmoOnBurntIbcDenom,
		Amount:  1_000_000,
	}))

	burntUserOsmoBal, err := burnt.GetBalance(ctx, burntUser.FormattedAddress(), osmoOnBurntIbcDenom)
	require.NoError(t, err)
	require.Equal(t, int64(1_000_000), burntUserOsmoBal)

	transfer = ibc.WalletAmount{
		Address: osmosisUser.FormattedAddress(),
		Denom:   osmoOnBurntIbcDenom,
		Amount:  int64(1_000_000),
	}
	_, err = burnt.SendIBCTransfer(ctx, burntChannelID, burntUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)
	require.NoError(t, relayer.FlushPackets(ctx, eRep, ibcPath, burntChannelID))
	require.NoError(t, relayer.FlushAcknowledgements(ctx, eRep, ibcPath, osmoChannelID))

	osmoUserBalAfterIbcReturnTransfer, err := osmosis.GetBalance(ctx, osmosisUser.FormattedAddress(), osmosis.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(10_000_000), osmoUserBalAfterIbcReturnTransfer)
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
			return nil, fmt.Errorf("failed to set max deposit period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "deposit_params", "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set min deposit denom in genesis json: %w", err)
		}
		if err := dyno.Set(g, "100", "app_state", "gov", "deposit_params", "min_deposit", 0, "amount"); err != nil {
			return nil, fmt.Errorf("failed to set min deposit amount in genesis json: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
