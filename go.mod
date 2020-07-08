module github.com/kava-labs/kava

go 1.13

require (
	github.com/cosmos/cosmos-sdk v0.38.5
	github.com/gogo/protobuf v1.3.1
	github.com/gorilla/mux v1.7.4
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.5.1
	github.com/tendermint/tendermint v0.33.6
	github.com/tendermint/tm-db v0.5.1
	gopkg.in/yaml.v2 v2.3.0
)

// patch bech32 decoding to enable larger string lengths
replace github.com/btcsuite/btcutil => github.com/kava-labs/btcutil v0.0.0-20200522184203-886d33430f06

replace github.com/cosmos/cosmos-sdk => github.com/cosmos/cosmos-sdk v0.34.4-0.20200528144628-f8bad078b7b3
