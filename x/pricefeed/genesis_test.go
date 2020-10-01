package pricefeed_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/pricefeed"
)

type GenesisTestSuite struct {
	suite.Suite

	ctx    sdk.Context
	keeper pricefeed.Keeper
}

func (suite *GenesisTestSuite) TestValidGenState() {
	tApp := app.NewTestApp()

	suite.NotPanics(func() {
		tApp.InitializeFromGenesisStates(
			NewPricefeedGenStateMulti(),
		)
	})
	_, addrs := app.GeneratePrivKeyAddressPairs(10)

	suite.NotPanics(func() {
		tApp.InitializeFromGenesisStates(
			NewPricefeedGenStateWithOracles(addrs),
		)
	})
}

func (suite *GenesisTestSuite) TestPostPriceAfterInitGenesis() {
	_, oracles := app.GeneratePrivKeyAddressPairs(10)
	genPriceExpiry := time.Date(1998, 1, 1, 0, 0, 0, 0, time.UTC)

	pfGenesis := pricefeed.GenesisState{
		Params: pricefeed.Params{
			Markets: []pricefeed.Market{
				{MarketID: "btc:usd", BaseAsset: "btc", QuoteAsset: "usd", Oracles: oracles, Active: true},
			},
		},
		PostedPrices: []pricefeed.PostedPrice{
			{
				MarketID:      "btc:usd",
				OracleAddress: oracles[0],
				Price:         sdk.MustNewDecFromStr("8000.00"),
				Expiry:        genPriceExpiry,
			},
			{
				MarketID:      "btc:usd",
				OracleAddress: oracles[1],
				Price:         sdk.MustNewDecFromStr("8000.00"),
				Expiry:        genPriceExpiry.Add(2 * time.Nanosecond),
			},
		},
	}

	tApp := app.NewTestApp()
	genesisTime := genPriceExpiry.Add(1 * time.Nanosecond)
	tApp.InitializeFromGenesisStatesWithTime(
		genesisTime,
		app.GenesisState{pricefeed.ModuleName: pricefeed.ModuleCdc.MustMarshalJSON(pfGenesis)},
	)
	keeper := tApp.GetPriceFeedKeeper()

	ctx := tApp.NewContext(false, abci.Header{Time: genesisTime.Add(1 * time.Hour)})
	_, err := keeper.SetPrice(
		ctx,
		oracles[0],
		"btc:usd",
		sdk.MustNewDecFromStr("9000.00"),
		genPriceExpiry.Add(11*time.Hour),
	)

	suite.Require().NoError(err)
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}
