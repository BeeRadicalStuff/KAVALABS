package v0_11

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	v0_11bep3 "github.com/kava-labs/kava/x/bep3"
	v0_9bep3 "github.com/kava-labs/kava/x/bep3/legacy/v0_9"
	v0_11cdp "github.com/kava-labs/kava/x/cdp"
	v0_9cdp "github.com/kava-labs/kava/x/cdp/legacy/v0_9"
	v0_11committee "github.com/kava-labs/kava/x/committee"
	v0_9committee "github.com/kava-labs/kava/x/committee/legacy/v0_9"
	v0_11pricefeed "github.com/kava-labs/kava/x/pricefeed"
	v0_9pricefeed "github.com/kava-labs/kava/x/pricefeed/legacy/v0_9"
)

// MigrateBep3 migrates from a v0.9 (or v0.10) bep3 genesis state to a v0.11 bep3 genesis state
func MigrateBep3(oldGenState v0_9bep3.GenesisState) v0_11bep3.GenesisState {
	var assetParams v0_11bep3.AssetParams
	var assetSupplies v0_11bep3.AssetSupplies
	v0_9Params := oldGenState.Params

	for _, asset := range v0_9Params.SupportedAssets {
		v11AssetParam := v0_11bep3.AssetParam{
			Active:        asset.Active,
			Denom:         asset.Denom,
			CoinID:        asset.CoinID,
			DeputyAddress: v0_9Params.BnbDeputyAddress,
			FixedFee:      v0_9Params.BnbDeputyFixedFee,
			MinSwapAmount: sdk.OneInt(), // set min swap to one - prevents accounts that hold zero bnb from creating spam txs
			MaxSwapAmount: v0_9Params.MaxAmount,
			MinBlockLock:  v0_9Params.MinBlockLock,
			MaxBlockLock:  v0_9Params.MaxBlockLock,
			SupplyLimit: v0_11bep3.SupplyLimit{
				Limit:          asset.Limit,
				TimeLimited:    false,
				TimePeriod:     time.Duration(0),
				TimeBasedLimit: sdk.ZeroInt(),
			},
		}
		assetParams = append(assetParams, v11AssetParam)
	}
	for _, supply := range oldGenState.AssetSupplies {
		newSupply := v0_11bep3.NewAssetSupply(supply.IncomingSupply, supply.OutgoingSupply, supply.CurrentSupply, sdk.NewCoin(supply.CurrentSupply.Denom, sdk.ZeroInt()), time.Duration(0))
		assetSupplies = append(assetSupplies, newSupply)
	}
	var swaps v0_11bep3.AtomicSwaps
	for _, oldSwap := range oldGenState.AtomicSwaps {
		newSwap := v0_11bep3.AtomicSwap{
			Amount:              oldSwap.Amount,
			RandomNumberHash:    oldSwap.RandomNumberHash,
			ExpireHeight:        oldSwap.ExpireHeight,
			Timestamp:           oldSwap.Timestamp,
			Sender:              oldSwap.Sender,
			Recipient:           oldSwap.Recipient,
			SenderOtherChain:    oldSwap.SenderOtherChain,
			RecipientOtherChain: oldSwap.RecipientOtherChain,
			ClosedBlock:         oldSwap.ClosedBlock,
			Status:              v0_11bep3.SwapStatus(oldSwap.Status),
			CrossChain:          oldSwap.CrossChain,
			Direction:           v0_11bep3.SwapDirection(oldSwap.Direction),
		}
		swaps = append(swaps, newSwap)
	}

	// -------------- ADD BTCB To BEP3 params --------------------
	btcbAssetParam := v0_11bep3.NewAssetParam(
		"btcb",
		0,
		v0_11bep3.SupplyLimit{
			Limit:          sdk.NewInt(10000000000), // 100 BTC limit at launch
			TimeLimited:    false,
			TimePeriod:     time.Duration(0),
			TimeBasedLimit: sdk.ZeroInt()},
		true,
		v0_9Params.BnbDeputyAddress, // TODO get an additional deputy address from binance
		v0_9Params.BnbDeputyFixedFee,
		sdk.OneInt(),
		sdk.NewInt(1000000000),
		220,
		270,
	)
	btcbAssetSupply := v0_11bep3.NewAssetSupply(
		sdk.NewCoin("btcb", sdk.ZeroInt()),
		sdk.NewCoin("btcb", sdk.ZeroInt()),
		sdk.NewCoin("btcb", sdk.ZeroInt()),
		sdk.NewCoin("btcb", sdk.ZeroInt()),
		time.Duration(0))
	assetParams = append(assetParams, btcbAssetParam)
	assetSupplies = append(assetSupplies, btcbAssetSupply)
	xrpbAssetParam := v0_11bep3.NewAssetParam(
		"xrpb",
		144,
		v0_11bep3.SupplyLimit{
			Limit:          sdk.NewInt(1000000000000), // 1,000,000 XRP limit at launch
			TimeLimited:    false,
			TimePeriod:     time.Duration(0),
			TimeBasedLimit: sdk.ZeroInt()},
		true,
		v0_9Params.BnbDeputyAddress, // TODO  get an additional deputy address from binance
		v0_9Params.BnbDeputyFixedFee,
		sdk.OneInt(),
		sdk.NewInt(100000000000),
		220,
		270,
	)
	xrpbAssetSupply := v0_11bep3.NewAssetSupply(
		sdk.NewCoin("xrpb", sdk.ZeroInt()),
		sdk.NewCoin("xrpb", sdk.ZeroInt()),
		sdk.NewCoin("xrpb", sdk.ZeroInt()),
		sdk.NewCoin("xrpb", sdk.ZeroInt()),
		time.Duration(0))
	assetParams = append(assetParams, xrpbAssetParam)
	assetSupplies = append(assetSupplies, xrpbAssetSupply)
	busdAssetParam := v0_11bep3.NewAssetParam(
		"busd",
		727, // note - no official SLIP 44 ID
		v0_11bep3.SupplyLimit{
			Limit:          sdk.NewInt(10000000000000), // 100,000 BUSD limit at launch
			TimeLimited:    false,
			TimePeriod:     time.Duration(0),
			TimeBasedLimit: sdk.ZeroInt()},
		true,
		v0_9Params.BnbDeputyAddress, // TODO  get an additional deputy address from binance
		v0_9Params.BnbDeputyFixedFee,
		sdk.OneInt(),
		sdk.NewInt(1000000000000),
		220,
		270,
	)
	busdAssetSupply := v0_11bep3.NewAssetSupply(
		sdk.NewCoin("busd", sdk.ZeroInt()),
		sdk.NewCoin("busd", sdk.ZeroInt()),
		sdk.NewCoin("busd", sdk.ZeroInt()),
		sdk.NewCoin("busd", sdk.ZeroInt()),
		time.Duration(0))
	assetParams = append(assetParams, busdAssetParam)
	assetSupplies = append(assetSupplies, busdAssetSupply)
	return v0_11bep3.GenesisState{
		Params:            v0_11bep3.NewParams(assetParams),
		AtomicSwaps:       swaps,
		Supplies:          assetSupplies,
		PreviousBlockTime: v0_11bep3.DefaultPreviousBlockTime,
	}
}

// MigrateCDP migrates from a v0.9 (or v0.10) cdp genesis state to a v0.11 cdp genesis state
func MigrateCDP(oldGenState v0_9cdp.GenesisState) v0_11cdp.GenesisState {
	var newCDPs v0_11cdp.CDPs
	var newDeposits v0_11cdp.Deposits
	var newCollateralParams v0_11cdp.CollateralParams
	newStartingID := uint64(0)

	for _, cdp := range oldGenState.CDPs {
		newCDP := v0_11cdp.NewCDPWithFees(cdp.ID, cdp.Owner, cdp.Collateral, "bnb-a", cdp.Principal, cdp.AccumulatedFees, cdp.FeesUpdated)
		newCDPs = append(newCDPs, newCDP)
		if cdp.ID >= newStartingID {
			newStartingID = cdp.ID + 1
		}
	}

	for _, dep := range oldGenState.Deposits {
		newDep := v0_11cdp.NewDeposit(dep.CdpID, dep.Depositor, dep.Amount)
		newDeposits = append(newDeposits, newDep)
	}

	for _, cp := range oldGenState.Params.CollateralParams {
		newCollateralParam := v0_11cdp.NewCollateralParam(cp.Denom, "bnb-a", cp.LiquidationRatio, cp.DebtLimit, cp.StabilityFee, cp.AuctionSize, cp.LiquidationPenalty, 0x01, cp.SpotMarketID, cp.LiquidationMarketID, cp.ConversionFactor)
		newCollateralParams = append(newCollateralParams, newCollateralParam)
	}
	btcbCollateralParam := v0_11cdp.NewCollateralParam("btcb", "btcb-a", sdk.MustNewDecFromStr("1.5"), sdk.NewCoin("usdx", sdk.NewInt(100000000000)), sdk.MustNewDecFromStr("1.000000001547125958"), sdk.NewInt(100000000), sdk.MustNewDecFromStr("0.075000000000000000"), 0x02, "btc:usd", "btc:usd:30", sdk.NewInt(8))
	busdaCollateralParam := v0_11cdp.NewCollateralParam("busd", "busd-a", sdk.MustNewDecFromStr("1.01"), sdk.NewCoin("usdx", sdk.NewInt(3000000000000)), sdk.OneDec(), sdk.NewInt(1000000000000), sdk.MustNewDecFromStr("0.075000000000000000"), 0x03, "busd:usd", "busd:usd:30", sdk.NewInt(8))
	busdbCollateralParam := v0_11cdp.NewCollateralParam("busd", "busd-b", sdk.MustNewDecFromStr("1.1"), sdk.NewCoin("usdx", sdk.NewInt(1000000000000)), sdk.MustNewDecFromStr("1.000000012857214317"), sdk.NewInt(1000000000000), sdk.MustNewDecFromStr("0.075000000000000000"), 0x04, "busd:usd", "busd:usd:30", sdk.NewInt(8))
	xrpbCollateralParam := v0_11cdp.NewCollateralParam("xrpb", "xrpb-a", sdk.MustNewDecFromStr("1.5"), sdk.NewCoin("usdx", sdk.NewInt(100000000000)), sdk.MustNewDecFromStr("1.000000001547125958"), sdk.NewInt(100000000000), sdk.MustNewDecFromStr("0.075000000000000000"), 0x05, "xrp:usd", "xrp:usd:30", sdk.NewInt(8))
	newCollateralParams = append(newCollateralParams, btcbCollateralParam, busdaCollateralParam, busdbCollateralParam, xrpbCollateralParam)
	oldDebtParam := oldGenState.Params.DebtParam

	newDebtParam := v0_11cdp.NewDebtParam(oldDebtParam.Denom, oldDebtParam.ReferenceAsset, oldDebtParam.ConversionFactor, oldDebtParam.DebtFloor, oldDebtParam.SavingsRate)

	newGlobalDebtLimit := oldGenState.Params.GlobalDebtLimit.Add(btcbCollateralParam.DebtLimit).Add(busdaCollateralParam.DebtLimit).Add(busdbCollateralParam.DebtLimit).Add(xrpbCollateralParam.DebtLimit)

	newParams := v0_11cdp.NewParams(newGlobalDebtLimit, newCollateralParams, newDebtParam, oldGenState.Params.SurplusAuctionThreshold, oldGenState.Params.SurplusAuctionLot, oldGenState.Params.DebtAuctionThreshold, oldGenState.Params.DebtAuctionLot, oldGenState.Params.SavingsDistributionFrequency, false)

	return v0_11cdp.NewGenesisState(
		newParams,
		newCDPs,
		newDeposits,
		newStartingID,
		oldGenState.DebtDenom,
		oldGenState.GovDenom,
		oldGenState.PreviousDistributionTime,
		sdk.ZeroInt(),
	)
}

// MigratePricefeed migrates from a v0.9 (or v0.10) pricefeed genesis state to a v0.11 pricefeed genesis state
func MigratePricefeed(oldGenState v0_9pricefeed.GenesisState) v0_11pricefeed.GenesisState {
	var newMarkets v0_11pricefeed.Markets
	var newPostedPrices v0_11pricefeed.PostedPrices
	var oracles []sdk.AccAddress

	for _, market := range oldGenState.Params.Markets {
		newMarket := v0_11pricefeed.NewMarket(market.MarketID, market.BaseAsset, market.QuoteAsset, market.Oracles, market.Active)
		newMarkets = append(newMarkets, newMarket)
		oracles = market.Oracles
	}
	// ------- add btc, xrp, busd markets --------
	btcSpotMarket := v0_11pricefeed.NewMarket("btc:usd", "btc", "usd", oracles, true)
	btcLiquidationMarket := v0_11pricefeed.NewMarket("btc:usd:30", "btc", "usd", oracles, true)
	xrpSpotMarket := v0_11pricefeed.NewMarket("xrp:usd", "xrp", "usd", oracles, true)
	xrpLiquidationMarket := v0_11pricefeed.NewMarket("xrp:usd:30", "xrp", "usd", oracles, true)
	busdSpotMarket := v0_11pricefeed.NewMarket("busd:usd", "busd", "usd", oracles, true)
	busdLiquidationMarket := v0_11pricefeed.NewMarket("busd:usd:30", "busd", "usd", oracles, true)
	newMarkets = append(newMarkets, btcSpotMarket, btcLiquidationMarket, xrpSpotMarket, xrpLiquidationMarket, busdSpotMarket, busdLiquidationMarket)

	for _, price := range oldGenState.PostedPrices {
		newPrice := v0_11pricefeed.NewPostedPrice(price.MarketID, price.OracleAddress, price.Price, price.Expiry)
		newPostedPrices = append(newPostedPrices, newPrice)
	}
	newParams := v0_11pricefeed.NewParams(newMarkets)

	return v0_11pricefeed.NewGenesisState(newParams, newPostedPrices)
}

// MigrateCommittee migrates from a v0.9 (or v0.10) committee genesis state to a v0.11 committee genesis state
func MigrateCommittee(oldGenState v0_9committee.GenesisState) v0_11committee.GenesisState {
	var newCommittees []v0_11committee.Committee
	var newStabilityCommittee v0_11committee.Committee
	var newSafetyCommittee v0_11committee.Committee
	var newProposals []v0_11committee.Proposal
	var newVotes []v0_11committee.Vote

	for _, committee := range oldGenState.Committees {
		if committee.ID == 1 {
			newStabilityCommittee.Description = committee.Description
			newStabilityCommittee.ID = committee.ID
			newStabilityCommittee.Members = committee.Members
			newStabilityCommittee.VoteThreshold = committee.VoteThreshold
			newStabilityCommittee.ProposalDuration = committee.ProposalDuration
			var newStabilityPermissions []v0_11committee.Permission
			var newStabilitySubParamPermissions v0_11committee.SubParamChangePermission
			for _, permission := range committee.Permissions {
				subPermission, ok := permission.(v0_9committee.SubParamChangePermission)
				if ok {
					oldCollateralParam := subPermission.AllowedCollateralParams[0]
					newCollateralParam := v0_11committee.AllowedCollateralParam{
						Type:                "bnb-a",
						Denom:               false,
						AuctionSize:         oldCollateralParam.AuctionSize,
						ConversionFactor:    oldCollateralParam.ConversionFactor,
						DebtLimit:           oldCollateralParam.DebtLimit,
						LiquidationMarketID: oldCollateralParam.LiquidationMarketID,
						SpotMarketID:        oldCollateralParam.SpotMarketID,
						LiquidationPenalty:  oldCollateralParam.LiquidationPenalty,
						LiquidationRatio:    oldCollateralParam.LiquidationRatio,
						Prefix:              oldCollateralParam.Prefix,
						StabilityFee:        oldCollateralParam.StabilityFee,
					}
					oldDebtParam := subPermission.AllowedDebtParam
					newDebtParam := v0_11committee.AllowedDebtParam{
						ConversionFactor: oldDebtParam.ConversionFactor,
						DebtFloor:        oldDebtParam.DebtFloor,
						Denom:            oldDebtParam.Denom,
						ReferenceAsset:   oldDebtParam.ReferenceAsset,
						SavingsRate:      oldDebtParam.SavingsRate,
					}
					oldAssetParam := subPermission.AllowedAssetParams[0]
					newAssetParam := v0_11committee.AllowedAssetParam{
						Active:        oldAssetParam.Active,
						CoinID:        oldAssetParam.CoinID,
						Denom:         oldAssetParam.Denom,
						Limit:         oldAssetParam.Limit,
						MaxSwapAmount: true,
						MinBlockLock:  true,
					}
					oldMarketParams := subPermission.AllowedMarkets
					var newMarketParams v0_11committee.AllowedMarkets
					for _, oldMarketParam := range oldMarketParams {
						newMarketParam := v0_11committee.AllowedMarket(oldMarketParam)
						newMarketParams = append(newMarketParams, newMarketParam)
					}
					oldAllowedParams := subPermission.AllowedParams
					var newAllowedParams v0_11committee.AllowedParams
					for _, oldAllowedParam := range oldAllowedParams {
						newAllowedParam := v0_11committee.AllowedParam(oldAllowedParam)
						if oldAllowedParam.Subspace == "bep3" && oldAllowedParam.Key == "SupportedAssets" {
							newAllowedParam.Key = "AssetParams"
						}

						newAllowedParams = append(newAllowedParams, newAllowedParam)
					}

					// --------------- ADD BUSD, XRP-B, BTC-B BEP3 parameters to Stability Committee Permissions
					busdAllowedAssetParam := v0_11committee.AllowedAssetParam{
						Active:        true,
						CoinID:        true, // allow busd coinID to be updated in case it gets its own slip-44
						Denom:         "busd",
						Limit:         true,
						MaxSwapAmount: true,
						MinBlockLock:  true,
					}
					xrpbAllowedAssetParam := v0_11committee.AllowedAssetParam{
						Active:        true,
						CoinID:        false,
						Denom:         "xrpb",
						Limit:         true,
						MaxSwapAmount: true,
						MinBlockLock:  true,
					}
					btcbAllowedAssetParam := v0_11committee.AllowedAssetParam{
						Active:        true,
						CoinID:        false,
						Denom:         "btcb",
						Limit:         true,
						MaxSwapAmount: true,
						MinBlockLock:  true,
					}
					// --------- ADD BTC-B, XRP-B, BUSD(a), BUSD(b) cdp collateral params to stability committee
					busdaAllowedCollateralParam := v0_11committee.NewAllowedCollateralParam(
						"busd-a", false, false, true, true, true, false, false, false, false, false,
					)
					busdbAllowedCollateralParam := v0_11committee.NewAllowedCollateralParam(
						"busd-b", false, false, true, true, true, false, false, false, false, false,
					)
					btcbAllowedCollateralParam := v0_11committee.NewAllowedCollateralParam(
						"btcb-a", false, false, true, true, true, false, false, false, false, false,
					)
					xrpbAllowedCollateralParam := v0_11committee.NewAllowedCollateralParam(
						"xrpb-a", false, false, true, true, true, false, false, false, false, false,
					)

					newStabilitySubParamPermissions.AllowedAssetParams = v0_11committee.AllowedAssetParams{
						newAssetParam, busdAllowedAssetParam, btcbAllowedAssetParam, xrpbAllowedAssetParam}
					newStabilitySubParamPermissions.AllowedCollateralParams = v0_11committee.AllowedCollateralParams{
						newCollateralParam, busdaAllowedCollateralParam, busdbAllowedCollateralParam, btcbAllowedCollateralParam, xrpbAllowedCollateralParam}
					newStabilitySubParamPermissions.AllowedDebtParam = newDebtParam
					newStabilitySubParamPermissions.AllowedMarkets = newMarketParams
					newStabilitySubParamPermissions.AllowedParams = newAllowedParams
					newStabilityPermissions = append(newStabilityPermissions, newStabilitySubParamPermissions)
				}
			}
			newStabilityPermissions = append(newStabilityPermissions, v0_11committee.TextPermission{})
			newStabilityCommittee.Permissions = newStabilityPermissions
			newCommittees = append(newCommittees, newStabilityCommittee)
		} else {
			newSafetyCommittee.ID = committee.ID
			newSafetyCommittee.Description = committee.Description
			newSafetyCommittee.Members = committee.Members
			newSafetyCommittee.Permissions = []v0_11committee.Permission{v0_11committee.SoftwareUpgradePermission{}}
			newSafetyCommittee.VoteThreshold = committee.VoteThreshold
			newSafetyCommittee.ProposalDuration = committee.ProposalDuration
			newCommittees = append(newCommittees, newSafetyCommittee)
		}
	}
	for _, oldProp := range oldGenState.Proposals {
		newPubProposal := v0_11committee.PubProposal(oldProp.PubProposal)
		newProp := v0_11committee.NewProposal(newPubProposal, oldProp.ID, oldProp.CommitteeID, oldProp.Deadline)
		newProposals = append(newProposals, newProp)
	}

	for _, oldVote := range oldGenState.Votes {
		newVote := v0_11committee.NewVote(oldVote.ProposalID, oldVote.Voter)
		newVotes = append(newVotes, newVote)
	}

	return v0_11committee.GenesisState{
		NextProposalID: oldGenState.NextProposalID,
		Committees:     newCommittees,
		Proposals:      newProposals,
		Votes:          newVotes,
	}
}
