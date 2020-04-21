package checkpoint

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/gorilla/mux"
	chainmanagerTypes "github.com/maticnetwork/heimdall/chainmanager/types"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"

	checkpointCli "github.com/maticnetwork/heimdall/checkpoint/client/cli"
	checkpointRest "github.com/maticnetwork/heimdall/checkpoint/client/rest"
	"github.com/maticnetwork/heimdall/checkpoint/types"
	"github.com/maticnetwork/heimdall/helper"
	"github.com/maticnetwork/heimdall/staking"
	hmModule "github.com/maticnetwork/heimdall/types/module"
)

var (
	_ module.AppModule             = AppModule{}
	_ module.AppModuleBasic        = AppModuleBasic{}
	_ hmModule.HeimdallModuleBasic = AppModule{}
	// _ module.AppModuleSimulation = AppModule{}
)

// AppModuleBasic defines the basic application module used by the auth module.
type AppModuleBasic struct{}

// Name returns the auth module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterCodec registers the auth module's types for the given codec.
func (AppModuleBasic) RegisterCodec(cdc *codec.Codec) {
	types.RegisterCodec(cdc)
}

// DefaultGenesis returns default genesis state as raw bytes for the auth
// module.
func (AppModuleBasic) DefaultGenesis() json.RawMessage {
	return types.ModuleCdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the auth module.
func (AppModuleBasic) ValidateGenesis(bz json.RawMessage) error {
	var data types.GenesisState
	err := types.ModuleCdc.UnmarshalJSON(bz, &data)
	if err != nil {
		return err
	}
	return types.ValidateGenesis(data)
}

// VerifyGenesis performs verification on auth module state.
func (AppModuleBasic) VerifyGenesis(bz map[string]json.RawMessage) error {
	var chainManagertData chainmanagerTypes.GenesisState
	errcm := chainmanagerTypes.ModuleCdc.UnmarshalJSON(bz[chainmanagerTypes.ModuleName], &chainManagertData)
	if errcm != nil {
		return errcm
	}

	var data types.GenesisState
	err := types.ModuleCdc.UnmarshalJSON(bz[types.ModuleName], &data)

	if err != nil {
		return err
	}
	return verifyGenesis(data, chainManagertData)
}

// RegisterRESTRoutes registers the REST routes for the auth module.
func (AppModuleBasic) RegisterRESTRoutes(ctx context.CLIContext, rtr *mux.Router) {
	checkpointRest.RegisterRoutes(ctx, rtr)
}

// GetTxCmd returns the root tx command for the auth module.
func (AppModuleBasic) GetTxCmd(cdc *codec.Codec) *cobra.Command {
	return checkpointCli.GetTxCmd(cdc)
}

// GetQueryCmd returns the root query command for the auth module.
func (AppModuleBasic) GetQueryCmd(cdc *codec.Codec) *cobra.Command {
	return checkpointCli.GetQueryCmd(cdc)
}

//____________________________________________________________________________

// AppModule implements an application module for the checkpoint module.
type AppModule struct {
	AppModuleBasic

	keeper         Keeper
	stakingKeeper  staking.Keeper
	contractCaller helper.IContractCaller
}

// NewAppModule creates a new AppModule object
func NewAppModule(keeper Keeper, sk staking.Keeper, contractCaller helper.IContractCaller) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         keeper,
		stakingKeeper:  sk,
		contractCaller: contractCaller,
	}
}

// Name returns the auth module's name.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterInvariants performs a no-op.
func (AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// Route returns the message routing key for the auth module.
func (AppModule) Route() string {
	return types.RouterKey
}

// NewHandler returns an sdk.Handler for the module.
func (am AppModule) NewHandler() sdk.Handler {
	return NewHandler(am.keeper, am.contractCaller)
}

// QuerierRoute returns the auth module's querier route name.
func (AppModule) QuerierRoute() string {
	return types.QuerierRoute
}

// NewQuerierHandler returns the auth module sdk.Querier.
func (am AppModule) NewQuerierHandler() sdk.Querier {
	return NewQuerier(am.keeper, am.stakingKeeper)
}

// InitGenesis performs genesis initialization for the auth module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	types.ModuleCdc.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.keeper, genesisState)
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the auth
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return types.ModuleCdc.MustMarshalJSON(gs)
}

// BeginBlock returns the begin blocker for the auth module.
func (AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock returns the end blocker for the auth module. It returns no validator
// updates.
func (AppModule) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

//
// Internal methods
//
func verifyGenesis(state types.GenesisState, chainManagerState chainmanagerTypes.GenesisState) error {
	contractCaller, err := helper.NewContractCaller()
	if err != nil {
		return err
	}

	rootChainAddress := chainManagerState.Params.ChainParams.RootChainAddress.EthAddress()
	rootChainInstance, _ := contractCaller.GetRootChainInstance(rootChainAddress)

	// check header count
	currentHeaderIndex, err := contractCaller.CurrentHeaderBlock(rootChainInstance)
	if err != nil {
		return nil
	}

	if state.AckCount*helper.GetConfig().ChildBlockInterval != currentHeaderIndex {
		fmt.Println("Header Count doesn't match",
			"ExpectedHeader", currentHeaderIndex,
			"HeaderIndexFound", state.AckCount*helper.GetConfig().ChildBlockInterval)
		return nil
	}

	fmt.Println("ACK count valid:", "count", currentHeaderIndex)

	// check all headers
	for i, header := range state.Headers {
		ackCount := uint64(i + 1)
		root, start, end, _, _, err := contractCaller.GetHeaderInfo(ackCount*helper.GetConfig().ChildBlockInterval, rootChainInstance)
		if err != nil {
			return err
		}

		if header.StartBlock != start || header.EndBlock != end || !bytes.Equal(header.RootHash.Bytes(), root.Bytes()) {
			return fmt.Errorf(
				"Checkpoint block doesnt match: startExpected %v, startReceived %v, endExpected %v, endReceived %v, rootHashExpected %v, rootHashReceived %v",
				header.StartBlock,
				start,
				header.EndBlock,
				header.EndBlock,
				header.RootHash.String(),
				root.String(),
			)
		}
		fmt.Println("Checkpoint block valid:", "start", start, "end", end, "root", root.String())
	}

	return nil
}
