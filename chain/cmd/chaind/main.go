package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"unsafe"

	"cosmossdk.io/core/address"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtcli "github.com/cometbft/cometbft/libs/cli"
	dbm "github.com/cosmos/cosmos-db"
	log "cosmossdk.io/log/v2"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	clientcfg "github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	codec "github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/store/v2"
	snapshottypes "github.com/cosmos/cosmos-sdk/store/v2/snapshots/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/std"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	"github.com/cosmos/cosmos-sdk/x/tx/signing"
	"github.com/cosmos/gogoproto/proto"
	evmaddress "github.com/cosmos/evm/encoding/address"

	cosmosevmcmd "github.com/cosmos/evm/client"
	evmdebug "github.com/cosmos/evm/client/debug"
	"github.com/cosmos/evm/crypto/hd"
	cosmosevmserver "github.com/cosmos/evm/server"
	cosmosevmserverconfig "github.com/cosmos/evm/server/config"
	srvflags "github.com/cosmos/evm/server/flags"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	evmcodec "github.com/cosmos/evm/crypto/codec"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"google.golang.org/protobuf/reflect/protoreflect"
	gproto "google.golang.org/protobuf/proto"

	"github.com/sovereign-l1/chain/app"
)

type EVMAppConfig struct {
	serverconfig.Config

	EVM     cosmosevmserverconfig.EVMConfig
	JSONRPC cosmosevmserverconfig.JSONRPCConfig
	TLS     cosmosevmserverconfig.TLSConfig
}

const EVMAppTemplate = serverconfig.DefaultConfigTemplate + cosmosevmserverconfig.DefaultEVMConfigTemplate

func InitAppConfig(denom string, evmChainID uint64) (string, interface{}) {
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.MinGasPrices = "0" + denom

	evmCfg := cosmosevmserverconfig.DefaultEVMConfig()
	evmCfg.EVMChainID = evmChainID

	customAppConfig := EVMAppConfig{
		Config:  *srvCfg,
		EVM:     *evmCfg,
		JSONRPC: *cosmosevmserverconfig.DefaultJSONRPCConfig(),
		TLS:     *cosmosevmserverconfig.DefaultTLSConfig(),
	}

	return EVMAppTemplate, customAppConfig
}

type logWriter struct {
	logger log.Logger
}

func (w logWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}

type appWrapper struct {
	*app.App
}

func (w appWrapper) GetMempool() sdkmempool.ExtMempool {
	if extMempool, ok := w.App.Mempool().(sdkmempool.ExtMempool); ok {
		return extMempool
	}
	return nil
}

func (w appWrapper) RegisterPendingTxListener(listener func(ethcommon.Hash)) {
	// No-op placeholder
}

func main() {
	setupSDKConfig()

	rootCmd := NewRootCmd()
	if err := svrcmd.Execute(rootCmd, app.Name+"d", app.DefaultNodeHome); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}

func setupSDKConfig() {
	// Enforce default power reduction of 10^6 since staking operates in ucsov (6 decimals)
	sdk.DefaultPowerReduction = math.NewInt(1_000_000)

	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(sdk.Bech32PrefixValAddr, sdk.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(sdk.Bech32PrefixConsAddr, sdk.Bech32PrefixConsPub)
	cfg.Seal()
}

func NewRootCmd() *cobra.Command {
	// Build encoding config directly from ModuleBasics to avoid creating
	// a full App instance (which sets global EVM chainConfig and panics
	// when NewApp is called again during 'chaind start').
	signingOptions := signing.Options{
		AddressCodec:          evmaddress.NewEvmCodec(sdk.Bech32PrefixAccAddr),
		ValidatorAddressCodec: evmaddress.NewEvmCodec(sdk.Bech32PrefixValAddr),
		CustomGetSigners: map[protoreflect.FullName]signing.GetSignersFunc{
			evmtypes.MsgEthereumTxCustomGetSigner.MsgType:     evmtypes.MsgEthereumTxCustomGetSigner.Fn,
			erc20types.MsgConvertERC20CustomGetSigner.MsgType: erc20types.MsgConvertERC20CustomGetSigner.Fn,
			"sovereign.oracle.v1.MsgCommitOracleHash": func(msg gproto.Message) ([][]byte, error) {
				refMsg := msg.ProtoReflect()
				descriptor := refMsg.Descriptor()
				fieldDesc := descriptor.Fields().ByName("operator")
				if fieldDesc == nil {
					return nil, fmt.Errorf("field 'operator' not found in message %s", descriptor.FullName())
				}
				operatorVal := refMsg.Get(fieldDesc).String()
				addr, err := sdk.ValAddressFromBech32(operatorVal)
				if err != nil {
					return nil, err
				}
				return [][]byte{addr.Bytes()}, nil
			},
			"sovereign.oracle.v1.MsgRevealOracleReport": func(msg gproto.Message) ([][]byte, error) {
				refMsg := msg.ProtoReflect()
				descriptor := refMsg.Descriptor()
				fieldDesc := descriptor.Fields().ByName("operator")
				if fieldDesc == nil {
					return nil, fmt.Errorf("field 'operator' not found in message %s", descriptor.FullName())
				}
				operatorVal := refMsg.Get(fieldDesc).String()
				addr, err := sdk.ValAddressFromBech32(operatorVal)
				if err != nil {
					return nil, err
				}
				return [][]byte{addr.Bytes()}, nil
			},
			"sovereign.milestone.v1.MsgCreateMilestone": func(msg gproto.Message) ([][]byte, error) {
				refMsg := msg.ProtoReflect()
				descriptor := refMsg.Descriptor()
				fieldDesc := descriptor.Fields().ByName("creator")
				if fieldDesc == nil {
					return nil, fmt.Errorf("field 'creator' not found in message %s", descriptor.FullName())
				}
				creatorVal := refMsg.Get(fieldDesc).String()
				addr, err := sdk.AccAddressFromBech32(creatorVal)
				if err != nil {
					return nil, err
				}
				return [][]byte{addr.Bytes()}, nil
			},
		},
	}
	interfaceRegistry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles:     proto.HybridResolver,
		SigningOptions: signingOptions,
	})
	if err != nil {
		panic(err)
	}

	std.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	app.ModuleBasics.RegisterInterfaces(interfaceRegistry)
	evmcodec.RegisterInterfaces(interfaceRegistry)
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)
	legacyAmino := codec.NewLegacyAmino()
	evmcodec.RegisterCrypto(legacyAmino)

	populateCodecs(app.ModuleBasics, appCodec, signingOptions.AddressCodec)

	encodingConfig := struct {
		InterfaceRegistry codectypes.InterfaceRegistry
		Codec             codec.Codec
		TxConfig          client.TxConfig
		Amino             *codec.LegacyAmino
	}{
		InterfaceRegistry: interfaceRegistry,
		Codec:             appCodec,
		TxConfig:          txConfig,
		Amino:             legacyAmino,
	}

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithBroadcastMode(flags.FlagBroadcastMode).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("").
		WithKeyringOptions(hd.EthSecp256k1Option()).
		WithLedgerHasProtobuf(true)

	rootCmd := &cobra.Command{
		Use:   app.Name + "d",
		Short: "Sovereign L1 Blockchain Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = clientcfg.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := InitAppConfig(evmtypes.DefaultEVMExtendedDenom, app.EVMChainID)
			customTMConfig := cmtcfg.DefaultConfig()

			return sdkserver.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customTMConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig.TxConfig)

	return rootCmd
}

func initRootCmd(rootCmd *cobra.Command, txConfig client.TxConfig) {
	sdkAppCreator := func(l log.Logger, d dbm.DB, ao servertypes.AppOptions) servertypes.Application {
		var cache storetypes.MultiStorePersistentCache
		if cast.ToBool(ao.Get(sdkserver.FlagInterBlockCache)) {
			cache = store.NewCommitKVStoreCacheManager()
		}

		pruningOpts, err := sdkserver.GetPruningOptionsFromFlags(ao)
		if err != nil {
			panic(err)
		}

		snapshotStore, err := sdkserver.GetSnapshotStore(ao)
		if err != nil {
			panic(err)
		}

		snapshotOptions := snapshottypes.NewSnapshotOptions(
			cast.ToUint64(ao.Get(sdkserver.FlagStateSyncSnapshotInterval)),
			cast.ToUint32(ao.Get(sdkserver.FlagStateSyncSnapshotKeepRecent)),
		)

		baseappOptions := []func(*baseapp.BaseApp){
			baseapp.SetPruning(pruningOpts),
			baseapp.SetMinGasPrices(cast.ToString(ao.Get(sdkserver.FlagMinGasPrices))),
			baseapp.SetQueryGasLimit(cast.ToUint64(ao.Get(sdkserver.FlagQueryGasLimit))),
			baseapp.SetHaltHeight(cast.ToUint64(ao.Get(sdkserver.FlagHaltHeight))),
			baseapp.SetHaltTime(cast.ToUint64(ao.Get(sdkserver.FlagHaltTime))),
			baseapp.SetMinRetainBlocks(cast.ToUint64(ao.Get(sdkserver.FlagMinRetainBlocks))),
			baseapp.SetInterBlockCache(cache),
			baseapp.SetTrace(cast.ToBool(ao.Get(sdkserver.FlagTrace))),
			baseapp.SetIndexEvents(cast.ToStringSlice(ao.Get(sdkserver.FlagIndexEvents))),
			baseapp.SetSnapshot(snapshotStore, snapshotOptions),
			baseapp.SetIAVLCacheSize(cast.ToInt(ao.Get(sdkserver.FlagIAVLCacheSize))),
			baseapp.SetChainID(cast.ToString(ao.Get(flags.FlagChainID))),
		}

		optsMap := populateOptsMap(ao)

		return app.NewApp(
			logWriter{logger: l},
			d,
			nil,
			true,
			optsMap,
			baseappOptions...,
		)
	}

	rootCmd.AddCommand(
		genutilcli.InitCmd(app.ModuleBasics, app.DefaultNodeHome),
		genutilcli.Commands(txConfig, app.ModuleBasics, app.DefaultNodeHome),
		cmtcli.NewCompletionCmd(rootCmd, true),
		evmdebug.Cmd(),
		pruning.Cmd(sdkAppCreator, app.DefaultNodeHome),
		snapshot.Cmd(sdkAppCreator),
	)

	cosmosevmserver.AddCommands(
		rootCmd,
		cosmosevmserver.NewDefaultStartOptions(newApp, app.DefaultNodeHome),
		appExport,
		func(_ *cobra.Command) {},
	)

	rootCmd.AddCommand(
		cosmosevmcmd.KeyCommands(app.DefaultNodeHome, true),
	)

	rootCmd.AddCommand(
		sdkserver.StatusCommand(),
		queryCommand(),
		txCommand(),
	)

	if _, err := srvflags.AddTxFlags(rootCmd); err != nil {
		panic(err)
	}
}

func newApp(
	logger log.Logger,
	db dbm.DB,
	appOpts servertypes.AppOptions,
) cosmosevmserver.Application {
	var cache storetypes.MultiStorePersistentCache
	if cast.ToBool(appOpts.Get(sdkserver.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	pruningOpts, err := sdkserver.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	snapshotStore, err := sdkserver.GetSnapshotStore(appOpts)
	if err != nil {
		panic(err)
	}

	snapshotOptions := snapshottypes.NewSnapshotOptions(
		cast.ToUint64(appOpts.Get(sdkserver.FlagStateSyncSnapshotInterval)),
		cast.ToUint32(appOpts.Get(sdkserver.FlagStateSyncSnapshotKeepRecent)),
	)

	baseappOptions := []func(*baseapp.BaseApp){
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(sdkserver.FlagMinGasPrices))),
		baseapp.SetQueryGasLimit(cast.ToUint64(appOpts.Get(sdkserver.FlagQueryGasLimit))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(sdkserver.FlagHaltHeight))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(sdkserver.FlagHaltTime))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(sdkserver.FlagMinRetainBlocks))),
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get(sdkserver.FlagTrace))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(sdkserver.FlagIndexEvents))),
		baseapp.SetSnapshot(snapshotStore, snapshotOptions),
		baseapp.SetIAVLCacheSize(cast.ToInt(appOpts.Get(sdkserver.FlagIAVLCacheSize))),
		baseapp.SetChainID(cast.ToString(appOpts.Get(flags.FlagChainID))),
	}

	optsMap := populateOptsMap(appOpts)

	return appWrapper{
		App: app.NewApp(
			logWriter{logger: logger},
			db,
			nil,
			true,
			optsMap,
			baseappOptions...,
		),
	}
}

func appExport(
	logger log.Logger,
	db dbm.DB,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}
	viperAppOpts.Set(sdkserver.FlagInvCheckPeriod, 1)

	var loadLatest bool
	if height == -1 {
		loadLatest = true
	}

	optsMap := make(map[string]interface{})
	optsMap["home"] = homePath

	exampleApp := app.NewApp(
		logWriter{logger: logger},
		db,
		nil,
		loadLatest,
		optsMap,
	)

	if height != -1 {
		if err := exampleApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return exampleApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		rpc.ValidatorCommand(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
		sdkserver.QueryBlockCmd(),
		sdkserver.QueryBlockResultsCmd(),
	)

	app.ModuleBasics.AddQueryCommands(cmd)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	app.ModuleBasics.AddTxCommands(cmd)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func populateCodecs(mb module.BasicManager, cdc codec.Codec, ac address.Codec) {
	for name, m := range mb {
		val := reflect.ValueOf(m)
		if val.Kind() == reflect.Struct {
			ptr := reflect.New(val.Type())
			ptr.Elem().Set(val)

			for i := 0; i < ptr.Elem().NumField(); i++ {
				fieldVal := ptr.Elem().Field(i)
				fieldType := fieldVal.Type()

				if fieldType.Implements(reflect.TypeOf((*codec.Codec)(nil)).Elem()) {
					if fieldVal.IsNil() {
						setUnexportedField(fieldVal, reflect.ValueOf(cdc))
					}
				}
				if fieldType.Implements(reflect.TypeOf((*address.Codec)(nil)).Elem()) {
					if fieldVal.IsNil() {
						setUnexportedField(fieldVal, reflect.ValueOf(ac))
					}
				}
			}
			mb[name] = ptr.Elem().Interface().(module.AppModuleBasic)
		}
	}
}

func setUnexportedField(field reflect.Value, value reflect.Value) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(value)
}

func populateOptsMap(ao servertypes.AppOptions) map[string]interface{} {
	optsMap := make(map[string]interface{})
	optsMap["home"] = cast.ToString(ao.Get(flags.FlagHome))

	keys := []string{
		"mempool.max-txs",
		"evm.mempool.price-limit",
		"evm.mempool.price-bump",
		"evm.mempool.account-slots",
		"evm.mempool.global-slots",
		"evm.mempool.account-queue",
		"evm.mempool.global-queue",
		"evm.mempool.insert-queue-size",
		"evm.mempool.lifetime",
		"evm.mempool.pending-tx-proposal-timeout",
		"evm.mempool.check-tx-timeout",
		"evm.mempool.enable-tx-tracker",
		"evm.mempool.cosmos-pool-max-tx",
		"evm.mempool.cosmos-max-txs",
		"evm.mempool.max-txs",
	}

	for _, key := range keys {
		val := ao.Get(key)
		if val != nil {
			optsMap[key] = val
		}
	}

	return optsMap
}
