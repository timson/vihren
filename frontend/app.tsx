import { Loader, Text } from "@mantine/core";
import { useCallback, useEffect, useMemo, useState } from "react";
import Sidebar from "./components/sidebar";
import StatsBar from "./components/statsbar";
import TopBar from "./components/topbar";
import RuntimeLegend from "./components/runtimelegend";
import ExportBar from "./components/exportbar";
import SamplesTimeline, { type SampleBin } from "./components/samplesTimeline";
import MetricsTimeline from "./components/metricsTimeline";
import {
  allRuntimeKeys,
  filterFlamegraphData,
  legendItems,
  applyFlamegraphSearch,
  renderFlamegraph
} from "./lib/flamegraph";
import {
  formatDateTimeLocal,
  formatForApi,
  getInitialTimeState,
  parseRangeToMs,
  syncTimeToUrl,
  timeRanges
} from "./lib/time";
import type {
  FilterValue,
  FlamegraphResponse,
  QueryMetaResponse,
  RuntimeKey,
  SampleCountResponse,
  SamplePoint,
  MetricsGraphResponse,
  MetricsSummary,
  SummaryResponse,
  SummaryStats,
  ServiceResponse
} from "./types";

const API = {
  services: "/api/v1/services",
  flamegraph: "/api/v1/flamegraph",
  queryMeta: "/api/v1/query",
  metricsGraph: "/api/v1/metrics/graph",
  summary: "/api/v1/summary"
};

const metaTargets = [
  { label: "Hostname", lookupFor: "hostname" },
  { label: "Container", lookupFor: "container" },
  { label: "Workload", lookupFor: "pod" }
] as const;

const filterStorageKey = "flamedb:selected_filters";

function App() {
  const [services, setServices] = useState<ServiceResponse[]>([]);
  const [selectedService, setSelectedService] = useState("");
  const [graphData, setGraphData] = useState<FlamegraphResponse | null>(null);
  const [loadingServices, setLoadingServices] = useState(true);
  const [loadingGraph, setLoadingGraph] = useState(false);
  const [error, setError] = useState("");
  const [timePickerOpen, setTimePickerOpen] = useState(false);

  // Compute once from URL on first render; relative ranges recalculate from
  // current time so they're always fresh, absolute ranges restore as-is.
  const [initialTimeState] = useState(() =>
    getInitialTimeState(timeRanges.map((r) => r.value))
  );
  const [timeRange, setTimeRange] = useState(initialTimeState.timeRange);
  const [startTime, setStartTime] = useState(initialTimeState.start);
  const [endTime, setEndTime] = useState(initialTimeState.end);
  const [appliedStartTime, setAppliedStartTime] = useState(initialTimeState.start);
  const [appliedEndTime, setAppliedEndTime] = useState(initialTimeState.end);
  const [filterOptions, setFilterOptions] = useState<
    { group: string; lookupFor: string; items: { value: string; label: string }[] }[]
  >([]);
  const [selectedFilters, setSelectedFilters] = useState<string[]>([]);
  const [loadingFilters, setLoadingFilters] = useState(false);
  const [sampleCounts, setSampleCounts] = useState<SamplePoint[]>([]);
  const [loadingSamples, setLoadingSamples] = useState(false);
  const [samplesResetKey, setSamplesResetKey] = useState(0);
  const [searchValue, setSearchValue] = useState("");
  const [graphView, setGraphView] = useState<"samples" | "cpu" | "memory">(
    "samples"
  );
  const [metricsGraph, setMetricsGraph] = useState<MetricsSummary[]>([]);
  const [loadingMetrics, setLoadingMetrics] = useState(false);
  const [summaryStats, setSummaryStats] = useState<SummaryStats | null>(null);
  const [loadingSummary, setLoadingSummary] = useState(false);
  const [selectedRootFrame, setSelectedRootFrame] = useState("");
  const [enabledRuntimes, setEnabledRuntimes] = useState<RuntimeKey[]>(
    () => allRuntimeKeys
  );

  const serviceOptions = useMemo(
    () =>
      services.map((service) => ({
        value: service,
        label: service
      })),
    [services]
  );

  useEffect(() => {
    try {
      const raw = localStorage.getItem(filterStorageKey);
      if (!raw) {
        return;
      }
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        setSelectedFilters(parsed.filter((value) => typeof value === "string"));
      }
    } catch {
      setSelectedFilters([]);
    }
  }, []);

  useEffect(() => {
    localStorage.setItem(filterStorageKey, JSON.stringify(selectedFilters));
  }, [selectedFilters]);

  const applyFiltersToUrl = useCallback(
    (url: URL) => {
      selectedFilters.forEach((value) => {
        const separator = value.indexOf(":");
        if (separator <= 0) {
          return;
        }
        let key = value.slice(0, separator);
        const item = value.slice(separator + 1);
        if (!item) {
          return;
        }
        url.searchParams.append(key, item);
      });
    },
    [selectedFilters]
  );

  const loadServices = useCallback(async (startValue: string, endValue: string) => {
    setLoadingServices(true);
    setError("");
    try {
      const url = new URL(API.services, window.location.origin);
      url.searchParams.set("start_datetime", formatForApi(startValue));
      url.searchParams.set("end_datetime", formatForApi(endValue));
      const response = await fetch(url.toString());
      if (!response.ok) {
        throw new Error(`Services request failed (${response.status})`);
      }
      const payload = await response.json();
      const result = Array.isArray(payload?.result) ? payload.result : [];
      setServices(result);
      if (result.length === 0) {
        setError("No services returned from the API.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load services.");
      setServices([]);
    } finally {
      setLoadingServices(false);
    }
  }, []);

  const buildMetaUrl = useCallback(
    (serviceId: string, startValue: string, endValue: string, lookupFor: string) => {
      const url = new URL(API.queryMeta, window.location.origin);
      url.searchParams.set("service", serviceId);
      url.searchParams.set("start_datetime", formatForApi(startValue));
      url.searchParams.set("end_datetime", formatForApi(endValue));
      url.searchParams.set("lookup_for", lookupFor);
      applyFiltersToUrl(url);
      return url.toString();
    },
    [applyFiltersToUrl]
  );

  const loadFilterOptions = useCallback(
    async (serviceId: string, startValue: string, endValue: string) => {
      if (!serviceId) {
        setFilterOptions([]);
        setSelectedFilters([]);
        return;
      }
      setLoadingFilters(true);
      try {
        const results = await Promise.all(
          metaTargets.map(async (target) => {
            const response = await fetch(
              buildMetaUrl(serviceId, startValue, endValue, target.lookupFor)
            );
            if (!response.ok) {
              throw new Error(`Filters request failed (${response.status})`);
            }
            const payload = (await response.json()) as QueryMetaResponse;
            const rawValues = Array.isArray(payload?.result)
              ? payload.result
              : [];
            const names = rawValues
              .map((item) =>
                typeof item === "string" ? item : (item as FilterValue)?.name
              )
              .filter((value): value is string => Boolean(value));
            return {
              group: target.label,
              lookupFor: target.lookupFor,
              items: names.map((name) => ({
                value: `${target.lookupFor}:${name}`,
                label: name
              }))
            };
          })
        );
        const groupedOptions = results.filter((group) => group.items.length > 0);
        setFilterOptions(groupedOptions);
        setSelectedFilters((prev) => {
          const allowed = new Set(
            groupedOptions.flatMap((group) => group.items.map((item) => item.value))
          );
          const next = prev.filter((value) => allowed.has(value));
          if (next.length === prev.length && next.every((value, index) => value === prev[index])) {
            return prev;
          }
          return next;
        });
      } catch (err) {
        setFilterOptions([]);
        setSelectedFilters([]);
      } finally {
        setLoadingFilters(false);
      }
    },
    [buildMetaUrl]
  );

  const loadSampleCounts = useCallback(
    async (serviceId: string, startValue: string, endValue: string) => {
      if (!serviceId) {
        setSampleCounts([]);
        return;
      }
      setLoadingSamples(true);
      try {
        const response = await fetch(
          buildMetaUrl(serviceId, startValue, endValue, "samples")
        );
        if (!response.ok) {
          throw new Error(`Samples request failed (${response.status})`);
        }
        const payload = (await response.json()) as SampleCountResponse;
        const result = Array.isArray(payload?.result) ? payload.result : [];
        const sorted = [...result].sort(
          (a, b) => new Date(a.time).getTime() - new Date(b.time).getTime()
        );
        setSampleCounts(sorted);
      } catch (err) {
        setSampleCounts([]);
      } finally {
        setLoadingSamples(false);
      }
    },
    [buildMetaUrl]
  );

  const buildFlamegraphUrl = useCallback(
    (serviceId: string, startValue: string, endValue: string) => {
      const url = new URL(API.flamegraph, window.location.origin);
      url.searchParams.set("service", serviceId);
      url.searchParams.set("start_datetime", formatForApi(startValue));
      url.searchParams.set("end_datetime", formatForApi(endValue));
      applyFiltersToUrl(url);
      return url.toString();
    },
    [applyFiltersToUrl]
  );

  const buildMetricsUrl = useCallback(
    (serviceId: string, startValue: string, endValue: string) => {
      const url = new URL(API.metricsGraph, window.location.origin);
      url.searchParams.set("service", serviceId);
      url.searchParams.set("start_datetime", formatForApi(startValue));
      url.searchParams.set("end_datetime", formatForApi(endValue));
      applyFiltersToUrl(url);
      return url.toString();
    },
    [applyFiltersToUrl]
  );

  const buildSummaryUrl = useCallback(
    (serviceId: string, startValue: string, endValue: string) => {
      const url = new URL(API.summary, window.location.origin);
      url.searchParams.set("service", serviceId);
      url.searchParams.set("start_datetime", formatForApi(startValue));
      url.searchParams.set("end_datetime", formatForApi(endValue));
      applyFiltersToUrl(url);
      return url.toString();
    },
    [applyFiltersToUrl]
  );

  const loadMetricsGraph = useCallback(
    async (serviceId: string, startValue: string, endValue: string) => {
      if (!serviceId) {
        setMetricsGraph([]);
        return;
      }
      setLoadingMetrics(true);
      try {
        const response = await fetch(
          buildMetricsUrl(serviceId, startValue, endValue)
        );
        if (!response.ok) {
          throw new Error(`Metrics request failed (${response.status})`);
        }
        const payload = (await response.json()) as MetricsGraphResponse;
        const result = Array.isArray(payload?.result) ? payload.result : [];
        const sorted = [...result].sort(
          (a, b) =>
            new Date(a.time ?? 0).getTime() - new Date(b.time ?? 0).getTime()
        );
        setMetricsGraph(sorted);
      } catch (err) {
        setMetricsGraph([]);
      } finally {
        setLoadingMetrics(false);
      }
    },
    [buildMetricsUrl]
  );

  const loadSummaryStats = useCallback(
    async (serviceId: string, startValue: string, endValue: string) => {
      if (!serviceId) {
        setSummaryStats(null);
        return;
      }
      setLoadingSummary(true);
      try {
        const response = await fetch(
          buildSummaryUrl(serviceId, startValue, endValue)
        );
        if (!response.ok) {
          throw new Error(`Summary request failed (${response.status})`);
        }
        const payload = (await response.json()) as SummaryResponse;
        setSummaryStats(payload?.result ?? null);
      } catch {
        setSummaryStats(null);
      } finally {
        setLoadingSummary(false);
      }
    },
    [buildSummaryUrl]
  );

  const loadFlamegraph = useCallback(
    async (serviceId: string, startValue?: string, endValue?: string) => {
      if (!serviceId) {
        return;
      }
      const resolvedStart = startValue ?? startTime;
      const resolvedEnd = endValue ?? endTime;
      setLoadingGraph(true);
      setError("");
      try {
        const response = await fetch(
          buildFlamegraphUrl(serviceId, resolvedStart, resolvedEnd)
        );
        if (!response.ok) {
          throw new Error(`Flamegraph request failed (${response.status})`);
        }
        const payload = (await response.json()) as FlamegraphResponse;
        setGraphData(payload);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load flamegraph.");
        setGraphData(null);
      } finally {
        setLoadingGraph(false);
      }
    },
    [buildFlamegraphUrl, startTime, endTime]
  );

  const filteredGraphData = useMemo(() => {
    if (!graphData) {
      return null;
    }
    return filterFlamegraphData(graphData, enabledRuntimes);
  }, [graphData, enabledRuntimes]);

  const rootFrameOptions = useMemo(() => {
    const options = [{ value: "", label: "All apps" }];
    const children = filteredGraphData?.children ?? [];
    if (children.length === 0) {
      return options;
    }
    const total =
      filteredGraphData?.value ??
      children.reduce((sum, child) => sum + (child.value ?? 0), 0);
    const entries = children
      .map((child) => {
        const value = child.value ?? 0;
        const percent = total > 0 ? (value / total) * 100 : 0;
        return {
          name: child.name ?? "",
          percent
        };
      })
      .filter((entry) => entry.name);
    entries.sort((a, b) => b.percent - a.percent);
    entries.forEach((entry) => {
      options.push({
        value: entry.name,
        label: `${entry.name} (${entry.percent.toFixed(1)}%)`
      });
    });
    return options;
  }, [filteredGraphData]);

  const displayGraphData = useMemo(() => {
    if (!filteredGraphData) {
      return null;
    }
    if (!selectedRootFrame) {
      return filteredGraphData;
    }
    const selected = filteredGraphData.children?.find(
      (child) => child.name === selectedRootFrame
    );
    return selected ?? filteredGraphData;
  }, [filteredGraphData, selectedRootFrame]);

  useEffect(() => {
    if (!selectedRootFrame) {
      return;
    }
    const available = new Set(
      rootFrameOptions.map((option) => option.value)
    );
    if (!available.has(selectedRootFrame)) {
      setSelectedRootFrame("");
    }
  }, [rootFrameOptions, selectedRootFrame]);

  const sampleBins = useMemo<SampleBin[]>(() => {
    if (sampleCounts.length === 0) {
      return [];
    }
    const centers = sampleCounts.map((point) => new Date(point.time).getTime());
    if (centers.length === 1) {
      const center = centers[0];
      const intervalMs = 60 * 1000;
      return [
        {
          startTs: center - intervalMs / 2,
          endTs: center + intervalMs / 2,
          centerTs: center,
          value: sampleCounts[0].samples
        }
      ];
    }
    const diffs = centers
      .slice(1)
      .map((value, index) => Math.max(1, value - centers[index]))
      .sort((a, b) => a - b);
    const mid = Math.floor(diffs.length / 2);
    const baseIntervalMs =
      diffs.length % 2 === 0
        ? Math.round((diffs[mid - 1] + diffs[mid]) / 2)
        : diffs[mid];
    return centers.map((center, index) => ({
      startTs: center - baseIntervalMs / 2,
      endTs: center + baseIntervalMs / 2,
      centerTs: center,
      value: sampleCounts[index].samples
    }));
  }, [sampleCounts]);

  const toggleRuntime = useCallback(
    (runtime: RuntimeKey) => {
      const isAllEnabled = enabledRuntimes.length === allRuntimeKeys.length;
      const isEnabled = enabledRuntimes.includes(runtime);
      if (isAllEnabled) {
        setEnabledRuntimes([runtime]);
        return;
      }
      if (!isEnabled) {
        setEnabledRuntimes([...enabledRuntimes, runtime]);
        return;
      }
      setEnabledRuntimes(allRuntimeKeys);
    },
    [enabledRuntimes]
  );

  // Returns fresh [start, end] for relative ranges, stored values for custom/absolute.
  // Call this at the moment of any data fetch so relative ranges always slide to now.
  const getEffectiveTimes = useCallback((): [string, string] => {
    if (timeRange !== "custom") {
      const now = new Date();
      return [
        formatDateTimeLocal(new Date(now.getTime() - parseRangeToMs(timeRange))),
        formatDateTimeLocal(now),
      ];
    }
    return [appliedStartTime, appliedEndTime];
  }, [timeRange, appliedStartTime, appliedEndTime]);

  const downloadBlob = useCallback((blob: Blob, filename: string) => {
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, []);

  const getSvgElement = useCallback((): SVGSVGElement | null => {
    const container = document.getElementById("flamegraph");
    return container?.querySelector("svg") ?? null;
  }, []);

  const exportFilename = useCallback(
    (ext: string) => {
      const [start] = getEffectiveTimes();
      const ts = start.replace(/[:\s]/g, "-").replace(/[^a-zA-Z0-9\-_]/g, "");
      const svc = selectedService || "flamegraph";
      return `${svc}_${ts}.${ext}`;
    },
    [getEffectiveTimes, selectedService]
  );

  const handleExportPng = useCallback(() => {
    const svg = getSvgElement();
    if (!svg) return;
    const clone = svg.cloneNode(true) as SVGSVGElement;
    const { width, height } = svg.getBoundingClientRect();
    clone.setAttribute("width", String(width));
    clone.setAttribute("height", String(height));
    const styleProps = ["fill", "stroke", "stroke-width", "font-size", "font-family", "opacity", "rx", "ry"];
    const originals = svg.querySelectorAll("*");
    const clones = clone.querySelectorAll("*");
    for (let i = 0; i < originals.length; i++) {
      const cs = window.getComputedStyle(originals[i]);
      const el = clones[i] as SVGElement | HTMLElement;
      for (const prop of styleProps) {
        const val = cs.getPropertyValue(prop);
        if (val) el.style.setProperty(prop, val);
      }
    }
    const serializer = new XMLSerializer();
    const svgString = serializer.serializeToString(clone);
    const blob = new Blob([svgString], { type: "image/svg+xml;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const img = new Image();
    img.onload = () => {
      const canvas = document.createElement("canvas");
      const scale = 2;
      canvas.width = width * scale;
      canvas.height = height * scale;
      const ctx = canvas.getContext("2d");
      if (!ctx) { URL.revokeObjectURL(url); return; }
      ctx.scale(scale, scale);
      ctx.fillStyle = "#ffffff";
      ctx.fillRect(0, 0, width, height);
      ctx.drawImage(img, 0, 0, width, height);
      URL.revokeObjectURL(url);
      canvas.toBlob((pngBlob) => {
        if (pngBlob) downloadBlob(pngBlob, exportFilename("png"));
      }, "image/png");
    };
    img.src = url;
  }, [getSvgElement, downloadBlob, exportFilename]);

  const handleExportSvg = useCallback(() => {
    const [start, end] = getEffectiveTimes();
    const url = new URL(API.flamegraph, window.location.origin);
    url.searchParams.set("service", selectedService);
    url.searchParams.set("start_datetime", formatForApi(start));
    url.searchParams.set("end_datetime", formatForApi(end));
    applyFiltersToUrl(url);
    url.searchParams.set("format", "svg");
    const a = document.createElement("a");
    a.href = url.toString();
    a.download = exportFilename("svg");
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }, [getEffectiveTimes, selectedService, applyFiltersToUrl, exportFilename]);

  const handleExportCollapsed = useCallback(() => {
    const [start, end] = getEffectiveTimes();
    const url = new URL(API.flamegraph, window.location.origin);
    url.searchParams.set("service", selectedService);
    url.searchParams.set("start_datetime", formatForApi(start));
    url.searchParams.set("end_datetime", formatForApi(end));
    applyFiltersToUrl(url);
    url.searchParams.set("format", "collapsed_file");
    const a = document.createElement("a");
    a.href = url.toString();
    a.download = exportFilename("collapsed");
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }, [getEffectiveTimes, selectedService, applyFiltersToUrl, exportFilename]);

  const applyTimeSelection = useCallback(
    (nextStart?: string, nextEnd?: string, nextRange?: string) => {
      const computedStart = nextStart ?? startTime;
      const computedEnd = nextEnd ?? endTime;
      const computedRange = nextRange ?? "custom";
      if (nextStart) {
        setStartTime(nextStart);
      }
      if (nextEnd) {
        setEndTime(nextEnd);
      }
      setTimeRange(computedRange);
      setAppliedStartTime(computedStart);
      setAppliedEndTime(computedEnd);
      setSamplesResetKey((prev) => prev + 1);
      syncTimeToUrl(computedRange, computedStart, computedEnd);
    },
    [startTime, endTime]
  );

  useEffect(() => {
    const [start, end] = getEffectiveTimes();
    loadServices(start, end);
  }, [loadServices, appliedStartTime, appliedEndTime, getEffectiveTimes]);

  useEffect(() => {
    if (!selectedService && serviceOptions.length > 0) {
      setSelectedService(serviceOptions[0].value);
    }
  }, [
    selectedService,
    serviceOptions,
    appliedStartTime,
    appliedEndTime
  ]);

  useEffect(() => {
    if (!selectedService) {
      return;
    }
    const [start, end] = getEffectiveTimes();
    loadFlamegraph(selectedService, start, end);
    loadFilterOptions(selectedService, start, end);
    loadSampleCounts(selectedService, start, end);
    loadSummaryStats(selectedService, start, end);
    if (graphView !== "samples") {
      loadMetricsGraph(selectedService, start, end);
    }
  }, [
    selectedFilters,
    selectedService,
    appliedStartTime,
    appliedEndTime,
    graphView,
    getEffectiveTimes,
    loadFlamegraph,
    loadFilterOptions,
    loadSampleCounts,
    loadSummaryStats,
    loadMetricsGraph
  ]);

  useEffect(() => {
    if (!displayGraphData) {
      return;
    }
    renderFlamegraph(displayGraphData);

    let raf = 0;
    const handleResize = () => {
      cancelAnimationFrame(raf);
      raf = requestAnimationFrame(() => {
        if (displayGraphData) {
          renderFlamegraph(displayGraphData);
        }
      });
    };
    window.addEventListener("resize", handleResize);
    return () => {
      window.removeEventListener("resize", handleResize);
      cancelAnimationFrame(raf);
    };
  }, [displayGraphData]);


  return (
    <div className="shell">
      <Sidebar />

      <main className="main">
        <TopBar
          loadingServices={loadingServices}
          loadingGraph={loadingGraph}
          loadingFilters={loadingFilters}
          serviceOptions={serviceOptions}
          selectedService={selectedService}
          filterOptions={filterOptions}
          selectedFilters={selectedFilters}
          onFiltersChange={setSelectedFilters}
          onResetFilters={() => setSelectedFilters([])}
          searchValue={searchValue}
          onSearchChange={setSearchValue}
          onSearchReset={() => {
            setSearchValue("");
            applyFlamegraphSearch("");
          }}
          onSearchSubmit={(value) => applyFlamegraphSearch(value.trim())}
          graphView={graphView}
          onGraphViewChange={setGraphView}
          rootFrameOptions={rootFrameOptions}
          selectedRootFrame={selectedRootFrame}
          onRootFrameChange={setSelectedRootFrame}
          onServiceChange={(value) => {
            setSelectedService(value);
          }}
          onRefreshServices={() => { const [s, e] = getEffectiveTimes(); loadServices(s, e); }}
          onRefreshFlamegraph={() => {
            const [s, e] = getEffectiveTimes();
            loadFlamegraph(selectedService, s, e);
            loadFilterOptions(selectedService, s, e);
            loadSampleCounts(selectedService, s, e);
            loadSummaryStats(selectedService, s, e);
            if (graphView !== "samples") {
              loadMetricsGraph(selectedService, s, e);
            }
          }}
          timeRanges={timeRanges}
          timeRange={timeRange}
          timePickerOpen={timePickerOpen}
          onToggleTimePicker={() => setTimePickerOpen((prev) => !prev)}
          onCloseTimePicker={() => setTimePickerOpen(false)}
          onTimeRangeChange={setTimeRange}
          startTime={startTime}
          endTime={endTime}
          onStartTimeChange={(value) => {
            setStartTime(value);
          }}
          onEndTimeChange={(value) => {
            setEndTime(value);
          }}
          onApplyTimeSelection={applyTimeSelection}
        />

        {graphView === "samples" ? (
          <SamplesTimeline
            bins={sampleBins}
            loading={loadingSamples}
            resetKey={samplesResetKey}
            onSelectRange={(start, end) => {
              applyTimeSelection(formatDateTimeLocal(start), formatDateTimeLocal(end));
              setSamplesResetKey((prev) => prev + 1);
            }}
          />
        ) : (
          <MetricsTimeline
            points={metricsGraph}
            loading={loadingMetrics}
            mode={graphView}
            resetKey={samplesResetKey}
            onSelectRange={(start, end) => {
              applyTimeSelection(formatDateTimeLocal(start), formatDateTimeLocal(end));
              setSamplesResetKey((prev) => prev + 1);
            }}
          />
        )}
        <StatsBar summary={summaryStats} loading={loadingSummary} />

        <section className="graph-card">
          <div className="status-row">
            {loadingServices ? (
              <div className="status-pill">
                <Loader size="sm" />
                <Text size="sm" c="dimmed">
                  Syncing services
                </Text>
              </div>
            ) : null}
            {error ? <Text className="status-text">{error}</Text> : null}
          </div>

          <div className="flamegraph-wrap">
            <div id="flamegraph" className="flamegraph-canvas"></div>
            {!graphData ? (
              <div className="flamegraph-empty">
                Select a service to load its flamegraph.
              </div>
            ) : null}

          </div>
          <div className="legend-row">
            <RuntimeLegend
              items={legendItems}
              enabled={enabledRuntimes}
              onToggle={toggleRuntime}
            />
            <ExportBar
              onExportPng={handleExportPng}
              onExportSvg={handleExportSvg}
              onExportCollapsed={handleExportCollapsed}
              disabled={!graphData}
            />
          </div>
        </section>
      </main>
    </div>
  );
}

export default App;
