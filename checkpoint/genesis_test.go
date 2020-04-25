package checkpoint_test

import (
	"math/rand"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/maticnetwork/heimdall/app"
	"github.com/maticnetwork/heimdall/checkpoint"
	"github.com/maticnetwork/heimdall/checkpoint/types"
	hmTypes "github.com/maticnetwork/heimdall/types"
	"github.com/maticnetwork/heimdall/types/simulation"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
)

type GenesisTestSuite struct {
	suite.Suite

	app *app.HeimdallApp
	ctx sdk.Context
}

// SetupTest setup necessary things for genesis test
func (suite *GenesisTestSuite) SetupTest() {
	suite.app = app.SetupCheckpointGenesis()
	suite.ctx = suite.app.BaseApp.NewContext(true, abci.Header{})
}

// TestGenesisTestSuite
func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}

func (suite *GenesisTestSuite) TestInitExportGenesis() {
	t, app, ctx := suite.T(), suite.app, suite.ctx
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	lastNoACK := simulation.RandIntBetween(r1, 1, 5)
	ackCount := simulation.RandIntBetween(r1, 1, 5)
	startBlock := uint64(0)
	endBlock := uint64(256)
	rootHash := hmTypes.HexToHeimdallHash("123")
	accountRootHash := hmTypes.HexToHeimdallHash("456")

	proposerAddress := hmTypes.HexToHeimdallAddress("123")
	timestamp := uint64(time.Now().Unix())
	bufferedCheckpoint := hmTypes.CreateBlock(
		startBlock,
		endBlock,
		rootHash,
		accountRootHash,
		proposerAddress,
		timestamp,
	)

	checkpointBlockHeaders := make([]hmTypes.CheckpointBlockHeader, ackCount)

	for i := range checkpointBlockHeaders {
		checkpointBlockHeaders[i] = bufferedCheckpoint
	}

	params := types.DefaultParams()
	genesisState := types.NewGenesisState(
		params,
		&bufferedCheckpoint,
		uint64(lastNoACK),
		uint64(ackCount),
		checkpointBlockHeaders,
	)

	checkpoint.InitGenesis(ctx, app.CheckpointKeeper, genesisState)
	actualParams := checkpoint.ExportGenesis(ctx, app.CheckpointKeeper)

	require.Equal(t, genesisState, actualParams)
}
