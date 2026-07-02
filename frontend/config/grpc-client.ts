import { GrpcWebFetchTransport } from "@protobuf-ts/grpcweb-transport";
import { RpcOptions, ServerStreamingCall } from "@protobuf-ts/runtime-rpc";

// Helper to get connected wallet address synchronously
export function getWalletAddressHeader(): string {
  if (typeof window === "undefined") return "";
  const l1 = window.localStorage.getItem("l1_address");
  if (l1) return l1;
  const l2 = window.localStorage.getItem("l2_address");
  if (l2) return l2;
  return "cosmos1rel1x9vy83p40pms777ed1f30e5801046d36";
}

const transport = new GrpcWebFetchTransport({
  baseUrl: process.env.NEXT_PUBLIC_GRPC_WEB_URL || "http://localhost:8080/api/grpcweb",
  interceptors: [
    {
      interceptUnary(next, method, input, options) {
        options.meta = {
          ...options.meta,
          "x-wallet-address": getWalletAddressHeader(),
        };
        return next(method, input, options);
      },
      interceptServerStreaming(next, method, input, options) {
        options.meta = {
          ...options.meta,
          "x-wallet-address": getWalletAddressHeader(),
        };
        return next(method, input, options);
      }
    }
  ]
});

export { transport };

export interface StreamOptions<TRequest extends object, TResponse extends object> {
  request: TRequest;
  onMessage: (message: TResponse) => void;
  onError?: (error: any) => void;
  onStatusChange?: (status: "connected" | "connecting" | "disconnected") => void;
}

export function startStreamWithReconnect<TRequest extends object, TResponse extends object>(
  rpcMethod: (input: TRequest, options?: RpcOptions) => ServerStreamingCall<TRequest, TResponse>,
  options: StreamOptions<TRequest, TResponse>
): { disconnect: () => void } {
  let active = true;
  let attempt = 0;
  let delay = 1000; // start with 1s
  let timeoutId: any = null;
  let abortController: AbortController | null = null;

  const connect = async () => {
    if (!active) return;
    options.onStatusChange?.("connecting");
    abortController = new AbortController();

    try {
      const call = rpcMethod(options.request, { abort: abortController.signal });
      options.onStatusChange?.("connected");
      attempt = 0;
      delay = 1000; // reset delay

      for await (const message of call.responses) {
        if (!active) break;
        options.onMessage(message);
      }

      if (active) {
        throw new Error("Stream closed by server");
      }
    } catch (err: any) {
      if (!active) return;
      options.onError?.(err);
      options.onStatusChange?.("disconnected");

      attempt++;
      // Exponential backoff with a cap of 30 seconds
      const nextDelay = Math.min(delay * Math.pow(2, attempt - 1), 30000);
      console.log(`Stream disconnected. Reconnecting in ${nextDelay}ms (attempt ${attempt})...`);
      timeoutId = setTimeout(connect, nextDelay);
    }
  };

  connect();

  return {
    disconnect: () => {
      active = false;
      if (timeoutId) clearTimeout(timeoutId);
      if (abortController) abortController.abort();
    }
  };
}
