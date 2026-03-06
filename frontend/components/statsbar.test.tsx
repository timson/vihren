import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import StatsBar from "./statsbar";
import type { SummaryStats } from "../types";

describe("StatsBar", () => {
  const summary: SummaryStats = {
    samples: 12345,
    avg_cpu: 45.678,
    max_cpu: 89.1,
    current_cpu: 50.0,
    avg_memory: 30.2,
    max_memory: 75.5,
    current_memory: 40.0,
    nodes: 3,
  };

  it("renders stat labels", () => {
    render(<StatsBar summary={summary} />);
    expect(screen.getByText("Samples collected")).toBeDefined();
    expect(screen.getByText("CPU Utilization")).toBeDefined();
    expect(screen.getByText("Memory utilization")).toBeDefined();
    expect(screen.getByText("Nodes")).toBeDefined();
  });

  it("displays formatted values when data is available", () => {
    render(<StatsBar summary={summary} />);
    // Samples uses compact notation (12345 → "12.3K")
    expect(screen.getByText("12.3K")).toBeDefined();
    // Nodes
    expect(screen.getByText("3")).toBeDefined();
  });

  it("shows dashes when loading", () => {
    render(<StatsBar summary={summary} loading={true} />);
    const dashes = screen.getAllByText("—");
    expect(dashes).toHaveLength(4);
  });

  it("shows dashes when no summary", () => {
    render(<StatsBar />);
    const dashes = screen.getAllByText("—");
    expect(dashes).toHaveLength(4);
  });

  it("formats CPU percentages", () => {
    render(<StatsBar summary={summary} />);
    // Should contain "45.7% avg" somewhere in CPU row
    const cpuValue = screen.getByText(/45\.7%/);
    expect(cpuValue).toBeDefined();
  });
});
