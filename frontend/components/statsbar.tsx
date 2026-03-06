import type { SummaryStats } from "../types";

type StatItem = {
  label: string;
  value: string;
};

type StatsBarProps = {
  summary?: SummaryStats | null;
  loading?: boolean;
};

const compactNumber = new Intl.NumberFormat("en", {
  notation: "compact",
  maximumFractionDigits: 1
});

const formatPercent = (value: number) => {
  if (!Number.isFinite(value)) {
    return "0.0%";
  }
  return `${value.toFixed(1)}%`;
};

const formatSamples = (value: number) => {
  if (!Number.isFinite(value)) {
    return "0";
  }
  return compactNumber.format(value);
};

function StatsBar({ summary, loading }: StatsBarProps) {
  const isReady = Boolean(summary) && !loading;
  const safeSummary = summary ?? {
    samples: 0,
    avg_cpu: 0,
    max_cpu: 0,
    current_cpu: 0,
    avg_memory: 0,
    max_memory: 0,
    current_memory: 0,
    nodes: 0
  };
  const stats: StatItem[] = [
    {
      label: "Samples collected",
      value: isReady ? formatSamples(safeSummary.samples) : "—"
    },
    {
      label: "CPU Utilization",
      value: isReady
        ? `${formatPercent(safeSummary.avg_cpu)} avg – ${formatPercent(
            safeSummary.max_cpu
          )} max · ${formatPercent(safeSummary.current_cpu)}`
        : "—"
    },
    {
      label: "Memory utilization",
      value: isReady
        ? `${formatPercent(safeSummary.avg_memory)} avg – ${formatPercent(
            safeSummary.max_memory
          )} max · ${formatPercent(safeSummary.current_memory)}`
        : "—"
    },
    {
      label: "Nodes",
      value: isReady ? String(safeSummary.nodes) : "—"
    }
  ];

  return (
    <div className="metrics-bar">
      {stats.map((item) => (
        <div className="metric-item" key={item.label}>
          <span className="metric-label">{item.label}</span>
          <span className="metric-value">{item.value}</span>
        </div>
      ))}
    </div>
  );
}

export default StatsBar;
