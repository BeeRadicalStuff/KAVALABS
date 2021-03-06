package v0_13

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/supply"
)

// Defaults for auction params
const (
	// DefaultMaxAuctionDuration max length of auction
	DefaultMaxAuctionDuration time.Duration = 2 * 24 * time.Hour
	// DefaultBidDuration how long an auction gets extended when someone bids
	DefaultBidDuration    time.Duration = 1 * time.Hour
	CollateralAuctionType               = "collateral"
	SurplusAuctionType                  = "surplus"
	DebtAuctionType                     = "debt"
	ForwardAuctionPhase                 = "forward"
	ReverseAuctionPhase                 = "reverse"
	DefaultNextAuctionID  uint64        = 1
)

// module variables
var (
	// DefaultIncrement is the smallest percent change a new bid must have from the old one
	DefaultIncrement       sdk.Dec = sdk.MustNewDecFromStr("0.05")
	KeyBidDuration                 = []byte("BidDuration")
	KeyMaxAuctionDuration          = []byte("MaxAuctionDuration")
	KeyIncrementSurplus            = []byte("IncrementSurplus")
	KeyIncrementDebt               = []byte("IncrementDebt")
	KeyIncrementCollateral         = []byte("IncrementCollateral")
	emptyDec                       = sdk.Dec{}
)

// GenesisAuction interface for auctions at genesis
type GenesisAuction interface {
	Auction
	GetModuleAccountCoins() sdk.Coins
	Validate() error
}

// GenesisAuctions is a slice of genesis auctions.
type GenesisAuctions []GenesisAuction

// GenesisState is auction state that must be provided at chain genesis.
type GenesisState struct {
	NextAuctionID uint64          `json:"next_auction_id" yaml:"next_auction_id"`
	Params        Params          `json:"params" yaml:"params"`
	Auctions      GenesisAuctions `json:"auctions" yaml:"auctions"`
}

// NewGenesisState returns a new genesis state object for auctions module.
func NewGenesisState(nextID uint64, ap Params, ga GenesisAuctions) GenesisState {
	return GenesisState{
		NextAuctionID: nextID,
		Params:        ap,
		Auctions:      ga,
	}
}

// DefaultGenesisState returns the default genesis state for auction module.
func DefaultGenesisState() GenesisState {
	return NewGenesisState(
		DefaultNextAuctionID,
		DefaultParams(),
		GenesisAuctions{},
	)
}

// Validate validates genesis inputs. It returns error if validation of any input fails.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	ids := map[uint64]bool{}
	for _, a := range gs.Auctions {

		if err := a.Validate(); err != nil {
			return fmt.Errorf("found invalid auction: %w", err)
		}

		if ids[a.GetID()] {
			return fmt.Errorf("found duplicate auction ID (%d)", a.GetID())
		}
		ids[a.GetID()] = true

		if a.GetID() >= gs.NextAuctionID {
			return fmt.Errorf("found auction ID ??? the nextAuctionID (%d ??? %d)", a.GetID(), gs.NextAuctionID)
		}
	}
	return nil
}

// Params is the governance parameters for the auction module.
type Params struct {
	MaxAuctionDuration  time.Duration `json:"max_auction_duration" yaml:"max_auction_duration"` // max length of auction
	BidDuration         time.Duration `json:"bid_duration" yaml:"bid_duration"`                 // additional time added to the auction end time after each bid, capped by the expiry.
	IncrementSurplus    sdk.Dec       `json:"increment_surplus" yaml:"increment_surplus"`       // percentage change (of auc.Bid) required for a new bid on a surplus auction
	IncrementDebt       sdk.Dec       `json:"increment_debt" yaml:"increment_debt"`             // percentage change (of auc.Lot) required for a new bid on a debt auction
	IncrementCollateral sdk.Dec       `json:"increment_collateral" yaml:"increment_collateral"` // percentage change (of auc.Bid or auc.Lot) required for a new bid on a collateral auction
}

// NewParams returns a new Params object.
func NewParams(maxAuctionDuration, bidDuration time.Duration, incrementSurplus, incrementDebt, incrementCollateral sdk.Dec) Params {
	return Params{
		MaxAuctionDuration:  maxAuctionDuration,
		BidDuration:         bidDuration,
		IncrementSurplus:    incrementSurplus,
		IncrementDebt:       incrementDebt,
		IncrementCollateral: incrementCollateral,
	}
}

// DefaultParams returns the default parameters for auctions.
func DefaultParams() Params {
	return NewParams(
		DefaultMaxAuctionDuration,
		DefaultBidDuration,
		DefaultIncrement,
		DefaultIncrement,
		DefaultIncrement,
	)
}

// Validate checks that the parameters have valid values.
func (p Params) Validate() error {
	if err := validateBidDurationParam(p.BidDuration); err != nil {
		return err
	}

	if err := validateMaxAuctionDurationParam(p.MaxAuctionDuration); err != nil {
		return err
	}

	if p.BidDuration > p.MaxAuctionDuration {
		return errors.New("bid duration param cannot be larger than max auction duration")
	}

	if err := validateIncrementSurplusParam(p.IncrementSurplus); err != nil {
		return err
	}

	if err := validateIncrementDebtParam(p.IncrementDebt); err != nil {
		return err
	}

	return validateIncrementCollateralParam(p.IncrementCollateral)
}

func validateBidDurationParam(i interface{}) error {
	bidDuration, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if bidDuration < 0 {
		return fmt.Errorf("bid duration cannot be negative %d", bidDuration)
	}

	return nil
}

func validateMaxAuctionDurationParam(i interface{}) error {
	maxAuctionDuration, ok := i.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if maxAuctionDuration < 0 {
		return fmt.Errorf("max auction duration cannot be negative %d", maxAuctionDuration)
	}

	return nil
}

func validateIncrementSurplusParam(i interface{}) error {
	incrementSurplus, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if incrementSurplus == emptyDec || incrementSurplus.IsNil() {
		return errors.New("surplus auction increment cannot be nil or empty")
	}

	if incrementSurplus.IsNegative() {
		return fmt.Errorf("surplus auction increment cannot be less than zero %s", incrementSurplus)
	}

	return nil
}

func validateIncrementDebtParam(i interface{}) error {
	incrementDebt, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if incrementDebt == emptyDec || incrementDebt.IsNil() {
		return errors.New("debt auction increment cannot be nil or empty")
	}

	if incrementDebt.IsNegative() {
		return fmt.Errorf("debt auction increment cannot be less than zero %s", incrementDebt)
	}

	return nil
}

func validateIncrementCollateralParam(i interface{}) error {
	incrementCollateral, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if incrementCollateral == emptyDec || incrementCollateral.IsNil() {
		return errors.New("collateral auction increment cannot be nil or empty")
	}

	if incrementCollateral.IsNegative() {
		return fmt.Errorf("collateral auction increment cannot be less than zero %s", incrementCollateral)
	}

	return nil
}

// Auction is an interface for handling common actions on auctions.
type Auction interface {
	GetID() uint64
	WithID(uint64) Auction

	GetInitiator() string
	GetLot() sdk.Coin
	GetBidder() sdk.AccAddress
	GetBid() sdk.Coin
	GetEndTime() time.Time

	GetType() string
	GetPhase() string
}

// Auctions is a slice of auctions.
type Auctions []Auction

// BaseAuction is a common type shared by all Auctions.
type BaseAuction struct {
	ID              uint64         `json:"id" yaml:"id"`
	Initiator       string         `json:"initiator" yaml:"initiator"`                 // Module name that starts the auction. Pays out Lot.
	Lot             sdk.Coin       `json:"lot" yaml:"lot"`                             // Coins that will paid out by Initiator to the winning bidder.
	Bidder          sdk.AccAddress `json:"bidder" yaml:"bidder"`                       // Latest bidder. Receiver of Lot.
	Bid             sdk.Coin       `json:"bid" yaml:"bid"`                             // Coins paid into the auction the bidder.
	HasReceivedBids bool           `json:"has_received_bids" yaml:"has_received_bids"` // Whether the auction has received any bids or not.
	EndTime         time.Time      `json:"end_time" yaml:"end_time"`                   // Current auction closing time. Triggers at the end of the block with time ??? EndTime.
	MaxEndTime      time.Time      `json:"max_end_time" yaml:"max_end_time"`           // Maximum closing time. Auctions can close before this but never after.
}

// GetID is a getter for auction ID.
func (a BaseAuction) GetID() uint64 { return a.ID }

// GetInitiator is a getter for auction Initiator.
func (a BaseAuction) GetInitiator() string { return a.Initiator }

// GetLot is a getter for auction Lot.
func (a BaseAuction) GetLot() sdk.Coin { return a.Lot }

// GetBidder is a getter for auction Bidder.
func (a BaseAuction) GetBidder() sdk.AccAddress { return a.Bidder }

// GetBid is a getter for auction Bid.
func (a BaseAuction) GetBid() sdk.Coin { return a.Bid }

// GetEndTime is a getter for auction end time.
func (a BaseAuction) GetEndTime() time.Time { return a.EndTime }

// GetType returns the auction type. Used to identify auctions in event attributes.
func (a BaseAuction) GetType() string { return "base" }

// Validate verifies that the auction end time is before max end time
func (a BaseAuction) Validate() error {
	// ID can be 0 for surplus, debt and collateral auctions
	if strings.TrimSpace(a.Initiator) == "" {
		return errors.New("auction initiator cannot be blank")
	}
	if !a.Lot.IsValid() {
		return fmt.Errorf("invalid lot: %s", a.Lot)
	}
	// NOTE: bidder can be empty for Surplus and Collateral auctions
	if !a.Bidder.Empty() && len(a.Bidder) != sdk.AddrLen {
		return fmt.Errorf("the expected bidder address length is %d, actual length is %d", sdk.AddrLen, len(a.Bidder))
	}
	if !a.Bid.IsValid() {
		return fmt.Errorf("invalid bid: %s", a.Bid)
	}
	if a.EndTime.IsZero() || a.MaxEndTime.IsZero() {
		return errors.New("end time cannot be zero")
	}
	if a.EndTime.After(a.MaxEndTime) {
		return fmt.Errorf("MaxEndTime < EndTime (%s < %s)", a.MaxEndTime, a.EndTime)
	}
	return nil
}

func (a BaseAuction) String() string {
	return fmt.Sprintf(`Auction %d:
  Initiator:              %s
  Lot:               			%s
  Bidder:            		  %s
  Bid:        						%s
  End Time:   						%s
  Max End Time:      			%s`,
		a.GetID(), a.Initiator, a.Lot,
		a.Bidder, a.Bid, a.GetEndTime().String(),
		a.MaxEndTime.String(),
	)
}

// SurplusAuction is a forward auction that burns what it receives from bids.
// It is normally used to sell off excess pegged asset acquired by the CDP system.
type SurplusAuction struct {
	BaseAuction `json:"base_auction" yaml:"base_auction"`
}

// WithID returns an auction with the ID set.
func (a SurplusAuction) WithID(id uint64) Auction { a.ID = id; return a }

// GetType returns the auction type. Used to identify auctions in event attributes.
func (a SurplusAuction) GetType() string { return SurplusAuctionType }

// GetModuleAccountCoins returns the total number of coins held in the module account for this auction.
// It is used in genesis initialize the module account correctly.
func (a SurplusAuction) GetModuleAccountCoins() sdk.Coins {
	// a.Bid is paid out on bids, so is never stored in the module account
	return sdk.NewCoins(a.Lot)
}

// GetPhase returns the direction of a surplus auction, which never changes.
func (a SurplusAuction) GetPhase() string { return ForwardAuctionPhase }

// NewSurplusAuction returns a new surplus auction.
func NewSurplusAuction(seller string, lot sdk.Coin, bidDenom string, endTime time.Time) SurplusAuction {
	auction := SurplusAuction{BaseAuction{
		// no ID
		Initiator:       seller,
		Lot:             lot,
		Bidder:          nil,
		Bid:             sdk.NewInt64Coin(bidDenom, 0),
		HasReceivedBids: false, // new auctions don't have any bids
		EndTime:         endTime,
		MaxEndTime:      endTime,
	}}
	return auction
}

// DebtAuction is a reverse auction that mints what it pays out.
// It is normally used to acquire pegged asset to cover the CDP system's debts that were not covered by selling collateral.
type DebtAuction struct {
	BaseAuction `json:"base_auction" yaml:"base_auction"`

	CorrespondingDebt sdk.Coin `json:"corresponding_debt" yaml:"corresponding_debt"`
}

// WithID returns an auction with the ID set.
func (a DebtAuction) WithID(id uint64) Auction { a.ID = id; return a }

// GetType returns the auction type. Used to identify auctions in event attributes.
func (a DebtAuction) GetType() string { return DebtAuctionType }

// GetModuleAccountCoins returns the total number of coins held in the module account for this auction.
// It is used in genesis initialize the module account correctly.
func (a DebtAuction) GetModuleAccountCoins() sdk.Coins {
	// a.Lot is minted at auction close, so is never stored in the module account
	// a.Bid is paid out on bids, so is never stored in the module account
	return sdk.NewCoins(a.CorrespondingDebt)
}

// GetPhase returns the direction of a debt auction, which never changes.
func (a DebtAuction) GetPhase() string { return ReverseAuctionPhase }

// Validate validates the DebtAuction fields values.
func (a DebtAuction) Validate() error {
	if !a.CorrespondingDebt.IsValid() {
		return fmt.Errorf("invalid corresponding debt: %s", a.CorrespondingDebt)
	}
	return a.BaseAuction.Validate()
}

// NewDebtAuction returns a new debt auction.
func NewDebtAuction(buyerModAccName string, bid sdk.Coin, initialLot sdk.Coin, endTime time.Time, debt sdk.Coin) DebtAuction {
	// Note: Bidder is set to the initiator's module account address instead of module name. (when the first bid is placed, it is paid out to the initiator)
	// Setting to the module account address bypasses calling supply.SendCoinsFromModuleToModule, instead calls SendCoinsFromModuleToAccount.
	// This isn't a problem currently, but if additional logic/validation was added for sending to coins to Module Accounts, it would be bypassed.
	auction := DebtAuction{
		BaseAuction: BaseAuction{
			// no ID
			Initiator:       buyerModAccName,
			Lot:             initialLot,
			Bidder:          supply.NewModuleAddress(buyerModAccName), // send proceeds from the first bid to the buyer.
			Bid:             bid,                                      // amount that the buyer is buying - doesn't change over course of auction
			HasReceivedBids: false,                                    // new auctions don't have any bids
			EndTime:         endTime,
			MaxEndTime:      endTime,
		},
		CorrespondingDebt: debt,
	}
	return auction
}

// CollateralAuction is a two phase auction.
// Initially, in forward auction phase, bids can be placed up to a max bid.
// Then it switches to a reverse auction phase, where the initial amount up for auction is bid down.
// Unsold Lot is sent to LotReturns, being divided among the addresses by weight.
// Collateral auctions are normally used to sell off collateral seized from CDPs.
type CollateralAuction struct {
	BaseAuction `json:"base_auction" yaml:"base_auction"`

	CorrespondingDebt sdk.Coin          `json:"corresponding_debt" yaml:"corresponding_debt"`
	MaxBid            sdk.Coin          `json:"max_bid" yaml:"max_bid"`
	LotReturns        WeightedAddresses `json:"lot_returns" yaml:"lot_returns"`
}

// WithID returns an auction with the ID set.
func (a CollateralAuction) WithID(id uint64) Auction { a.ID = id; return a }

// GetType returns the auction type. Used to identify auctions in event attributes.
func (a CollateralAuction) GetType() string { return CollateralAuctionType }

// GetModuleAccountCoins returns the total number of coins held in the module account for this auction.
// It is used in genesis initialize the module account correctly.
func (a CollateralAuction) GetModuleAccountCoins() sdk.Coins {
	// a.Bid is paid out on bids, so is never stored in the module account
	return sdk.NewCoins(a.Lot).Add(sdk.NewCoins(a.CorrespondingDebt)...)
}

// IsReversePhase returns whether the auction has switched over to reverse phase or not.
// CollateralAuctions initially start in forward phase.
func (a CollateralAuction) IsReversePhase() bool {
	return a.Bid.IsEqual(a.MaxBid)
}

// GetPhase returns the direction of a collateral auction.
func (a CollateralAuction) GetPhase() string {
	if a.IsReversePhase() {
		return ReverseAuctionPhase
	}
	return ForwardAuctionPhase
}

// GetLotReturns returns a collateral auction's lot owners
func (a CollateralAuction) GetLotReturns() WeightedAddresses {
	return a.LotReturns
}

// Validate validates the CollateralAuction fields values.
func (a CollateralAuction) Validate() error {
	if !a.CorrespondingDebt.IsValid() {
		return fmt.Errorf("invalid corresponding debt: %s", a.CorrespondingDebt)
	}
	if !a.MaxBid.IsValid() {
		return fmt.Errorf("invalid max bid: %s", a.MaxBid)
	}
	if err := a.LotReturns.Validate(); err != nil {
		return fmt.Errorf("invalid lot returns: %w", err)
	}
	return a.BaseAuction.Validate()
}

func (a CollateralAuction) String() string {
	return fmt.Sprintf(`Auction %d:
  Initiator:              %s
  Lot:               			%s
  Bidder:            		  %s
  Bid:        						%s
  End Time:   						%s
	Max End Time:      			%s
	Max Bid									%s
	LotReturns						%s
	Corresponding Debt %s`,
		a.GetID(), a.Initiator, a.Lot,
		a.Bidder, a.Bid, a.GetEndTime().String(),
		a.MaxEndTime.String(), a.MaxBid, a.LotReturns, a.CorrespondingDebt,
	)
}

// NewCollateralAuction returns a new collateral auction.
func NewCollateralAuction(seller string, lot sdk.Coin, endTime time.Time, maxBid sdk.Coin, lotReturns WeightedAddresses, debt sdk.Coin) CollateralAuction {
	auction := CollateralAuction{
		BaseAuction: BaseAuction{
			// no ID
			Initiator:       seller,
			Lot:             lot,
			Bidder:          nil,
			Bid:             sdk.NewInt64Coin(maxBid.Denom, 0),
			HasReceivedBids: false, // new auctions don't have any bids
			EndTime:         endTime,
			MaxEndTime:      endTime},
		CorrespondingDebt: debt,
		MaxBid:            maxBid,
		LotReturns:        lotReturns,
	}
	return auction
}

// WeightedAddresses is a type for storing some addresses and associated weights.
type WeightedAddresses struct {
	Addresses []sdk.AccAddress `json:"addresses" yaml:"addresses"`
	Weights   []sdk.Int        `json:"weights" yaml:"weights"`
}

// NewWeightedAddresses returns a new list addresses with weights.
func NewWeightedAddresses(addrs []sdk.AccAddress, weights []sdk.Int) (WeightedAddresses, error) {
	wa := WeightedAddresses{
		Addresses: addrs,
		Weights:   weights,
	}
	if err := wa.Validate(); err != nil {
		return WeightedAddresses{}, err
	}
	return wa, nil
}

// Validate checks for that the weights are not negative, not all zero, and the lengths match.
func (wa WeightedAddresses) Validate() error {
	if len(wa.Weights) < 1 {
		return fmt.Errorf("must be at least 1 weighted address")
	}

	if len(wa.Addresses) != len(wa.Weights) {
		return fmt.Errorf("number of addresses doesn't match number of weights, %d ??? %d", len(wa.Addresses), len(wa.Weights))
	}

	totalWeight := sdk.ZeroInt()
	for i := range wa.Addresses {
		if wa.Addresses[i].Empty() {
			return fmt.Errorf("address %d cannot be empty", i)
		}
		if len(wa.Addresses[i]) != sdk.AddrLen {
			return fmt.Errorf("address %d has an invalid length: expected %d, got %d", i, sdk.AddrLen, len(wa.Addresses[i]))
		}
		if wa.Weights[i].IsNegative() {
			return fmt.Errorf("weight %d contains a negative amount: %s", i, wa.Weights[i])
		}
		totalWeight = totalWeight.Add(wa.Weights[i])
	}

	if !totalWeight.IsPositive() {
		return fmt.Errorf("total weight must be positive")
	}

	return nil
}
