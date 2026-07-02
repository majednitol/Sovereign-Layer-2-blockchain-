# ADR 012: Cursor-Based Pagination for gRPC and REST APIs

## Context & Problem Statement
Sovereign L1 serves query requests for event-stream data (e.g. `QueryBridgeActivity`, `QuerySettlements`). Standard offset-based pagination (`OFFSET N LIMIT M`) is inefficient for large datasets ($O(N)$ lookup time) and prone to duplicate or missing elements when items are inserted between pages. We require a consistent, performant paging mechanism.

## Decision & Design

1. **Mandatory Cursor-Based Pagination**: All list-oriented gRPC and REST RPC endpoints in the off-chain backend (`module/api`) and custom Cosmos SDK modules must use cursor-based (keyset) pagination.
2. **Cursor Structure**:
   - The cursor is an opaque base64-encoded string representing a unique tuple: `(block_height, event_index)`.
   - In the database, queries filter by coordinates: `WHERE (block_height, event_index) < (cursor_height, cursor_index) ORDER BY block_height DESC, event_index DESC LIMIT L`.
3. **Protobuf Contract Definitions**:
   - Request messages must include a `PageRequest` object:
     ```protobuf
     message PageRequest {
       bytes cursor = 1;
       uint32 limit = 2;
     }
     ```
   - Response messages must return a `PageResponse` object:
     ```protobuf
     message PageResponse {
       bytes next_cursor = 1;
       bool has_more = 2;
     }
     ```
4. **Client Usage**: Clients receive `next_cursor` and pass it unchanged to the subsequent query request until `has_more` is false.
