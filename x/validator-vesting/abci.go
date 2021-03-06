package validatorvesting

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/kava-labs/kava/x/validator-vesting/keeper"
)

// BeginBlocker updates the vote signing information for each validator vesting account, updates account when period changes, and updates the previousBlockTime value in the store.
func BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock, k keeper.Keeper) {
	previousBlockTime, found := k.GetPreviousBlockTime(ctx)
	if !found {
		k.SetPreviousBlockTime(ctx, ctx.BlockTime())
		return
	}

	currentBlockTime := ctx.BlockTime()
	var voteInfos VoteInfos
	voteInfos = req.LastCommitInfo.GetVotes()
	validatorVestingKeys := k.GetAllAccountKeys(ctx)

	for _, key := range validatorVestingKeys {
		acc := k.GetAccountFromAuthKeeper(ctx, key)
		if !k.AccountIsVesting(ctx, acc.GetAddress()) {
			continue
		}

		vote, found := voteInfos.FilterByValidatorAddress(acc.ValidatorAddress)

		var missedBlock bool
		// if the validator was not found or explicitly didn't sign, increment the missing sign count
		if (!found || !vote.SignedLastBlock) && ctx.BlockHeight() > 1 {
			missedBlock = true
		}

		k.UpdateMissingSignCount(ctx, acc.GetAddress(), missedBlock)

		// check if a period ended in the last block
		endTimes := k.GetPeriodEndTimes(ctx, key)

		for i, t := range endTimes {
			if currentBlockTime.Unix() >= t && previousBlockTime.Unix() < t {
				k.UpdateVestedCoinsProgress(ctx, key, i)
			}
		}
		// handle any new/remaining debt on the account
		k.HandleVestingDebt(ctx, key, currentBlockTime)
	}

	k.SetPreviousBlockTime(ctx, currentBlockTime)
}

// VoteInfos an array of abci.VoteInfo
type VoteInfos []abci.VoteInfo

// FilterByValidatorAddress returns the VoteInfo of the validator address matching the input validator address
// and a boolean for if the address was found.
func (vis VoteInfos) FilterByValidatorAddress(consAddress sdk.ConsAddress) (abci.VoteInfo, bool) {
	for i, vi := range vis {
		votingAddress := sdk.ConsAddress(vi.Validator.Address)
		if bytes.Equal(consAddress, votingAddress) {
			return vis[i], true
		}
	}
	return abci.VoteInfo{}, false
}
