package precompiles

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethvm "github.com/ethereum/go-ethereum/core/vm"

	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/sovereign-l1/chain/x/milestone"
)

const MilestonePrecompileABIString = `[
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "id",
				"type": "string"
			}
		],
		"name": "getMilestone",
		"outputs": [
			{
				"internalType": "string",
				"name": "milestoneID",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "feedID",
				"type": "string"
			},
			{
				"internalType": "uint64",
				"name": "targetPrice",
				"type": "uint64"
			},
			{
				"internalType": "int64",
				"name": "remainingBlocks",
				"type": "int64"
			},
			{
				"internalType": "string",
				"name": "state",
				"type": "string"
			},
			{
				"internalType": "string",
				"name": "vestingPoolAddress",
				"type": "string"
			}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

var (
	MilestonePrecompileABI     abi.ABI
	MilestonePrecompileAddress = common.HexToAddress("0x0000000000000000000000000000000000000102")
)

type MilestonePrecompile struct {
	keeper milestone.Keeper
}

func init() {
	var err error
	MilestonePrecompileABI, err = abi.JSON(strings.NewReader(MilestonePrecompileABIString))
	if err != nil {
		panic(err)
	}
}

func NewMilestonePrecompile(k milestone.Keeper) *MilestonePrecompile {
	return &MilestonePrecompile{
		keeper: k,
	}
}

func (mp *MilestonePrecompile) Address() common.Address {
	return MilestonePrecompileAddress
}

func (mp *MilestonePrecompile) RequiredGas(input []byte) uint64 {
	return 2000
}

func (mp *MilestonePrecompile) Name() string {
	return "MilestonePrecompile"
}

func (mp *MilestonePrecompile) Run(evm *ethvm.EVM, contract *ethvm.Contract, readonly bool) ([]byte, error) {
	if len(contract.Input) < 4 {
		return nil, fmt.Errorf("invalid input length: %d", len(contract.Input))
	}

	methodID := contract.Input[:4]
	method, err := MilestonePrecompileABI.MethodById(methodID)
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
	case "getMilestone":
		id := args[0].(string)
		m, found := mp.keeper.GetMilestone(ctx, id)
		if !found {
			return nil, fmt.Errorf("milestone %s not found", id)
		}
		res, err := method.Outputs.Pack(m.ID, m.FeedID, m.TargetPrice, m.RemainingBlocks, m.State, m.VestingPoolAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to pack output: %w", err)
		}
		return res, nil

	default:
		return nil, fmt.Errorf("unsupported method: %s", method.Name)
	}
}
