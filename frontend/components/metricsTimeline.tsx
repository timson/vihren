import { bisector, line, scaleLinear, scaleTime, timeFormat } from "d3";
import { useEffect, useMemo, useRef, useState } from "react";
import ChartTooltip from "./chartTooltip";
import type { MetricsSummary } from "../types";

type MetricsTimelineProps = {
  points: MetricsSummary[];
  loading: boolean;
  mode: "cpu" | "memory";
  onSelectRange: (start: Date, end: Date) => void;
  resetKey: number;
};

const plotHeightCollapsed = 44;
const plotHeightExpanded = 120;
const axisHeight = 26;
const chartPadding = 8;
const yAxisWidth = 36;
const minLabelPx = 64;
const gridLineCount = 4;

function MetricsTimeline({
  points,
  loading,
  mode,
  onSelectRange,
  resetKey
}: MetricsTimelineProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [width, setWidth] = useState(0);
  const [selection, setSelection] = useState<{
    start: Date;
    end: Date;
    active: boolean;
  } | null>(null);
  const [expanded, setExpanded] = useState(false);
  const plotHeight = expanded ? plotHeightExpanded : plotHeightCollapsed;
  const [hovered, setHovered] = useState<{
    x: number;
    y: number;
    value: number;
    time: Date;
  } | null>(null);
  const hoverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const el = containerRef.current;
    if (!el || typeof ResizeObserver === "undefined") {
      return;
    }
    let raf = 0;
    const observer = new ResizeObserver((entries) => {
      if (raf) {
        cancelAnimationFrame(raf);
      }
      raf = requestAnimationFrame(() => {
        const nextWidth = entries[0]?.contentRect?.width ?? 0;
        setWidth(nextWidth);
      });
    });
    observer.observe(el);
    return () => {
      if (raf) {
        cancelAnimationFrame(raf);
      }
      observer.disconnect();
    };
  }, []);

  useEffect(() => {
    setSelection(null);
  }, [resetKey]);

  const leftPadding = expanded ? yAxisWidth : 0;
  const innerWidth = useMemo(
    () => Math.max(0, width - chartPadding * 2 - leftPadding),
    [width, leftPadding]
  );

  const series = useMemo(() => {
    return points
      .filter((point) => point.time)
      .map((point) => ({
        time: new Date(point.time as string),
        value: mode === "cpu" ? point.avg_cpu : point.avg_memory
      }));
  }, [points, mode]);

  const timeExtent = useMemo(() => {
    if (series.length === 0) {
      const now = new Date();
      return [now, now] as [Date, Date];
    }
    return [
      series[0].time,
      series[series.length - 1].time
    ] as [Date, Date];
  }, [series]);

  const xScale = useMemo(() => {
    return scaleTime().domain(timeExtent).range([0, innerWidth]);
  }, [timeExtent, innerWidth]);

  const yScale = useMemo(() => {
    if (series.length === 0) {
      return scaleLinear().domain([0, 1]).range([plotHeight, 0]);
    }
    const minValue = series.reduce(
      (min, point) => (point.value < min ? point.value : min),
      series[0].value
    );
    const maxValue = series.reduce(
      (max, point) => (point.value > max ? point.value : max),
      series[0].value
    );
    const span = Math.max(1, maxValue - minValue);
    const padding = span * 0.08;
    return scaleLinear()
      .domain([minValue - padding, maxValue + padding])
      .range([plotHeight, 0]);
  }, [series, plotHeight]);

  const yGridLines = useMemo(() => {
    const [min, max] = yScale.domain();
    const lines = [];
    for (let i = 1; i < gridLineCount; i++) {
      const value = min + ((max - min) * i) / gridLineCount;
      lines.push({ y: yScale(value), label: `${value.toFixed(0)}%` });
    }
    return lines;
  }, [yScale]);

  const formatTime = useMemo(() => timeFormat("%H:%M"), []);
  const formatFull = useMemo(() => timeFormat("%b %d, %Y %H:%M"), []);

  const ticks = useMemo(() => {
    if (series.length === 0 || innerWidth === 0) {
      return [];
    }
    const maxLabels = Math.max(1, Math.floor(innerWidth / minLabelPx));
    const step = Math.max(1, Math.ceil(series.length / maxLabels));
    return series.filter((_, index) => index % step === 0);
  }, [series, innerWidth]);

  const path = useMemo(() => {
    if (series.length === 0) {
      return "";
    }
    const builder = line<{ time: Date; value: number }>()
      .x((d) => xScale(d.time))
      .y((d) => yScale(d.value));
    return builder(series) ?? "";
  }, [series, xScale, yScale]);

  const getTimeFromClientX = (clientX: number) => {
    if (!svgRef.current) {
      return timeExtent[0];
    }
    const rect = svgRef.current.getBoundingClientRect();
    const x = clientX - rect.left - chartPadding;
    const clamped = Math.max(0, Math.min(innerWidth, x));
    return xScale.invert(clamped);
  };

  const startSelection = (clientX: number) => {
    clearHoverTimer();
    setHovered(null);
    const time = getTimeFromClientX(clientX);
    setSelection({ start: time, end: time, active: true });
  };

  const updateSelection = (clientX: number) => {
    setSelection((prev) => {
      if (!prev || !prev.active) {
        return prev;
      }
      const time = getTimeFromClientX(clientX);
      return { ...prev, end: time };
    });
  };

  const clearHoverTimer = () => {
    if (hoverTimerRef.current) {
      clearTimeout(hoverTimerRef.current);
      hoverTimerRef.current = null;
    }
  };

  const updateHover = (clientX: number, clientY: number) => {
    if (!svgRef.current || series.length === 0) {
      clearHoverTimer();
      setHovered(null);
      return;
    }
    const rect = svgRef.current.getBoundingClientRect();
    const x = clientX - rect.left - chartPadding;
    const clamped = Math.max(0, Math.min(innerWidth, x));
    const time = xScale.invert(clamped);
    const index = bisector((d: { time: Date }) => d.time).left(series, time);
    const prev = series[Math.max(0, index - 1)];
    const next = series[Math.min(series.length - 1, index)];
    const target =
      !prev || !next
        ? prev ?? next
        : time.getTime() - prev.time.getTime() <= next.time.getTime() - time.getTime()
          ? prev
          : next;
    if (!target) {
      clearHoverTimer();
      setHovered(null);
      return;
    }
    clearHoverTimer();
    hoverTimerRef.current = setTimeout(() => {
      setHovered({
        x: chartPadding + xScale(target.time),
        y: Math.min(plotHeight, Math.max(0, clientY - rect.top)),
        value: target.value,
        time: target.time
      });
    }, 1000);
  };

  const finishSelection = () => {
    setSelection((prev) => {
      if (!prev || !prev.active) {
        return prev;
      }
      const start = prev.start < prev.end ? prev.start : prev.end;
      const end = prev.start < prev.end ? prev.end : prev.start;
      onSelectRange(start, end);
      return { ...prev, active: false };
    });
  };

  const selectionBounds = useMemo(() => {
    if (!selection) {
      return null;
    }
    const start = selection.start < selection.end ? selection.start : selection.end;
    const end = selection.start < selection.end ? selection.end : selection.start;
    return {
      startX: xScale(start),
      endX: xScale(end)
    };
  }, [selection, xScale]);

  return (
    <div className="metrics-timeline" ref={containerRef}>
      <svg
        className="samples-svg"
        width="100%"
        height={plotHeight + axisHeight}
        role="img"
        aria-label={`${mode} timeline`}
        ref={svgRef}
      >
        <g transform={`translate(${chartPadding + leftPadding}, 0)`}>
          {yGridLines.map((gl, i) => (
            <g key={i}>
              <line
                className="metrics-gridline"
                x1={0}
                x2={innerWidth}
                y1={gl.y}
                y2={gl.y}
              />
              {expanded ? (
                <text
                  className="metrics-y-label"
                  x={-4}
                  y={gl.y}
                  textAnchor="end"
                  dominantBaseline="middle"
                >
                  {gl.label}
                </text>
              ) : null}
            </g>
          ))}
          {path ? (
            <path className="metrics-line" d={path} fill="none" />
          ) : null}
          {series.map((point, index) => (
            <g key={`${point.time.toISOString()}-${index}`}>
              <circle
                className="metrics-point"
                cx={xScale(point.time)}
                cy={yScale(point.value)}
                r={2}
              />
            </g>
          ))}
          {ticks.map((point, index) => (
            <g key={`${point.time.toISOString()}-${index}`}>
              <line
                className="samples-tick-mark"
                x1={xScale(point.time)}
                x2={xScale(point.time)}
                y1={plotHeight}
                y2={plotHeight + 8}
              />
              <text
                className="samples-tick-label"
                x={xScale(point.time)}
                y={plotHeight + axisHeight / 2 + 2}
                textAnchor="middle"
                dominantBaseline="middle"
              >
                {formatTime(point.time)}
              </text>
            </g>
          ))}
          {selectionBounds ? (
            <rect
              className="samples-selection"
              x={Math.min(selectionBounds.startX, selectionBounds.endX)}
              y={0}
              width={Math.abs(selectionBounds.endX - selectionBounds.startX)}
              height={plotHeight}
              rx={4}
              ry={4}
            ></rect>
          ) : null}
          <rect
            className="samples-overlay"
            x={0}
            y={0}
            width={innerWidth}
            height={plotHeight}
            onPointerDown={(event) => {
              if (series.length === 0) {
                return;
              }
              event.currentTarget.setPointerCapture(event.pointerId);
              startSelection(event.clientX);
            }}
            onPointerMove={(event) => {
              if (selection?.active) {
                updateSelection(event.clientX);
              } else {
                updateHover(event.clientX, event.clientY);
              }
            }}
            onPointerUp={(event) => {
              if (!selection?.active) {
                return;
              }
              event.currentTarget.releasePointerCapture(event.pointerId);
              updateSelection(event.clientX);
              finishSelection();
            }}
            onPointerLeave={() => {
              clearHoverTimer();
              setHovered(null);
              if (selection?.active) {
                finishSelection();
              }
            }}
          />
        </g>
      </svg>
      {hovered ? (
        <ChartTooltip
          x={hovered.x}
          y={hovered.y}
          lines={[`${hovered.value.toFixed(1)}%`, formatFull(hovered.time)]}
        />
      ) : null}
      {expanded ? (
        <div className="metrics-legend">
          <span className="metrics-legend-dot" />
          {mode === "cpu" ? "CPU Usage" : "Memory Usage"}
        </div>
      ) : null}
      {loading && series.length === 0 ? (
        <div className="samples-loading">Loading metrics...</div>
      ) : null}
      <button
        className="metrics-expand-toggle"
        type="button"
        aria-label={expanded ? "Collapse" : "Expand"}
        onClick={() => setExpanded((prev) => !prev)}
      >
        <i className={`bi bi-chevron-${expanded ? "up" : "down"}`} aria-hidden="true"></i>
      </button>
    </div>
  );
}

export default MetricsTimeline;
