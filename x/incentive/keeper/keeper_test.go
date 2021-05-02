package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/kava-labs/kava/app"
	committeekeeper "github.com/kava-labs/kava/x/committee/keeper"
	hardkeeper "github.com/kava-labs/kava/x/hard/keeper"
	"github.com/kava-labs/kava/x/incentive/keeper"
	"github.com/kava-labs/kava/x/incentive/types"
)

// Test suite used for all keeper tests
type KeeperTestSuite struct {
	suite.Suite

	keeper          keeper.Keeper
	hardKeeper      hardkeeper.Keeper
	stakingKeeper   stakingkeeper.Keeper
	committeeKeeper committeekeeper.Keeper
	app             app.TestApp
	ctx             sdk.Context
	addrs           []sdk.AccAddress
	validatorAddrs  []sdk.ValAddress
}

// SetupTest is run automatically before each suite test
func (suite *KeeperTestSuite) SetupTest() {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)

	_, allAddrs := app.GeneratePrivKeyAddressPairs(10)
	suite.addrs = allAddrs[:5]
	for _, a := range allAddrs[5:] {
		suite.validatorAddrs = append(suite.validatorAddrs, sdk.ValAddress(a))
	}
}

func (suite *KeeperTestSuite) SetupApp() {
	suite.app = app.NewTestApp()

	suite.keeper = suite.app.GetIncentiveKeeper()
	suite.hardKeeper = suite.app.GetHardKeeper()
	suite.stakingKeeper = suite.app.GetStakingKeeper()
	suite.committeeKeeper = suite.app.GetCommitteeKeeper()

	suite.ctx = suite.app.NewContext(true, abci.Header{Height: 1, Time: tmtime.Now()})
}

// getAllAddrs returns all user and validator addresses in the suite
func (suite *KeeperTestSuite) getAllAddrs() []sdk.AccAddress {
	accAddrs := []sdk.AccAddress{} // initialize new slice to avoid accidental modifications to underlying
	accAddrs = append(accAddrs, suite.addrs...)
	for _, a := range suite.validatorAddrs {
		accAddrs = append(accAddrs, sdk.AccAddress(a))
	}
	return accAddrs
}

func (suite *KeeperTestSuite) SetupWithGenState() {
	suite.SetupApp()

	suite.app.InitializeFromGenesisStates(
		NewAuthGenState(suite.getAllAddrs(), cs(c("ukava", 1_000_000_000))),
		NewStakingGenesisState(),
		NewPricefeedGenStateMulti(),
		NewCDPGenStateMulti(),
		NewHardGenStateMulti(),
		NewCommitteeGenesisState(suite.addrs[:2]), // TODO add committee members to suite
	)

}
func (suite *KeeperTestSuite) TestGetSetDeleteUSDXMintingClaim() {
	suite.SetupApp()
	c := types.NewUSDXMintingClaim(suite.addrs[0], c("ukava", 1000000), types.RewardIndexes{types.NewRewardIndex("bnb-a", sdk.ZeroDec())})
	_, found := suite.keeper.GetUSDXMintingClaim(suite.ctx, suite.addrs[0])
	suite.Require().False(found)
	suite.Require().NotPanics(func() {
		suite.keeper.SetUSDXMintingClaim(suite.ctx, c)
	})
	testC, found := suite.keeper.GetUSDXMintingClaim(suite.ctx, suite.addrs[0])
	suite.Require().True(found)
	suite.Require().Equal(c, testC)
	suite.Require().NotPanics(func() {
		suite.keeper.DeleteUSDXMintingClaim(suite.ctx, suite.addrs[0])
	})
	_, found = suite.keeper.GetUSDXMintingClaim(suite.ctx, suite.addrs[0])
	suite.Require().False(found)
}

func (suite *KeeperTestSuite) TestIterateUSDXMintingClaims() {
	suite.SetupApp()
	for i := 0; i < len(suite.addrs); i++ {
		c := types.NewUSDXMintingClaim(suite.addrs[i], c("ukava", 100000), types.RewardIndexes{types.NewRewardIndex("bnb-a", sdk.ZeroDec())})
		suite.Require().NotPanics(func() {
			suite.keeper.SetUSDXMintingClaim(suite.ctx, c)
		})
	}
	claims := types.USDXMintingClaims{}
	suite.keeper.IterateUSDXMintingClaims(suite.ctx, func(c types.USDXMintingClaim) bool {
		claims = append(claims, c)
		return false
	})
	suite.Require().Equal(len(suite.addrs), len(claims))

	claims = suite.keeper.GetAllUSDXMintingClaims(suite.ctx)
	suite.Require().Equal(len(suite.addrs), len(claims))
}

func createPeriodicVestingAccount(origVesting sdk.Coins, periods vesting.Periods, startTime, endTime int64) (*vesting.PeriodicVestingAccount, error) {
	_, addr := app.GeneratePrivKeyAddressPairs(1)
	bacc := auth.NewBaseAccountWithAddress(addr[0])
	bacc.Coins = origVesting
	bva, err := vesting.NewBaseVestingAccount(&bacc, origVesting, endTime)
	if err != nil {
		return &vesting.PeriodicVestingAccount{}, err
	}
	pva := vesting.NewPeriodicVestingAccountRaw(bva, startTime, periods)
	err = pva.Validate()
	if err != nil {
		return &vesting.PeriodicVestingAccount{}, err
	}
	return pva, nil
}

// Avoid cluttering test cases with long function names
func i(in int64) sdk.Int                    { return sdk.NewInt(in) }
func d(str string) sdk.Dec                  { return sdk.MustNewDecFromStr(str) }
func c(denom string, amount int64) sdk.Coin { return sdk.NewInt64Coin(denom, amount) }
func cs(coins ...sdk.Coin) sdk.Coins        { return sdk.NewCoins(coins...) }

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
