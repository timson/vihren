export type ServiceResponse = string;

export type FilterValue = {
  name: string;
  samples?: number;
};

export type QueryMetaResponse = {
  result?: FilterValue[];
};

export type SamplePoint = {
  time: string;
  samples: number;
};

export type SampleCountResponse = {
  result?: SamplePoint[];
};

export type MetricsSummary = {
  time?: string;
  avg_cpu: number;
  max_cpu: number;
  avg_memory: number;
  max_memory: number;
  percentile_memory: number;
};

export type MetricsGraphResponse = {
  result?: MetricsSummary[];
};

export type SummaryStats = {
  samples: number;
  avg_cpu: number;
  max_cpu: number;
  current_cpu: number;
  avg_memory: number;
  max_memory: number;
  current_memory: number;
  nodes: number;
};

export type SummaryResponse = {
  result?: SummaryStats;
};

export type FlamegraphNode = {
  name: string;
  value?: number;
  language?: string;
  children?: FlamegraphNode[];
};

export type FlamegraphResponse = FlamegraphNode;

export type RuntimeKey =
  | "appid"
  | "java"
  | "python"
  | "php"
  | "ruby"
  | "node"
  | "dotnet"
  | "cpp"
  | "kernel"
  | "go"
  | "rust"
  | "other";

export type LegendItem = {
  key: RuntimeKey;
  label: string;
  color: string;
};

export type TimeRangeOption = {
  label: string;
  value: string;
};
