package feemarket

const (
	ModuleName = "feemarket"
	StoreKey   = "feemarket"
)

type Params struct {
	MinGasPrice          uint64 `json:"min_gas_price"`
	BaseFee              uint64 `json:"base_fee"`
	ElasticityMultiplier uint32 `json:"elasticity_multiplier"`
	EnableHeight         int64  `json:"enable_height"`
}
