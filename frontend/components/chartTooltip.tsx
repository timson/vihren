type ChartTooltipProps = {
  x: number;
  y: number;
  lines: string[];
};

function ChartTooltip({ x, y, lines }: ChartTooltipProps) {
  return (
    <div className="chart-tooltip" style={{ left: x, top: y }}>
      {lines.map((line, index) => (
        <div
          key={`${line}-${index}`}
          className={index === 0 ? "chart-tooltip-value" : "chart-tooltip-time"}
        >
          {line}
        </div>
      ))}
    </div>
  );
}

export default ChartTooltip;
