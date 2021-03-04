package incentive

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/kava-labs/kava/x/incentive/keeper"
	"github.com/kava-labs/kava/x/incentive/types"
)

// InitGenesis initializes the store state from a genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, supplyKeeper types.SupplyKeeper, cdpKeeper types.CdpKeeper, gs types.GenesisState) {

	// check if the module account exists
	moduleAcc := supplyKeeper.GetModuleAccount(ctx, types.IncentiveMacc)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.IncentiveMacc))
	}

	if err := gs.Validate(); err != nil {
		panic(fmt.Sprintf("failed to validate %s genesis state: %s", types.ModuleName, err))
	}

	for _, rp := range gs.Params.USDXMintingRewardPeriods {
		_, found := cdpKeeper.GetCollateral(ctx, rp.CollateralType)
		if !found {
			panic(fmt.Sprintf("usdx minting collateral type %s not found in cdp collateral types", rp.CollateralType))
		}
		k.SetUSDXMintingRewardFactor(ctx, rp.CollateralType, sdk.ZeroDec())
	}

	for _, mrp := range gs.Params.HardSupplyRewardPeriods {
		newRewardIndexes := types.RewardIndexes{}
		for _, rc := range mrp.RewardsPerSecond {
			ri := types.NewRewardIndex(rc.Denom, sdk.ZeroDec())
			newRewardIndexes = append(newRewardIndexes, ri)
		}
		k.SetHardSupplyRewardIndexes(ctx, mrp.CollateralType, newRewardIndexes)
	}

	for _, mrp := range gs.Params.HardBorrowRewardPeriods {
		newRewardIndexes := types.RewardIndexes{}
		for _, rc := range mrp.RewardsPerSecond {
			ri := types.NewRewardIndex(rc.Denom, sdk.ZeroDec())
			newRewardIndexes = append(newRewardIndexes, ri)
		}
		k.SetHardBorrowRewardIndexes(ctx, mrp.CollateralType, newRewardIndexes)
	}

	for _, rp := range gs.Params.HardDelegatorRewardPeriods {
		k.SetHardDelegatorRewardFactor(ctx, rp.CollateralType, sdk.ZeroDec())
	}

	k.SetParams(ctx, gs.Params)

	for _, gat := range gs.USDXAccumulationTimes {
		k.SetPreviousUSDXMintingAccrualTime(ctx, gat.CollateralType, gat.PreviousAccumulationTime)
		k.SetUSDXMintingRewardFactor(ctx, gat.CollateralType, gat.RewardFactor)
	}

	for _, gat := range gs.HardSupplyAccumulationTimes {
		k.SetPreviousHardSupplyRewardAccrualTime(ctx, gat.CollateralType, gat.PreviousAccumulationTime)
	}

	for _, gat := range gs.HardBorrowAccumulationTimes {
		k.SetPreviousHardBorrowRewardAccrualTime(ctx, gat.CollateralType, gat.PreviousAccumulationTime)
	}

	for _, gat := range gs.HardDelegatorAccumulationTimes {
		k.SetPreviousHardDelegatorRewardAccrualTime(ctx, gat.CollateralType, gat.PreviousAccumulationTime)
	}

	for i, claim := range gs.USDXMintingClaims {
		for j, ri := range claim.RewardIndexes {
			if ri.RewardFactor != sdk.ZeroDec() {
				gs.USDXMintingClaims[i].RewardIndexes[j].RewardFactor = sdk.ZeroDec()
			}
		}
		k.SetUSDXMintingClaim(ctx, claim)
	}

	for i, claim := range gs.HardLiquidityProviderClaims {
		for j, mri := range claim.SupplyRewardIndexes {
			for k, ri := range mri.RewardIndexes {
				if ri.RewardFactor != sdk.ZeroDec() {
					gs.HardLiquidityProviderClaims[i].SupplyRewardIndexes[j].RewardIndexes[k].RewardFactor = sdk.ZeroDec()
				}
			}
		}
		for j, mri := range claim.BorrowRewardIndexes {
			for k, ri := range mri.RewardIndexes {
				if ri.RewardFactor != sdk.ZeroDec() {
					gs.HardLiquidityProviderClaims[i].BorrowRewardIndexes[j].RewardIndexes[k].RewardFactor = sdk.ZeroDec()
				}
			}
		}
		for j, ri := range claim.DelegatorRewardIndexes {
			if ri.RewardFactor != sdk.ZeroDec() {
				gs.HardLiquidityProviderClaims[i].DelegatorRewardIndexes[j].RewardFactor = sdk.ZeroDec()
			}
		}
		k.SetHardLiquidityProviderClaim(ctx, claim)
	}
}

// ExportGenesis export genesis state for incentive module
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) types.GenesisState {
	params := k.GetParams(ctx)

	usdxClaims := k.GetAllUSDXMintingClaims(ctx)
	hardClaims := k.GetAllHardLiquidityProviderClaims(ctx)

	synchronizedUsdxClaims := types.USDXMintingClaims{}
	synchronizedHardClaims := types.HardLiquidityProviderClaims{}

	for _, usdxClaim := range usdxClaims {
		claim, err := k.SynchronizeUSDXMintingClaim(ctx, usdxClaim)
		if err != nil {
			panic(err)
		}
		// for i := range claim.RewardIndexes {
		// 	claim.RewardIndexes[i].RewardFactor = sdk.ZeroDec()
		// }
		synchronizedUsdxClaims = append(synchronizedUsdxClaims, claim)
	}

	for _, hardClaim := range hardClaims {
		k.SynchronizeHardLiquidityProviderClaim(ctx, hardClaim.Owner)
		claim, found := k.GetHardLiquidityProviderClaim(ctx, hardClaim.Owner)
		if !found {
			panic("hard liquidity provider claim should always be found after synchronization")
		}
		// for i, bri := range claim.BorrowRewardIndexes {
		// 	for j := range bri.RewardIndexes {
		// 		claim.BorrowRewardIndexes[i].RewardIndexes[j].RewardFactor = sdk.ZeroDec()
		// 	}
		// }
		// for i, sri := range claim.SupplyRewardIndexes {
		// 	for j := range sri.RewardIndexes {
		// 		claim.SupplyRewardIndexes[i].RewardIndexes[j].RewardFactor = sdk.ZeroDec()
		// 	}
		// }
		// for i := range claim.DelegatorRewardIndexes {
		// 	claim.DelegatorRewardIndexes[i].RewardFactor = sdk.ZeroDec()
		// }
		synchronizedHardClaims = append(synchronizedHardClaims, claim)
	}

	var usdxGenesisAccumulationTimes GenesisAccumulationTimes
	for _, rp := range params.USDXMintingRewardPeriods {
		pat, found := k.GetPreviousUSDXMintingAccrualTime(ctx, rp.CollateralType)
		if !found {
			pat = ctx.BlockTime()
		}
		factor, found := k.GetUSDXMintingRewardFactor(ctx, rp.CollateralType)
		if !found {
			factor = sdk.ZeroDec()
		}
		gat := types.NewGenesisAccumulationTime(rp.CollateralType, pat, factor)
		usdxGenesisAccumulationTimes = append(usdxGenesisAccumulationTimes, gat)
	}

	var hardDelegatorAccumTimes GenesisAccumulationTimes
	for _, rp := range params.HardDelegatorRewardPeriods {
		pat, found := k.GetPreviousHardDelegatorRewardAccrualTime(ctx, rp.CollateralType)
		if !found {
			pat = ctx.BlockTime()
		}
		factor, found := k.GetHardDelegatorRewardFactor(ctx, rp.CollateralType)
		if !found {
			factor = sdk.ZeroDec()
		}
		gat := types.NewGenesisAccumulationTime(rp.CollateralType, pat, factor)
		hardDelegatorAccumTimes = append(hardDelegatorAccumTimes, gat)
	}

	var hardSupplyGenesisAccumulationTimes types.GenesisAccumulationTimes
	for _, mrp := range params.HardSupplyRewardPeriods {
		pat, found := k.GetPreviousHardSupplyRewardAccrualTime(ctx, mrp.CollateralType)
		if !found {
			pat = ctx.BlockTime()
		}
		rewardIndexes, found := k.GetHardSupplyRewardIndexes(ctx, mrp.CollateralType)
		if !found {
			rewardIndexes = types.RewardIndexes{}
		}
		factor := sdk.ZeroDec()
		for _, ri := range rewardIndexes {
			if ri.RewardFactor.GT(sdk.ZeroDec()) {
				factor = ri.RewardFactor
			}
		}
		hardSupplyGenesisAccumulationTime := types.NewGenesisAccumulationTime(mrp.CollateralType, pat, factor)
		hardSupplyGenesisAccumulationTimes = append(hardSupplyGenesisAccumulationTimes, hardSupplyGenesisAccumulationTime)
	}

	var hardBorrowGenesisAccumulationTimes types.GenesisAccumulationTimes
	for _, mrp := range params.HardBorrowRewardPeriods {
		pat, found := k.GetPreviousHardBorrowRewardAccrualTime(ctx, mrp.CollateralType)
		if !found {
			pat = ctx.BlockTime()
		}
		rewardIndexes, found := k.GetHardBorrowRewardIndexes(ctx, mrp.CollateralType)
		if !found {
			rewardIndexes = types.RewardIndexes{}
		}
		factor := sdk.ZeroDec()
		for _, ri := range rewardIndexes {
			if ri.RewardFactor.GT(sdk.ZeroDec()) {
				factor = ri.RewardFactor
			}
		}
		hardBorrowGenesisAccumulationTime := types.NewGenesisAccumulationTime(mrp.CollateralType, pat, factor)
		hardBorrowGenesisAccumulationTimes = append(hardBorrowGenesisAccumulationTimes, hardBorrowGenesisAccumulationTime)
	}

	return types.NewGenesisState(params, usdxGenesisAccumulationTimes, hardSupplyGenesisAccumulationTimes, hardBorrowGenesisAccumulationTimes, hardDelegatorAccumTimes, synchronizedUsdxClaims, synchronizedHardClaims)
}
