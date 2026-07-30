import { z } from "zod";

export const PaginationSchema = z.object({
  hasMore: z.boolean().optional(),
  nextCursor: z.string().optional(),
});

export const BlockSchema = z.object({
  height: z.union([z.number(), z.string()]).transform(val => Number(val)),
  time: z.string(),
  proposer: z.string(),
  txCount: z.union([z.number(), z.string()]).transform(val => Number(val)),
  gasUsed: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  gasLimit: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  appHash: z.string().optional(),
});

export const BlockListSchema = z.object({
  blocks: z.array(BlockSchema),
  pagination: PaginationSchema.optional(),
});

export const TxSchema = z.object({
  hash: z.string(),
  height: z.union([z.number(), z.string()]).transform(val => Number(val)),
  time: z.string(),
  type: z.string(),
  msgTypes: z.array(z.string()),
  status: z.union([z.number(), z.string()]).transform(val => Number(val)),
  fee: z.union([z.number(), z.string()]).transform(val => Number(val)),
  gasUsed: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
});

export const TxListSchema = z.object({
  txs: z.array(TxSchema),
  pagination: PaginationSchema.optional(),
});

export const AccountSchema = z.object({
  address_bech32: z.string(),
  address_hex: z.string().nullable().optional(),
  first_seen: z.union([z.number(), z.string()]).transform(val => Number(val)).nullable().optional(),
  last_active: z.union([z.number(), z.string()]).transform(val => Number(val)).nullable().optional(),
  balance_usd: z.number().optional(),
  balances: z.array(z.object({ denom: z.string(), amount: z.string() })).optional(),
});

export const ValidatorSchema = z.object({
  slot_index: z.number(),
  validator_address: z.string(),
  power: z.union([z.number(), z.string()]).transform(val => Number(val)),
  status: z.string(),
  missed_blocks: z.union([z.number(), z.string()]).transform(val => Number(val)),
  certification_score: z.number(),
});

export const ConsensusRoundSchema = z.object({
  height: z.union([z.number(), z.string()]).transform(val => Number(val)),
  round: z.number(),
  step: z.string(),
  validators: z.array(z.object({
    address: z.string(),
    moniker: z.string().optional(),
    voted: z.boolean(),
    power: z.number(),
  })).optional(),
  time_in_step: z.number().optional(),
});

export const SearchResultItemSchema = z.object({
  type: z.string(),
  id: z.string(),
  label: z.string(),
});

export const SearchResponseSchema = z.object({
  results: z.array(SearchResultItemSchema),
});

export const ParamGroupSchema = z.object({
  groupName: z.string(),
  params: z.record(z.string(), z.string()),
});

export const ParamsResponseSchema = z.object({
  groups: z.array(ParamGroupSchema),
});

export const StatusResponseSchema = z.object({
  indexerLagSeconds: z.number(),
  blockscoutLagBlocks: z.number(),
  natsHealth: z.string(),
  apiLatencyP95Ms: z.number(),
  redisHitRatio: z.number(),
});

export const StatsSummarySchema = z.object({
  latestHeight: z.union([z.number(), z.string()]).transform(val => Number(val)),
  totalTxCount: z.union([z.number(), z.string()]).transform(val => Number(val)),
  avgBlockTimeSec: z.number(),
  liveTps: z.number(),
  activeValidators: z.number(),
  totalValidators: z.number(),
  medianGasPrice: z.string().optional(),
  totalSupply: z.string().optional(),
  bondedRatio: z.number().optional(),
});

export const GasPriceSchema = z.object({
  baseFee: z.string(),
  slow: z.string(),
  standard: z.string(),
  fast: z.string(),
  rapid: z.string(),
  unit: z.string(),
});

export const SlotEventSchema = z.object({
  slot: z.number(),
  eventType: z.string(),
  blockHeight: z.union([z.number(), z.string()]).transform(val => Number(val)),
  time: z.string(),
  validatorAddress: z.string(),
});

export const SlotEventsSchema = z.object({
  events: z.array(SlotEventSchema),
});

export const BlockSignSchema = z.object({
  height: z.union([z.number(), z.string()]).transform(val => Number(val)),
  signed: z.boolean(),
});

export const SigningHistorySchema = z.object({
  validatorAddress: z.string(),
  blocks: z.array(BlockSignSchema),
  latestHeight: z.union([z.number(), z.string()]).transform(val => Number(val)),
});

export const Cw20HolderSchema = z.object({
  address: z.string(),
  balance: z.string(),
  share: z.number(),
});

export const Cw20HoldersSchema = z.object({
  contractAddress: z.string(),
  holders: z.array(Cw20HolderSchema),
  totalHolders: z.number(),
});

export const ConstitutionCheckItemSchema = z.object({
  name: z.string(),
  passed: z.boolean(),
  detail: z.string(),
});

export const ConstitutionCheckSchema = z.object({
  passed: z.boolean().nullable(),
  reason: z.string().optional(),
  checks: z.array(ConstitutionCheckItemSchema).optional(),
});

export const BridgeTxSchema = z.object({
  nonce: z.union([z.number(), z.string()]).transform(val => Number(val)),
  direction: z.string(),
  status: z.string(),
  bscLockHash: z.string().optional(),
  bscBlock: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  cosmosMintHash: z.string().optional(),
  cosmosBlock: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  amount: z.string(),
  sender: z.string(),
  receiver: z.string(),
  height: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  time: z.string().optional(),
});

export const BridgeDepositsSchema = z.object({
  deposits: z.array(BridgeTxSchema),
});

export const BridgeWithdrawsSchema = z.object({
  withdrawals: z.array(BridgeTxSchema),
});

export const TpsPointSchema = z.object({
  time: z.string(),
  tps: z.number(),
});

export const TpsHistorySchema = z.object({
  points: z.array(TpsPointSchema),
});

// ─── Phase 2+ Schemas ───

export const ValidatorDetailSchema = z.object({
  address: z.string(),
  slotIndex: z.number(),
  power: z.union([z.number(), z.string()]).transform(val => Number(val)),
  status: z.string(),
  missedBlocks: z.union([z.number(), z.string()]).transform(val => Number(val)),
  certificationScore: z.number(),
  moniker: z.string().optional(),
  uptime: z.number().optional(),
  commission: z.number().optional(),
  delegatedTokens: z.string().optional(),
  delegatorCount: z.number().optional(),
});

export const ValidatorListSchema = z.object({
  validators: z.array(ValidatorDetailSchema),
});

export const OracleFeedSchema = z.object({
  feedId: z.string(),
  description: z.string().optional(),
  latestPrice: z.string().optional(),
  latestRound: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  staleness: z.string().optional(),
  participationRate: z.number().optional(),
  updatedAt: z.string().optional(),
});

export const OracleFeedListSchema = z.object({
  feeds: z.array(OracleFeedSchema),
});

export const OracleRoundSchema = z.object({
  roundId: z.union([z.number(), z.string()]).transform(val => Number(val)),
  feedId: z.string(),
  status: z.string(),
  commits: z.array(z.object({
    operator: z.string(),
    commitHash: z.string().optional(),
  })).optional(),
  reveals: z.array(z.object({
    operator: z.string(),
    value: z.string().optional(),
  })).optional(),
  aggregatedValue: z.string().optional(),
});

export const MilestoneSchema = z.object({
  id: z.string(),
  state: z.string(),
  oracleFeed: z.string().optional(),
  targetValue: z.string().optional(),
  currentValue: z.string().optional(),
  deadline: z.string().optional(),
  createdAt: z.string().optional(),
  achievedAt: z.string().optional(),
});

export const MilestoneListSchema = z.object({
  milestones: z.array(MilestoneSchema),
});

export const SettlementSchema = z.object({
  id: z.string(),
  witnessId: z.string().optional(),
  timestamp: z.string().optional(),
  payloadHash: z.string().optional(),
  status: z.string().optional(),
  amount: z.string().optional(),
});

export const SettlementListSchema = z.object({
  settlements: z.array(SettlementSchema),
});

export const GovernanceProposalSchema = z.object({
  id: z.union([z.number(), z.string()]).transform(val => String(val)),
  title: z.string().optional(),
  description: z.string().optional(),
  type: z.string().optional(),
  status: z.string().optional(),
  votingStartTime: z.string().optional(),
  votingEndTime: z.string().optional(),
  yesVotes: z.string().optional(),
  noVotes: z.string().optional(),
  abstainVotes: z.string().optional(),
  vetoVotes: z.string().optional(),
});

export const GovernanceProposalListSchema = z.object({
  proposals: z.array(GovernanceProposalSchema),
});

export const IbcChannelSchema = z.object({
  channelId: z.string(),
  counterparty: z.string().optional(),
  state: z.string().optional(),
  port: z.string().optional(),
  ordering: z.string().optional(),
  packetCount: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
});

export const IbcChannelListSchema = z.object({
  channels: z.array(IbcChannelSchema),
});

export const IbcAssetSchema = z.object({
  denom: z.string(),
  originChain: z.string().optional(),
  ibcPath: z.string().optional(),
  denomHash: z.string().optional(),
  amount: z.string().optional(),
});

export const IbcAssetListSchema = z.object({
  assets: z.array(IbcAssetSchema),
});

export const ContractSchema = z.object({
  address: z.string(),
  codeId: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  label: z.string().optional(),
  creator: z.string().optional(),
  admin: z.string().optional(),
  typeBadge: z.string().optional(),
  verified: z.boolean().optional(),
});

export const ContractListSchema = z.object({
  contracts: z.array(ContractSchema),
});

export const CodeSchema = z.object({
  codeId: z.union([z.number(), z.string()]).transform(val => Number(val)),
  uploader: z.string().optional(),
  checksum: z.string().optional(),
  instantiationCount: z.number().optional(),
  createdAt: z.string().optional(),
});

export const CodeListSchema = z.object({
  codes: z.array(CodeSchema),
});

// ─── Bridge Schemas ───

export const BridgeSupplyMetricsSchema = z.object({
  cosmosMinted: z.string(),
  bscCirculating: z.string(),
  totalSupply: z.string(),
  bridgeSupplyGauge: z.string(),
});

export const BridgeTxListSchema = z.object({
  txs: z.array(BridgeTxSchema),
});

export const RelayerSchema = z.object({
  address: z.string(),
  status: z.string(),
  lastActive: z.union([z.number(), z.string()]).transform(val => Number(val)),
  missCount: z.union([z.number(), z.string()]).transform(val => Number(val)),
});

export const RelayerListSchema = z.object({
  relayers: z.array(RelayerSchema),
});

export const CircuitBreakerEventSchema = z.object({
  height: z.union([z.number(), z.string()]).transform(val => Number(val)),
  eventType: z.string(),
  triggerAddress: z.string(),
  time: z.string(),
});

export const CircuitBreakerHistorySchema = z.object({
  events: z.array(CircuitBreakerEventSchema),
});

export const NonceRegistrySchema = z.object({
  usedNonces: z.array(z.union([z.number(), z.string()]).transform(val => Number(val))),
  inFlightNonces: z.array(z.union([z.number(), z.string()]).transform(val => Number(val))),
});

// ─── EVM Schemas ───

export const EvmBlockSchema = z.object({
  number: z.union([z.number(), z.string()]).transform(val => Number(val)),
  hash: z.string(),
  timestamp: z.string(),
  miner: z.string().optional(),
  txCount: z.number().optional(),
  gasUsed: z.string().optional(),
  gasLimit: z.string().optional(),
  size: z.string().optional(),
});

export const EvmTxSchema = z.object({
  hash: z.string(),
  blockNumber: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  from: z.string().optional(),
  to: z.string().optional(),
  value: z.string().optional(),
  gas: z.string().optional(),
  gasPrice: z.string().optional(),
  nonce: z.number().optional(),
  status: z.boolean().optional(),
  input: z.string().optional(),
});

export const EvmTokenSchema = z.object({
  address: z.string(),
  name: z.string().optional(),
  symbol: z.string().optional(),
  decimals: z.number().optional(),
  totalSupply: z.string().optional(),
  holders: z.number().optional(),
  transfers: z.number().optional(),
  type: z.string().optional(),
});

export const EvmTokenListSchema = z.object({
  tokens: z.array(EvmTokenSchema),
});

export const TokenTransferSchema = z.object({
  txHash: z.string(),
  blockHeight: z.union([z.number(), z.string()]).transform(val => Number(val)),
  blockTime: z.string(),
  fromAddress: z.string(),
  toAddress: z.string(),
  value: z.string(),
  tokenAddress: z.string().optional(),
  tokenId: z.string().optional(),
  tokenStandard: z.string().optional(),
});

export const TokenTransferListSchema = z.object({
  transfers: z.array(TokenTransferSchema),
});

export const TokenHolderSchema = z.object({
  address: z.string(),
  balance: z.string(),
  share: z.number().optional(),
});

export const TokenHolderListSchema = z.object({
  holders: z.array(TokenHolderSchema),
  totalHolders: z.number().optional(),
});

// ─── Analytics Schemas ───

export const BlockTimePointSchema = z.object({
  time: z.string(),
  blockTime: z.number(),
  p50: z.number().optional(),
  p95: z.number().optional(),
  p99: z.number().optional(),
});

export const BlockTimeHistorySchema = z.object({
  points: z.array(BlockTimePointSchema),
});

export const BridgeVolumePointSchema = z.object({
  time: z.string(),
  volume: z.number(),
});

export const BridgeVolumeHistorySchema = z.object({
  points: z.array(BridgeVolumePointSchema),
});

export const ValidatorUptimeSchema = z.object({
  validatorAddress: z.string(),
  uptime: z.number(),
  signedBlocks: z.number().optional(),
  missedBlocks: z.number().optional(),
});

export const ValidatorUptimeListSchema = z.object({
  validators: z.array(ValidatorUptimeSchema),
});

// ─── Staking Schemas ───

export const StakingStatsSchema = z.object({
  totalBonded: z.string(),
  totalSupply: z.string().optional(),
  bondedRatio: z.number().optional(),
  apr: z.number().optional(),
  inflation: z.number().optional(),
  communityPool: z.string().optional(),
});

// ─── Search Schemas ───

export const SearchResultsSchema = z.object({
  results: z.array(z.object({
    type: z.string(),
    id: z.string(),
    label: z.string(),
    description: z.string().optional(),
  })),
});

// ─── Top Accounts Schema ───

export const TopAccountSchema = z.object({
  address: z.string(),
  balance: z.string(),
  balanceUsd: z.number().optional(),
  txCount: z.number().optional(),
  percentage: z.number().optional(),
  label: z.string().optional(),
});

export const TopAccountListSchema = z.object({
  accounts: z.array(TopAccountSchema),
});

// ─── Supply Distribution Schema ───

export const SupplyDistributionSchema = z.object({
  bonded: z.string().optional(),
  unbonded: z.string().optional(),
  communityPool: z.string().optional(),
  bridgeEscrow: z.string().optional(),
  rewardsBucket: z.string().optional(),
  totalSupply: z.string().optional(),
});

// ─── Chart Data Schema ───

export const ChartDataSchema = z.object({
  chartName: z.string(),
  data: z.array(z.object({
    time: z.string().optional(),
    value: z.union([z.number(), z.string()]).transform(val => Number(val)),
  })),
  description: z.string().optional(),
  source: z.string().optional(),
});

// ─── Address Portfolio Schema ───

export const AddressPortfolioSchema = z.object({
  address: z.string(),
  nativeBalance: z.string().optional(),
  nativeBalanceUsd: z.number().optional(),
  cw20Holdings: z.array(z.object({
    contractAddress: z.string(),
    symbol: z.string().optional(),
    balance: z.string(),
    balanceUsd: z.number().optional(),
  })).optional(),
  erc20Holdings: z.array(z.object({
    tokenAddress: z.string(),
    symbol: z.string().optional(),
    balance: z.string(),
    balanceUsd: z.number().optional(),
  })).optional(),
  nftCount: z.number().optional(),
});

// ─── Mempool Schema ───

export const MempoolTxSchema = z.object({
  hash: z.string(),
  from: z.string().optional(),
  to: z.string().optional(),
  gasPrice: z.string().optional(),
  gasLimit: z.string().optional(),
  nonce: z.number().optional(),
  value: z.string().optional(),
  age: z.string().optional(),
  type: z.string().optional(),
});

export const MempoolListSchema = z.object({
  txs: z.array(MempoolTxSchema),
  total: z.number().optional(),
});

// ─── Contract Deployment Schema ───

export const ContractDeploymentSchema = z.object({
  address: z.string(),
  standard: z.string().optional(),
  deployer: z.string().optional(),
  txHash: z.string().optional(),
  blockHeight: z.union([z.number(), z.string()]).transform(val => Number(val)).optional(),
  blockTime: z.string().optional(),
});

export const ContractDeploymentListSchema = z.object({
  deployments: z.array(ContractDeploymentSchema),
});

// ─── Verify Result Schema ───

export const VerifyResultSchema = z.object({
  success: z.boolean(),
  message: z.string().optional(),
  contractName: z.string().optional(),
  compilerVersion: z.string().optional(),
  verified: z.boolean().optional(),
});
