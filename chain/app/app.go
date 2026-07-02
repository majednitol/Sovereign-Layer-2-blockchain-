package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	abci "github.com/cometbft/cometbft/abci/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"cosmossdk.io/log/v2"
	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/tx/signing"
	"github.com/cosmos/gogoproto/proto"
	evmaddress "github.com/cosmos/evm/encoding/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	client "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	// Cosmos EVM — EIP-1559 fee market
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	"github.com/cosmos/evm/x/feemarket"

	// Cosmos EVM — EVM (x/vm)
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	evmcodec "github.com/cosmos/evm/crypto/codec"
	"github.com/cosmos/evm/x/vm"

	// Cosmos EVM — ERC-20 bridge
	erc20keeper "github.com/cosmos/evm/x/erc20/keeper"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/cosmos/evm/x/erc20"

	// Cosmos EVM — unified ante handler
	evmante "github.com/cosmos/evm/ante"
	antetypes "github.com/cosmos/evm/ante/types"
	"google.golang.org/protobuf/reflect/protoreflect"

	// Cosmos EVM — mempool
	evmmempool "github.com/cosmos/evm/mempool"
	evmserver "github.com/cosmos/evm/server"

	// IBC
	ibckeeper "github.com/cosmos/ibc-go/v11/modules/core/keeper"
	ibcexported "github.com/cosmos/ibc-go/v11/modules/core/exported"
	ibctransfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	transferkeeper "github.com/cosmos/ibc-go/v11/modules/apps/transfer/keeper"
	porttypes "github.com/cosmos/ibc-go/v11/modules/core/05-port/types"
	transfer "github.com/cosmos/ibc-go/v11/modules/apps/transfer"
	ibc "github.com/cosmos/ibc-go/v11/modules/core"
	ibctm "github.com/cosmos/ibc-go/v11/modules/light-clients/07-tendermint"
	ibcfee "github.com/cosmos/ibc-go/v11/modules/apps/fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v11/modules/apps/fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v11/modules/apps/fee/types"

	// CosmWasm
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/spf13/cast"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/std"
)

const Name = "sovereign"

// EVMChainID is the integer chain ID used by the EVM (registered on chainlist.org per ADR-009)
const EVMChainID = uint64(7777)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager that is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		distribution.AppModuleBasic{},
		slashing.AppModuleBasic{},
		gov.NewAppModuleBasic(getGovProposalHandlers()),
		upgrade.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		consensus.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		// IBC
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		transfer.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
		// Cosmos EVM
		feemarket.AppModuleBasic{},
		vm.AppModuleBasic{},
		erc20.AppModuleBasic{},
		// CosmWasm
		wasm.AppModuleBasic{},
	)

	// maccPerms is a mapping of module account names to their permission flags.
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		ibcfeetypes.ModuleName:        nil,
		evmtypes.ModuleName:            {authtypes.Minter, authtypes.Burner},
		erc20types.ModuleName:          {authtypes.Minter, authtypes.Burner},
		feemarkettypes.ModuleName:      nil,
		wasm.ModuleName:                {authtypes.Burner},
	}
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, "."+Name)
}

// App extends an ABCI application, but with most of its parameters exportable.
type App struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry
	txConfig          client.TxConfig

	// store keys
	keys map[string]*storetypes.KVStoreKey

	// Standard Cosmos SDK keepers
	AccountKeeper  authkeeper.AccountKeeper
	BankKeeper     bankkeeper.Keeper
	StakingKeeper  *stakingkeeper.Keeper
	SlashingKeeper slashingkeeper.Keeper
	DistrKeeper    distrkeeper.Keeper
	GovKeeper      *govkeeper.Keeper
	UpgradeKeeper  *upgradekeeper.Keeper
	FeeGrantKeeper feegrantkeeper.Keeper
	AuthzKeeper    authzkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// IBC keepers
	IBCKeeper      *ibckeeper.Keeper
	TransferKeeper *transferkeeper.Keeper
	IBCFeeKeeper   ibcfeekeeper.Keeper

	// Cosmos EVM keepers
	FeeMarketKeeper feemarketkeeper.Keeper
	EVMKeeper       *evmkeeper.Keeper
	Erc20Keeper     erc20keeper.Keeper

	// CosmWasm
	WasmKeeper wasmkeeper.Keeper

	// the module manager
	mm *module.Manager

	// configurator for upgrade handler RunMigrations
	Configurator module.Configurator
}

type MapAppOptions map[string]interface{}

func (m MapAppOptions) Get(key string) interface{} {
	return m[key]
}

// NewApp returns a reference to an initialized Sovereign App.
func NewApp(
	logger io.Writer,
	db interface{},
	traceStore io.Writer,
	loadLatest bool,
	appOpts map[string]interface{},
	baseAppOptions ...func(*baseapp.BaseApp),
) *App {
	sdk.DefaultPowerReduction = math.NewInt(1_000_000)
	signingOptions := signing.Options{
		AddressCodec:          evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		CustomGetSigners: map[protoreflect.FullName]signing.GetSignersFunc{
			evmtypes.MsgEthereumTxCustomGetSigner.MsgType:     evmtypes.MsgEthereumTxCustomGetSigner.Fn,
			erc20types.MsgConvertERC20CustomGetSigner.MsgType: erc20types.MsgConvertERC20CustomGetSigner.Fn,
		},
	}
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles:     proto.HybridResolver,
		SigningOptions: signingOptions,
	})
	if err != nil {
		panic(err)
	}
	std.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	evmcodec.RegisterInterfaces(interfaceRegistry)
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)
	legacyAmino := codec.NewLegacyAmino()
	evmcodec.RegisterCrypto(legacyAmino)

	// Initialize KV store keys
	keys := storetypes.NewKVStoreKeys(
		// Standard SDK
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		upgradetypes.StoreKey,
		feegrant.StoreKey,
		authzkeeper.StoreKey,
		consensusparamtypes.StoreKey,
		// IBC
		ibcexported.StoreKey,
		ibctransfertypes.StoreKey,
		ibcfeetypes.StoreKey,
		// Cosmos EVM
		evmtypes.StoreKey,
		feemarkettypes.StoreKey,
		erc20types.StoreKey,
		// CosmWasm
		wasm.StoreKey,
	)

	// Object store keys (transient per-block data, reset on every Commit)
	oKeys := storetypes.NewObjectStoreKeys(banktypes.ObjectStoreKey, evmtypes.ObjectKey)

	// Initialize BaseApp
	var sdkLogger log.Logger
	if logger != nil {
		sdkLogger = log.NewLogger(logger)
	} else {
		sdkLogger = log.NewNopLogger()
	}

	var sdkDb dbm.DB
	if db != nil {
		if d, ok := db.(dbm.DB); ok {
			sdkDb = d
		}
	}

	bApp := baseapp.NewBaseApp(Name, sdkLogger, sdkDb, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	app := &App{
		BaseApp:           bApp,
		appCodec:          appCodec,
		legacyAmino:       legacyAmino,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		txConfig:          txConfig,
	}

	// Governance authority address
	authAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// --- Standard SDK Keepers ---
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authAddr,
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		BlockedAddresses(),
		authAddr,
		sdkLogger,
	)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authAddr,
		addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authAddr,
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		authAddr,
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[feegrant.StoreKey]),
		app.AccountKeeper,
	)

	// Register staking hooks (DistrKeeper + SlashingKeeper)
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			app.DistrKeeper.Hooks(),
			app.SlashingKeeper.Hooks(),
		),
	)

	app.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[authzkeeper.StoreKey]),
		appCodec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
	)

	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
		authAddr,
		runtime.EventService{},
	)
	app.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		map[int64]bool{},
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		DefaultNodeHome,
		app.BaseApp,
		authAddr,
	)

	// --- IBC Keepers ---
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibcexported.StoreKey]),
		app.UpgradeKeeper,
		authAddr,
	)

	app.TransferKeeper = transferkeeper.NewKeeper(
		appCodec,
		nil, // addressCodec — set nil for scaffold; will wire EvmCodec in full production
		runtime.NewKVStoreService(keys[ibctransfertypes.StoreKey]),
		app.IBCKeeper.ChannelKeeper,
		app.MsgServiceRouter(),
		app.AccountKeeper,
		app.BankKeeper,
		authAddr,
	)

	app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibcfeetypes.StoreKey]),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	// --- Cosmos EVM Keepers (order matters: feemarket → vm → erc20) ---
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		keys[feemarkettypes.StoreKey],
	)

	app.EVMKeeper = evmkeeper.NewKeeper(
		appCodec,
		keys[evmtypes.StoreKey],
		oKeys[evmtypes.ObjectKey],
		nil, // nonTransientKeys — not needed without transient stores
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.FeeMarketKeeper,
		app.ConsensusParamsKeeper, // ConsensusParamsKeeper
		&app.Erc20Keeper,
		EVMChainID,
		"", // tracer
	)

	app.Erc20Keeper = erc20keeper.NewKeeper(
		keys[erc20types.StoreKey],
		appCodec,
		authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.EVMKeeper,
		app.StakingKeeper,
		app.TransferKeeper,
	)

	// --- Set up IBC transfer stack (transfer → ibcfee → erc20 middleware) ---
	var transferStack porttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)
	transferStack = erc20.NewIBCMiddleware(app.Erc20Keeper, transferStack)

	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
	app.IBCKeeper.SetRouter(ibcRouter)

	// Light client module (Tendermint)
	clientKeeper := app.IBCKeeper.ClientKeeper
	storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()
	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)

	// --- CosmWasm Keeper ---
	homePath := cast.ToString(appOpts["home"])
	wasmDir := filepath.Join(homePath, "wasm")
	nodeConfig, err := wasm.ReadNodeConfig(MapAppOptions(appOpts))
	if err != nil {
		panic(fmt.Sprintf("error while reading wasm config: %s", err))
	}
	app.WasmKeeper = wasmkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[wasm.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeperV2,
		app.TransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		nodeConfig,
		wasmtypes.VMConfig{},
		wasmkeeper.BuiltInCapabilities(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		GetWasmOpts(nil)...,
	)


	// --- GovKeeper (after wasm so proposals can target x/governance-ext) ---
	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.DistrKeeper,
		app.MsgServiceRouter(),
		govConfig,
		authAddr,
		nil,
	)
	app.GovKeeper = govKeeper

	// --- Module Manager ---
	transferModule := transfer.NewAppModule(app.TransferKeeper)
	app.mm = module.NewManager(
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, app.txConfig),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, nil),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, nil),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, nil),
		distribution.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, nil),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, nil, interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper, nil),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, interfaceRegistry),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, interfaceRegistry),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		// IBC
		ibc.NewAppModule(app.IBCKeeper),
		ibctm.NewAppModule(tmLightClientModule),
		transferModule,
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		// Cosmos EVM (order: feemarket → vm → erc20)
		feemarket.NewAppModule(app.FeeMarketKeeper),
		vm.NewAppModule(app.EVMKeeper, app.AccountKeeper, app.BankKeeper, app.AccountKeeper.AddressCodec()),
		erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper),
		// CosmWasm
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), nil),
	)

	// --- Module Ordering ---
	// PreBlockers: upgrade first, then EVM pre-block, then rest
	app.mm.SetOrderPreBlockers(
		upgradetypes.ModuleName,
		evmtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		feegrant.ModuleName,
		"authz",
		consensusparamtypes.ModuleName,
		ibcexported.ModuleName,
		ibctm.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		feemarkettypes.ModuleName,
		erc20types.ModuleName,
	)

	// BeginBlockers: EIP-1559 feemarket first, then EVM, then ERC-20, then rest
	app.mm.SetOrderBeginBlockers(
		feemarkettypes.ModuleName,
		evmtypes.ModuleName,
		erc20types.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		stakingtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		govtypes.ModuleName,
		upgradetypes.ModuleName,
		wasm.ModuleName,
		feegrant.ModuleName,
		"authz",
		consensusparamtypes.ModuleName,
		ibctm.ModuleName,
	)

	// EndBlockers: EVM → ERC-20 → feemarket (to get full block gas used), then rest
	app.mm.SetOrderEndBlockers(
		evmtypes.ModuleName,
		erc20types.ModuleName,
		feemarkettypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		upgradetypes.ModuleName,
		wasm.ModuleName,
		feegrant.ModuleName,
		"authz",
		consensusparamtypes.ModuleName,
		ibctm.ModuleName,
	)

	// InitGenesis: feemarket before vm, vm before erc20
	genesisOrder := []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		consensusparamtypes.ModuleName,
		ibcexported.ModuleName,
		// EVM genesis order: vm → feemarket → erc20
		evmtypes.ModuleName,
		feemarkettypes.ModuleName,
		erc20types.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		genutiltypes.ModuleName,
		upgradetypes.ModuleName,
		wasm.ModuleName,
		feegrant.ModuleName,
		"authz",
		ibctm.ModuleName,
	}
	app.mm.SetOrderInitGenesis(genesisOrder...)
	app.mm.SetOrderExportGenesis(genesisOrder...)

	// Register services with configurator
	app.Configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	if err := app.mm.RegisterServices(app.Configurator); err != nil {
		panic(err)
	}

	// Register Upgrade handlers (now that mm and Configurator are ready)
	app.RegisterUpgradeHandlers()


	// Set ABCI hooks
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.SetPreBlocker(app.PreBlocker)

	// Set the EVM-aware ante handler
	app.setAnteHandler()

	// Mount KV stores
	app.MountKVStores(keys)
	// Mount object stores (transient per-block data for EVM bloom, gas tracking)
	app.MountObjectStores(oKeys)

	// Configure the EVM mempool (ReapTxs, InsertTx, CheckTx handlers)
	// MUST happen before LoadLatestVersion() which seals the BaseApp.
	app.configureEVMMempool(MapAppOptions(appOpts), sdkLogger)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			panic(err)
		}
	}

	return app
}

// setAnteHandler wires the unified cosmos/evm ante handler that routes EVM txs
// through EVM decorators and standard Cosmos txs through SDK decorators.
func (app *App) setAnteHandler() {
	authante.FeeRecipientModule = authtypes.FeeCollectorName

	options := evmante.HandlerOptions{
		Cdc:                    app.appCodec,
		AccountKeeper:          app.AccountKeeper,
		BankKeeper:             app.BankKeeper,
		ExtensionOptionChecker: antetypes.HasDynamicFeeExtensionOption,
		EvmKeeper:              app.EVMKeeper,
		FeegrantKeeper:         app.FeeGrantKeeper,
		IBCKeeper:              app.IBCKeeper,
		FeeMarketKeeper:        app.FeeMarketKeeper,
		MaxTxGasWanted:         0, // 0 means no limit; set via app.toml EVMMaxTxGasWanted
		DynamicFeeChecker:      true,
		SigGasConsumer:         evmante.SigVerificationGasConsumer,
		SignModeHandler:        app.txConfig.SignModeHandler(),
		PendingTxListener:      func(ethcommon.Hash) {},
	}
	if err := options.Validate(); err != nil {
		panic(err)
	}

	baseHandler := evmante.NewAnteHandler(options)

	// blocked x/authz message types
	blockedMsgs := map[string]bool{
		"/sovereign.bridge.v1.MsgBridgeIn":             true,
		"/sovereign.bridge.v1.MsgBridgeOut":            true,
		"/sovereign.oracle.v1.MsgSubmitOracleCommit":    true,
		"/sovereign.oracle.v1.MsgRevealOracleReport":    true,
		"/sovereign.settlement.v1.MsgSettlement":        true,
		"/cosmos.evm.vm.v1.MsgEthereumTx":              true,
	}

	wrappedAnteHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		for _, msg := range tx.GetMsgs() {
			if msgGrant, ok := msg.(*authz.MsgGrant); ok {
				auth, err := msgGrant.GetAuthorization()
				if err != nil {
					return ctx, fmt.Errorf("failed to get authorization: %w", err)
				}
				msgType := auth.MsgTypeURL()
				if blockedMsgs[msgType] {
					return ctx, fmt.Errorf("authorization grant for message type %s is blocked", msgType)
				}
			}
		}
		return baseHandler(ctx, tx, simulate)
	}

	app.SetAnteHandler(wrappedAnteHandler)
}

// configureEVMMempool sets up the EVM priority nonce mempool and wires
// CometBFT ABCI handlers (ReapTxs, InsertTx, CheckTx, PrepareProposal).
// Without this, CometBFT cannot produce blocks because ReapTxs has no handler.
func (app *App) configureEVMMempool(appOpts servertypes.AppOptions, logger log.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Info("failed to configure EVM mempool (likely stores not loaded/mounted in test), skipping", "err", r)
		}
	}()

	if evmtypes.GetChainConfig() == nil {
		logger.Debug("evm chain config is not set, skipping mempool configuration")
		return
	}

	var (
		mpConfig = evmserver.ResolveMempoolConfig(app.AnteHandler(), appOpts, logger)

		txEncoder       = evmmempool.NewTxEncoder(app.txConfig)
		evmRechecker    = evmmempool.NewTxRechecker(mpConfig.AnteHandler, txEncoder)
		cosmosRechecker = evmmempool.NewTxRechecker(mpConfig.AnteHandler, txEncoder)
		cosmosPoolMaxTx = evmserver.GetCosmosPoolMaxTx(appOpts, logger)
		checkTxTimeout  = evmserver.GetMempoolCheckTxTimeout(appOpts, logger)
	)

	if cosmosPoolMaxTx < 0 {
		logger.Debug("evm mempool is disabled, skipping configuration")
		return
	}

	if err := evmserver.ValidateReapBounds(appOpts, mpConfig.BlockGasLimit); err != nil {
		panic(fmt.Sprintf("failed to validate reap bounds: %s", err.Error()))
	}

	// create mempool
	mempool := evmmempool.NewMempool(
		app.CreateQueryContext,
		logger,
		app.EVMKeeper,
		app.FeeMarketKeeper,
		app.txConfig,
		evmRechecker,
		cosmosRechecker,
		mpConfig,
		cosmosPoolMaxTx,
	)

	// create ABCI handlers
	prepareProposalHandler := baseapp.
		NewDefaultProposalHandler(mempool, &NoCheckProposalTxVerifier{BaseApp: app.BaseApp, txEncoder: app.txConfig.TxEncoder()}).
		PrepareProposalHandler()

	insertTxHandler := mempool.NewInsertTxHandler(app.TxDecode)
	reapTxsHandler := mempool.NewReapTxsHandler()
	checkTxHandler := mempool.NewCheckTxHandler(app.TxDecode, checkTxTimeout)

	// set handlers and the mempool
	app.SetPrepareProposal(prepareProposalHandler)
	app.SetInsertTxHandler(insertTxHandler)
	app.SetReapTxsHandler(reapTxsHandler)
	app.SetCheckTxHandler(checkTxHandler)

	app.SetMempool(mempool)

	app.SetPrepareCheckStater(func(_ sdk.Context) {
		mempool.NotifyNewBlock()
	})
}

// NoCheckProposalTxVerifier skips full ante-handler re-verification during
// PrepareProposal because the EVM mempool already guarantees tx validity.
type NoCheckProposalTxVerifier struct {
	*baseapp.BaseApp
	txEncoder sdk.TxEncoder
}

func (txv *NoCheckProposalTxVerifier) PrepareProposalVerifyTx(tx sdk.Tx) ([]byte, error) {
	return txv.txEncoder(tx)
}

// InitChainer initialises the blockchain from genesis state.
func (app *App) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if len(req.AppStateBytes) > 0 {
		if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
			panic(fmt.Sprintf("failed to unmarshal genesis state: %v", err))
		}
	}
	if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap()); err != nil {
		panic(err)
	}
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// BeginBlocker runs begin-block logic for all modules.
func (app *App) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.mm.BeginBlock(ctx)
}

// EndBlocker runs end-block logic for all modules.
func (app *App) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	res, err := app.mm.EndBlock(ctx)
	if err != nil {
		return res, err
	}
	app.OverrideHistoricalInfo(ctx)
	res.ValidatorUpdates = app.GetEqualizedValidatorUpdates(ctx, res.ValidatorUpdates)
	return res, nil
}

// PreBlocker runs pre-block logic (upgrade module first).
func (app *App) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.mm.PreBlock(ctx)
}



// BlockedAddresses returns the set of module account addresses blocked from receiving tokens.
func BlockedAddresses() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}
	return modAccAddrs
}

func getGovProposalHandlers() []govclient.ProposalHandler {
	return []govclient.ProposalHandler{}
}

// LegacyAmino returns App's amino codec.
func (app *App) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns App's app codec.
func (app *App) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns App's InterfaceRegistry.
func (app *App) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns App's TxConfig.
func (app *App) TxConfig() client.TxConfig {
	return app.txConfig
}

// LoadHeight loads the state at the given block height.
func (app *App) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// GetKey returns the KVStoreKey for the provided store key name.
func (app *App) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// RegisterAPIRoutes registers all application gRPC-Gateway routes.
func (app *App) RegisterAPIRoutes(apiSrv *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSrv.ClientCtx
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSrv.GRPCGatewayRouter)
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSrv.GRPCGatewayRouter)
	node.RegisterGRPCGatewayRoutes(clientCtx, apiSrv.GRPCGatewayRouter)
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSrv.GRPCGatewayRouter)
}

// RegisterTxService registers all Tx-related queries on the gRPC query router.
func (app *App) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.GRPCQueryRouter(), clientCtx, app.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService registers all Tendermint-related queries on the gRPC query router.
func (app *App) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(clientCtx, app.GRPCQueryRouter(), app.interfaceRegistry, app.Query)
}

// RegisterNodeService registers all Node-related queries on the gRPC query router.
func (app *App) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg, func() int64 { return 0 })
}

// GenesisState represents the initial state of each module.
type GenesisState map[string]json.RawMessage
