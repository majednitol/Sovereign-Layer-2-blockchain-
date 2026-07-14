// TODO: post-launch — custom precompiles should not ship in mainnet v1.0 binary
package precompiles

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethvm "github.com/ethereum/go-ethereum/core/vm"

	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/sovereign-l1/chain/x/oracle"
)

const OraclePrecompileABIString = `[
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "feedID",
				"type": "string"
			}
		],
		"name": "getLatestPrice",
		"outputs": [
			{
				"internalType": "uint64",
				"name": "price",
				"type": "uint64"
			},
			{
				"internalType": "int64",
				"name": "blockHeight",
				"type": "int64"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "feedID",
				"type": "string"
			}
		],
		"name": "isFeedStale",
		"outputs": [
			{
				"internalType": "bool",
				"name": "stale",
				"type": "bool"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

var (
	OraclePrecompileABI     abi.ABI
	OraclePrecompileAddress = common.HexToAddress("0x0000000000000000000000000000000000000801")
)

type OraclePrecompile struct {
	keeper oracle.Keeper
}

func init() {
	var err error
	OraclePrecompileABI, err = abi.JSON(strings.NewReader(OraclePrecompileABIString))
	if err != nil {
		panic(err)
	}
}

func NewOraclePrecompile(k oracle.Keeper) *OraclePrecompile {
	return &OraclePrecompile{
		keeper: k,
	}
}

func (op *OraclePrecompile) Address() common.Address {
	return OraclePrecompileAddress
}

func (op *OraclePrecompile) RequiredGas(input []byte) uint64 {
	return 2000
}

func (op *OraclePrecompile) Name() string {
	return "OraclePrecompile"
}

func (op *OraclePrecompile) Run(evm *ethvm.EVM, contract *ethvm.Contract, readonly bool) ([]byte, error) {
	if len(contract.Input) < 4 {
		return nil, fmt.Errorf("invalid input length: %d", len(contract.Input))
	}

	methodID := contract.Input[:4]
	method, err := OraclePrecompileABI.MethodById(methodID)
	if err != nil {
		return nil, fmt.Errorf("method not found: %w", err)
	}

	stateDB, ok := evm.StateDB.(*statedb.StateDB)
	if !ok {
		return nil, fmt.Errorf("invalid stateDB type: %T", evm.StateDB)
	}
	ctx := stateDB.GetContext()

	args, err := method.Inputs.Unpack(contract.Input[4:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack input arguments: %w", err)
	}

	switch method.Name {
	case "getLatestPrice":
		feedID := args[0].(string)
		price, height, err := op.keeper.GetLatestPrice(ctx, feedID)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest price: %w", err)
		}
		res, err := method.Outputs.Pack(price, height)
		if err != nil {
			return nil, fmt.Errorf("failed to pack output: %w", err)
		}
		return res, nil

	case "isFeedStale":
		feedID := args[0].(string)
		stale := op.keeper.IsFeedStale(ctx, feedID)
		res, err := method.Outputs.Pack(stale)
		if err != nil {
			return nil, fmt.Errorf("failed to pack output: %w", err)
		}
		return res, nil

	default:
		return nil, fmt.Errorf("unsupported method: %s", method.Name)
	}
}
