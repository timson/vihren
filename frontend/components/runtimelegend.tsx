import type { LegendItem, RuntimeKey } from "../types";

export type RuntimeLegendProps = {
  items: LegendItem[];
  enabled: RuntimeKey[];
  onToggle: (key: RuntimeKey) => void;
};

function RuntimeLegend({ items, enabled, onToggle }: RuntimeLegendProps) {
  return (
    <>
      {items.map((item) => {
        const isActive = enabled.includes(item.key);
        return (
          <button
            key={item.key}
            type="button"
            className={`legend-item ${isActive ? "active" : "inactive"}`}
            onClick={() => onToggle(item.key)}
            aria-pressed={isActive}
          >
            <span className="legend-dot" style={{ background: item.color }}></span>
            {item.label}
          </button>
        );
      })}
    </>
  );
}

export default RuntimeLegend;
