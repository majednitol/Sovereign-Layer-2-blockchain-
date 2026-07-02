"use client";

interface Event {
  type: string;
  attributes: { key: string; value: string }[];
}

interface EventLogProps {
  events: Event[];
  maxHeight?: string;
  title?: string;
  emptyMessage?: string;
}

export default function EventLog({
  events,
  maxHeight = "max-h-[500px]",
  title = "Events",
  emptyMessage = "No events.",
}: EventLogProps) {
  if (!events || events.length === 0) {
    return (
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-6">
        <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-2">
          {title}
        </h3>
        <p className="py-6 text-center text-gray-500 text-sm">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 space-y-3 shadow-xl">
      <h3 className="text-lg font-bold text-white border-b border-gray-900 pb-2">
        {title} ({events.length})
      </h3>
      <div
        className={`space-y-3 overflow-y-auto pr-2 divide-y divide-gray-900 ${maxHeight}`}
      >
        {events.map((ev, idx) => (
          <div key={idx} className="pt-3 first:pt-0 space-y-1">
            <span className="text-xs font-bold text-blue-400 font-mono">{ev.type}</span>
            <div className="grid grid-cols-1 gap-1 pl-3">
              {(ev.attributes || []).map((attr, attrIdx) => (
                <div
                  key={attrIdx}
                  className="text-xs font-mono flex items-start space-x-2"
                >
                  <span className="text-gray-500 font-medium shrink-0">{attr.key}:</span>
                  <span className="text-gray-300 break-all">{attr.value || ""}</span>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
