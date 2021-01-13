package keeper

import (
	"math"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	"github.com/kava-labs/kava/x/incentive/types"
)

// AccumulateUSDXMintingRewards updates the rewards accumulated for the input reward period
func (k Keeper) AccumulateUSDXMintingRewards(ctx sdk.Context, rewardPeriod types.RewardPeriod) error {
	previousAccrualTime, found := k.GetPreviousUSDXMintingAccrualTime(ctx, rewardPeriod.CollateralType)
	if !found {
		k.SetPreviousUSDXMintingAccrualTime(ctx, rewardPeriod.CollateralType, ctx.BlockTime())
		return nil
	}
	timeElapsed := CalculateTimeElapsed(rewardPeriod, ctx.BlockTime(), previousAccrualTime)
	if timeElapsed.IsZero() {
		return nil
	}
	if rewardPeriod.RewardsPerSecond.Amount.IsZero() {
		k.SetPreviousUSDXMintingAccrualTime(ctx, rewardPeriod.CollateralType, ctx.BlockTime())
		return nil
	}
	totalPrincipal := k.cdpKeeper.GetTotalPrincipal(ctx, rewardPeriod.CollateralType, types.PrincipalDenom).ToDec()
	if totalPrincipal.IsZero() {
		k.SetPreviousUSDXMintingAccrualTime(ctx, rewardPeriod.CollateralType, ctx.BlockTime())
		return nil
	}
	newRewards := timeElapsed.Mul(rewardPeriod.RewardsPerSecond.Amount)
	rewardFactor := newRewards.ToDec().Quo(totalPrincipal)

	previousRewardFactor, found := k.GetUSDXMintingRewardFactor(ctx, rewardPeriod.CollateralType)
	if !found {
		previousRewardFactor = sdk.ZeroDec()
	}
	newRewardFactor := previousRewardFactor.Add(rewardFactor)
	k.SetUSDXMintingRewardFactor(ctx, rewardPeriod.CollateralType, newRewardFactor)
	k.SetPreviousUSDXMintingAccrualTime(ctx, rewardPeriod.CollateralType, ctx.BlockTime())
	return nil
}

// TODO: reward period 'denom' instead of 'collateralType'

// AccumulateHardBorrowRewards updates the rewards accumulated for the input reward period
func (k Keeper) AccumulateHardBorrowRewards(ctx sdk.Context, rewardPeriod types.RewardPeriod) error {
	previousAccrualTime, found := k.GetPreviousHardBorrowRewardAccrualTime(ctx, rewardPeriod.CollateralType)
	if !found {
		k.SetPreviousHardBorrowRewardAccrualTime(ctx, rewardPeriod.CollateralType, ctx.BlockTime())
		return nil
	}
	timeElapsed := CalculateTimeElapsed(rewardPeriod, ctx.BlockTime(), previousAccrualTime)
	if timeElapsed.IsZero() {
		return nil
	}
	if rewardPeriod.RewardsPerSecond.Amount.IsZero() {
		k.SetPreviousHardBorrowRewardAccrualTime(ctx, rewardPeriod.CollateralType, ctx.BlockTime())
		return nil
	}
	totalBorrowedCoins := k.hardKeeper.GetBorrowedCoins(ctx)
	for _, coin := range totalBorrowedCoins {
		if coin.Denom == rewardPeriod.CollateralType {
			totalBorrowed := totalBorrowedCoins.AmountOf(coin.Denom).ToDec()
			if totalBorrowed.IsZero() {
				k.SetPreviousHardBorrowRewardAccrualTime(ctx, rewardPeriod.CollateralType, ctx.BlockTime())
				return nil
			}
			newRewards := timeElapsed.Mul(rewardPeriod.RewardsPerSecond.Amount)
			rewardFactor := newRewards.ToDec().Quo(totalBorrowed)

			previousRewardFactor, found := k.GetHardBorrowRewardFactor(ctx, rewardPeriod.CollateralType)
			if !found {
				previousRewardFactor = sdk.ZeroDec()
			}
			newRewardFactor := previousRewardFactor.Add(rewardFactor)
			k.SetHardBorrowRewardFactor(ctx, rewardPeriod.CollateralType, newRewardFactor)
			k.SetPreviousHardBorrowRewardAccrualTime(ctx, rewardPeriod.CollateralType, ctx.BlockTime())
		}
	}

	return nil
}

// InitializeUSDXMintingClaim creates or updates a claim such that no new rewards are accrued, but any existing rewards are not lost.
// this function should be called after a cdp is created. If a user previously had a cdp, then closed it, they shouldn't
// accrue rewards during the period the cdp was closed. By setting the reward factor to the current global reward factor,
// any unclaimed rewards are preserved, but no new rewards are added.
func (k Keeper) InitializeUSDXMintingClaim(ctx sdk.Context, cdp cdptypes.CDP) {
	_, found := k.GetRewardPeriod(ctx, cdp.Type)
	if !found {
		// this collateral type is not incentivized, do nothing
		return
	}
	rewardFactor, found := k.GetUSDXMintingRewardFactor(ctx, cdp.Type)
	if !found {
		rewardFactor = sdk.ZeroDec()
	}
	claim, found := k.GetUSDXMintingClaim(ctx, cdp.Owner)
	if !found { // this is the owner's first usdx minting reward claim
		claim = types.NewUSDXMintingClaim(cdp.Owner, sdk.NewCoin(types.USDXMintingRewardDenom, sdk.ZeroInt()), types.RewardIndexes{types.NewRewardIndex(cdp.Type, rewardFactor)})
		k.SetUSDXMintingClaim(ctx, claim)
		return
	}
	// the owner has an existing usdx minting reward claim
	index, hasRewardIndex := claim.HasRewardIndex(cdp.Type)
	if !hasRewardIndex { // this is the owner's first usdx minting reward for this collateral type
		claim.RewardIndexes = append(claim.RewardIndexes, types.NewRewardIndex(cdp.Type, rewardFactor))
	} else { // the owner has a previous usdx minting reward for this collateral type
		claim.RewardIndexes[index] = types.NewRewardIndex(cdp.Type, rewardFactor)
	}
	k.SetUSDXMintingClaim(ctx, claim)
}

// SynchronizeUSDXMintingReward updates the claim object by adding any accumulated rewards and updating the reward index value.
// this should be called before a cdp is modified, immediately after the 'SynchronizeInterest' method is called in the cdp module
func (k Keeper) SynchronizeUSDXMintingReward(ctx sdk.Context, cdp cdptypes.CDP) {
	_, found := k.GetRewardPeriod(ctx, cdp.Type)
	if !found {
		// this collateral type is not incentivized, do nothing
		return
	}

	globalRewardFactor, found := k.GetUSDXMintingRewardFactor(ctx, cdp.Type)
	if !found {
		globalRewardFactor = sdk.ZeroDec()
	}
	claim, found := k.GetUSDXMintingClaim(ctx, cdp.Owner)
	if !found {
		claim = types.NewUSDXMintingClaim(cdp.Owner, sdk.NewCoin(types.USDXMintingRewardDenom, sdk.ZeroInt()), types.RewardIndexes{types.NewRewardIndex(cdp.Type, globalRewardFactor)})
		k.SetUSDXMintingClaim(ctx, claim)
		return
	}

	// the owner has an existing usdx minting reward claim
	index, hasRewardIndex := claim.HasRewardIndex(cdp.Type)
	if !hasRewardIndex { // this is the owner's first usdx minting reward for this collateral type
		claim.RewardIndexes = append(claim.RewardIndexes, types.NewRewardIndex(cdp.Type, globalRewardFactor))
		k.SetUSDXMintingClaim(ctx, claim)
		return
	}
	userRewardFactor := claim.RewardIndexes[index].RewardFactor
	rewardsAccumulatedFactor := globalRewardFactor.Sub(userRewardFactor)
	if rewardsAccumulatedFactor.IsZero() {
		return
	}
	claim.RewardIndexes[index].RewardFactor = globalRewardFactor
	newRewardsAmount := rewardsAccumulatedFactor.Mul(cdp.GetTotalPrincipal().Amount.ToDec()).RoundInt()
	if newRewardsAmount.IsZero() {
		k.SetUSDXMintingClaim(ctx, claim)
		return
	}
	newRewardsCoin := sdk.NewCoin(types.USDXMintingRewardDenom, newRewardsAmount)
	claim.Reward = claim.Reward.Add(newRewardsCoin)
	k.SetUSDXMintingClaim(ctx, claim)
	return
}

// InitializeHardLiquidityBorrowReward initializes the borrow-side of a hard liquidity provider claim
// by creating the claim and setting the borrow reward factor index
func (k Keeper) InitializeHardLiquidityBorrowReward(ctx sdk.Context, borrow hardtypes.Borrow, denom string) {
	_, found := k.GetRewardPeriod(ctx, denom)
	if !found {
		return
	}

	borrowFactor, foundBorrowFactor := k.GetHardBorrowRewardFactor(ctx, denom)
	if !foundBorrowFactor {
		borrowFactor = sdk.ZeroDec()
	}

	claim, found := k.GetHardLiquidityProviderClaim(ctx, borrow.Borrower)
	// User's first hard liquidity reward for this denom on the borrow-side
	if !found {
		claim = types.NewHardLiquidityProviderClaim(
			borrow.Borrower,
			sdk.NewCoin(types.HardLiquidityRewardDenom, sdk.ZeroInt()),
			types.RewardIndexes{},
			types.RewardIndexes{types.NewRewardIndex(denom, borrowFactor)},
			types.RewardIndexes{},
		)
		k.SetHardLiquidityProviderClaim(ctx, claim)
		return
	}

	// Update user's existing hard liquidity reward claim with current borrow reward index factor
	borrowIndex, hasBorrowRewardIndex := claim.HasBorrowRewardIndex(denom)
	if !hasBorrowRewardIndex {
		claim.BorrowRewardIndexes = append(claim.BorrowRewardIndexes, types.NewRewardIndex(denom, borrowFactor))
	} else {
		claim.BorrowRewardIndexes[borrowIndex] = types.NewRewardIndex(denom, borrowFactor)
	}

	k.SetHardLiquidityProviderClaim(ctx, claim)
}

// SynchronizeHardLiquidityBorrowReward updates the claim object by adding any accumulated rewards
// and updating the reward index value
func (k Keeper) SynchronizeHardLiquidityBorrowReward(ctx sdk.Context, borrow hardtypes.Borrow, denom string) {
	_, found := k.GetRewardPeriod(ctx, denom)
	if !found {
		return
	}

	borrowFactor, found := k.GetHardBorrowRewardFactor(ctx, denom)
	if !found {
		borrowFactor = sdk.ZeroDec()
	}

	claim, found := k.GetHardLiquidityProviderClaim(ctx, borrow.Borrower)
	if !found {
		claim = types.NewHardLiquidityProviderClaim(
			borrow.Borrower,
			sdk.NewCoin(types.HardLiquidityRewardDenom, sdk.ZeroInt()),
			types.RewardIndexes{},
			types.RewardIndexes{types.NewRewardIndex(denom, borrowFactor)},
			types.RewardIndexes{},
		)
		k.SetHardLiquidityProviderClaim(ctx, claim)
		return
	}

	borrowIndex, hasBorrowRewardIndex := claim.HasBorrowRewardIndex(denom)
	if !hasBorrowRewardIndex {
		claim.BorrowRewardIndexes = append(claim.BorrowRewardIndexes, types.NewRewardIndex(denom, borrowFactor))
		k.SetHardLiquidityProviderClaim(ctx, claim)
	}

	userRewardFactor := claim.BorrowRewardIndexes[borrowIndex].RewardFactor
	rewardsAccumulatedFactor := borrowFactor.Sub(userRewardFactor)
	if rewardsAccumulatedFactor.IsZero() {
		return
	}
	claim.BorrowRewardIndexes[borrowIndex].RewardFactor = borrowFactor
	newRewardsAmount := rewardsAccumulatedFactor.Mul(borrow.Amount.AmountOf(denom).ToDec()).RoundInt()
	if newRewardsAmount.IsZero() {
		k.SetHardLiquidityProviderClaim(ctx, claim)
		return
	}
	newRewardsCoin := sdk.NewCoin(types.HardLiquidityRewardDenom, newRewardsAmount)
	claim.Reward = claim.Reward.Add(newRewardsCoin)
	k.SetHardLiquidityProviderClaim(ctx, claim)
	return
}

// ZeroClaim zeroes out the claim object's rewards and returns the updated claim object
func (k Keeper) ZeroClaim(ctx sdk.Context, claim types.USDXMintingClaim) types.USDXMintingClaim {
	claim.Reward = sdk.NewCoin(claim.Reward.Denom, sdk.ZeroInt())
	k.SetUSDXMintingClaim(ctx, claim)
	return claim
}

// SynchronizeClaim updates the claim object by adding any rewards that have accumulated.
// Returns the updated claim object
func (k Keeper) SynchronizeClaim(ctx sdk.Context, claim types.USDXMintingClaim) (types.USDXMintingClaim, error) {
	for _, ri := range claim.RewardIndexes {
		cdp, found := k.cdpKeeper.GetCdpByOwnerAndCollateralType(ctx, claim.Owner, ri.CollateralType)
		if !found {
			// if the cdp for this collateral type has been closed, no updates are needed
			continue
		}
		claim = k.synchronizeRewardAndReturnClaim(ctx, cdp)
	}
	return claim, nil
}

// this function assumes a claim already exists, so don't call it if that's not the case
func (k Keeper) synchronizeRewardAndReturnClaim(ctx sdk.Context, cdp cdptypes.CDP) types.USDXMintingClaim {
	k.SynchronizeUSDXMintingReward(ctx, cdp)
	claim, _ := k.GetUSDXMintingClaim(ctx, cdp.Owner)
	return claim
}

// CalculateTimeElapsed calculates the number of reward-eligible seconds that have passed since the previous
// time rewards were accrued, taking into account the end time of the reward period
func CalculateTimeElapsed(rewardPeriod types.RewardPeriod, blockTime time.Time, previousAccrualTime time.Time) sdk.Int {
	if rewardPeriod.End.Before(blockTime) &&
		(rewardPeriod.End.Before(previousAccrualTime) || rewardPeriod.End.Equal(previousAccrualTime)) {
		return sdk.ZeroInt()
	}
	if rewardPeriod.End.Before(blockTime) {
		return sdk.NewInt(int64(math.RoundToEven(
			rewardPeriod.End.Sub(previousAccrualTime).Seconds(),
		)))
	}
	return sdk.NewInt(int64(math.RoundToEven(
		blockTime.Sub(previousAccrualTime).Seconds(),
	)))
}
