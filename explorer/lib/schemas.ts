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
