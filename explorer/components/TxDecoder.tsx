"use client";

interface TxDecoderProps {
  decoded: Record<string, unknown> | null | undefined;
  messageSchema?: string[];
  title?: string;
}

export default function TxDecoder({
  decoded,
  messageSchema,
  title = "Decoded Payload",
}: TxDecoderProps) {
  const hasData = decoded && Object.keys(decoded).length > 0;

  return (
    <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-4 shadow-xl">
      <div className="flex items-center justify-between border-b border-gray-900 pb-2">
        <h3 className="text-lg font-bold text-white">{title}</h3>
        {messageSchema && messageSchema.length > 0 && (
          <span className="text-xs font-mono text-gray-500 bg-gray-900 border border-gray-800 rounded px-2 py-1">
            {messageSchema.join(", ")}
          </span>
        )}
      </div>

      {hasData ? (
        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 bg-black/30 border border-gray-900 rounded-xl p-4">
            {Object.entries(decoded).map(([key, val]) => (
              <div key={key} className="space-y-1">
                <span className="text-xs font-bold text-gray-500 uppercase tracking-wider">
                  {key}
                </span>
                <div className="text-sm font-mono text-gray-300 break-all">
                  {typeof val === "object"
                    ? JSON.stringify(val)
                    : String(val ?? "null")}
                </div>
              </div>
            ))}
          </div>

          <details className="group">
            <summary className="text-xs text-gray-500 hover:text-white cursor-pointer transition select-none">
              View Raw JSON
            </summary>
            <pre className="mt-2 bg-black/50 border border-gray-900 rounded-xl p-4 overflow-x-auto text-xs font-mono text-green-400 leading-relaxed max-h-[300px]">
              {JSON.stringify(decoded, null, 2)}
            </pre>
          </details>
        </div>
      ) : (
        <p className="py-6 text-center text-gray-500 text-sm">
          No message payload data parsed for this transaction.
        </p>
      )}
    </div>
  );
}
