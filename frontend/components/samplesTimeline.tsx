import { scaleBand, scaleLinear, timeFormat } from "d3";
import { useEffect, useMemo, useRef, useState } from "react";
import ChartTooltip from "./chartTooltip";

type SampleBin = {
  startTs: number;
  endTs: number;
  centerTs: number;
  value: number;
};

type SamplesTimelineProps = {
  bins: SampleBin[];
  loading: boolean;
  onSelectRange: (start: Date, end: Date) => void;
  resetKey: number;
};

const plotHeight = 44;
const axisHeight = 26;
const minLabelPx = 64;
const chartPadding = 8;
const barMaxWidth = 6;
const barMinWidth = 2;

function SamplesTimeline({
  bins,
  loading,
  onSelectRange,
  resetKey
}: SamplesTimelineProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);
  const [width, setWidth] = useState(0);
  const [selection, setSelection] = useState<{
    startIndex: number;
    endIndex: number;
    active: boolean;
  } | null>(null);
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const [hoverPoint, setHoverPoint] = useState<{ x: number; y: number } | null>(
    null
  );

  useEffect(() => {
    setSelection(null);
  }, [resetKey]);

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

  const innerWidth = useMemo(
    () => Math.max(0, width - chartPadding * 2),
    [width]
  );

  const maxValue = useMemo(
    () => bins.reduce((max, bin) => (bin.value > max ? bin.value : max), 0),
    [bins]
  );

  const indices = useMemo(() => bins.map((_, index) => index), [bins]);

  const xScale = useMemo(() => {
    return scaleBand<number>()
      .domain(indices)
      .range([0, innerWidth])
      .paddingInner(0.2)
      .paddingOuter(0);
  }, [indices, innerWidth]);

  const yScale = useMemo(() => {
    return scaleLinear().domain([0, maxValue || 1]).range([plotHeight, 0]);
  }, [maxValue]);

  const tickIndices = useMemo(() => {
    if (bins.length === 0 || innerWidth === 0) {
      return [];
    }
    const binWidthPx = innerWidth / bins.length;
    let labelStep = Math.max(1, Math.ceil(minLabelPx / binWidthPx));
    const maxLabels = Math.max(1, Math.floor(innerWidth / minLabelPx));
    if (bins.length / labelStep > maxLabels) {
      labelStep = Math.max(1, Math.ceil(bins.length / maxLabels));
    }
    return indices.filter((index) => index % labelStep === 0);
  }, [bins.length, innerWidth, indices]);

  const formatTime = useMemo(() => timeFormat("%H:%M"), []);
  const formatFull = useMemo(() => timeFormat("%b %d, %Y %H:%M"), []);

  const barWidth = useMemo(() => {
    if (bins.length === 0) {
      return barMinWidth;
    }
    const band = xScale.bandwidth();
    return Math.max(barMinWidth, Math.min(barMaxWidth, band * 0.7));
  }, [bins.length, xScale]);

  const binWidth = bins.length > 0 ? innerWidth / bins.length : 0;

  const getIndexFromClientX = (clientX: number) => {
    if (!svgRef.current || bins.length === 0) {
      return 0;
    }
    const rect = svgRef.current.getBoundingClientRect();
    const x = clientX - rect.left - chartPadding;
    const rawIndex = binWidth > 0 ? Math.floor(x / binWidth) : 0;
    return Math.max(0, Math.min(bins.length - 1, rawIndex));
  };

  const startSelection = (clientX: number) => {
    const index = getIndexFromClientX(clientX);
    setSelection({ startIndex: index, endIndex: index, active: true });
  };

  const updateSelection = (clientX: number) => {
    setSelection((prev) => {
      if (!prev || !prev.active) {
        return prev;
      }
      const index = getIndexFromClientX(clientX);
      if (index === prev.endIndex) {
        return prev;
      }
      return { ...prev, endIndex: index };
    });
  };

  const finishSelection = () => {
    setSelection((prev) => {
      if (!prev || !prev.active || bins.length === 0) {
        return prev;
      }
      const minIndex = Math.min(prev.startIndex, prev.endIndex);
      const maxIndex = Math.max(prev.startIndex, prev.endIndex);
      const startTs = bins[minIndex]?.startTs ?? bins[minIndex]?.centerTs;
      const endTs = bins[maxIndex]?.endTs ?? bins[maxIndex]?.centerTs;
      if (startTs && endTs) {
        onSelectRange(new Date(startTs), new Date(endTs));
      }
      return { ...prev, active: false };
    });
  };

  const handleHover = (clientX: number, clientY: number) => {
    if (selection?.active) {
      return;
    }
    const index = getIndexFromClientX(clientX);
    setHoverIndex(index);
    if (svgRef.current) {
      const rect = svgRef.current.getBoundingClientRect();
      setHoverPoint({ x: clientX - rect.left, y: clientY - rect.top });
    }
  };

  const selectionBounds = useMemo(() => {
    if (!selection) {
      return null;
    }
    const minIndex = Math.min(selection.startIndex, selection.endIndex);
    const maxIndex = Math.max(selection.startIndex, selection.endIndex);
    return {
      minIndex,
      maxIndex,
      active: selection.active
    };
  }, [selection]);

  return (
    <div className="samples-card" ref={containerRef}>
      <svg
        className="samples-svg"
        width="100%"
        height={plotHeight + axisHeight}
        role="img"
        aria-label="Samples timeline"
        ref={svgRef}
      >
        <g transform={`translate(${chartPadding}, 0)`}>
          {tickIndices.map((index) => {
            const x = xScale(index);
            if (x === undefined) {
              return null;
            }
            const center = x + xScale.bandwidth() / 2;
            return (
              <line
                key={`grid-${index}`}
                className="samples-gridline"
                x1={center}
                x2={center}
                y1={0}
                y2={plotHeight}
              />
            );
          })}
          {bins.map((bin, index) => {
            const x = xScale(index);
            if (x === undefined) {
              return null;
            }
            const barHeight = Math.max(2, plotHeight - yScale(bin.value));
            const center = x + xScale.bandwidth() / 2;
            const fullLabel = formatFull(new Date(bin.centerTs));
            return (
              <rect
                key={`${bin.centerTs}-${bin.value}`}
                className="sample-bar"
                x={center - barWidth / 2}
                y={plotHeight - barHeight}
                width={barWidth}
                height={barHeight}
                rx={2}
                ry={2}
              >
                <title>{`${bin.value} samples\n${fullLabel}`}</title>
              </rect>
            );
          })}
          {selectionBounds ? (
            <rect
              className="samples-selection"
              x={selectionBounds.minIndex * binWidth}
              y={0}
              width={
                (selectionBounds.maxIndex - selectionBounds.minIndex + 1) *
                binWidth
              }
              height={plotHeight}
              rx={4}
              ry={4}
            ></rect>
          ) : null}
          {tickIndices.map((index) => {
            const x = xScale(index);
            if (x === undefined) {
              return null;
            }
            const center = x + xScale.bandwidth() / 2;
            const bin = bins[index];
            const label = bin ? formatTime(new Date(bin.centerTs)) : "";
            const fullLabel = bin ? formatFull(new Date(bin.centerTs)) : "";
            return (
              <g key={`tick-${index}`} transform={`translate(${center}, 0)`}>
                <line
                  className="samples-tick-mark"
                  x1={0}
                  x2={0}
                  y1={plotHeight}
                  y2={plotHeight + 8}
                />
                <text
                  className="samples-tick-label"
                  x={0}
                  y={plotHeight + axisHeight / 2 + 2}
                  textAnchor="middle"
                  dominantBaseline="middle"
                >
                  {label}
                </text>
                {fullLabel ? <title>{fullLabel}</title> : null}
              </g>
            );
          })}
          <rect
            className="samples-overlay"
            x={0}
            y={0}
            width={innerWidth}
            height={plotHeight}
            onPointerDown={(event) => {
              if (bins.length === 0) {
                return;
              }
              event.currentTarget.setPointerCapture(event.pointerId);
              startSelection(event.clientX);
            }}
            onPointerMove={(event) => {
              if (selection?.active) {
                updateSelection(event.clientX);
                return;
              }
              handleHover(event.clientX, event.clientY);
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
              setHoverIndex(null);
              setHoverPoint(null);
            }}
          >
          </rect>
        </g>
      </svg>
      {hoverIndex !== null && hoverPoint && bins[hoverIndex] ? (
        <ChartTooltip
          x={hoverPoint.x}
          y={hoverPoint.y}
          lines={[
            `${bins[hoverIndex].value} samples`,
            formatFull(new Date(bins[hoverIndex].centerTs))
          ]}
        />
      ) : null}
      {loading && bins.length === 0 ? (
        <div className="samples-loading">Loading samples...</div>
      ) : null}
    </div>
  );
}

export type { SampleBin };
export default SamplesTimeline;
